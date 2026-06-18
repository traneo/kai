package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type Config struct {
	TLSCertFile       string
	TLSKeyFile        string
	TLSCACertFile     string
	PreSharedToken    string
	Insecure          bool
}

type Authenticator struct {
	mu  sync.RWMutex
	cfg Config
}

func New(cfg Config) *Authenticator {
	return &Authenticator{cfg: cfg}
}

func (a *Authenticator) ServerCredentials() (grpc.ServerOption, error) {
	if a.cfg.Insecure {
		return nil, nil
	}

	if a.cfg.TLSCertFile != "" && a.cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(a.cfg.TLSCertFile, a.cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load server cert: %w", err)
		}

		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS13,
		}

		if a.cfg.TLSCACertFile != "" {
			ca, err := os.ReadFile(a.cfg.TLSCACertFile)
			if err != nil {
				return nil, fmt.Errorf("read CA cert: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(ca) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
			tlsCfg.ClientCAs = pool
		}

		return grpc.Creds(credentials.NewTLS(tlsCfg)), nil
	}

	if a.cfg.PreSharedToken != "" {
		return nil, nil
	}

	return grpc.Creds(insecure.NewCredentials()), nil
}

func (a *Authenticator) ClientCredentials() (grpc.DialOption, error) {
	if a.cfg.Insecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	if a.cfg.TLSCertFile != "" && a.cfg.TLSKeyFile != "" && a.cfg.TLSCACertFile != "" {
		cert, err := tls.LoadX509KeyPair(a.cfg.TLSCertFile, a.cfg.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}

		ca, err := os.ReadFile(a.cfg.TLSCACertFile)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      pool,
			MinVersion:   tls.VersionTLS13,
		}

		return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), nil
	}

	if a.cfg.PreSharedToken != "" {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
}

func (a *Authenticator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := a.authenticate(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (a *Authenticator) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := a.authenticate(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}
}

func (a *Authenticator) SetConfig(cfg Config) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg = cfg
}

func (a *Authenticator) authenticate(ctx context.Context) error {
	if a.cfg.Insecure {
		return nil
	}

	if a.cfg.TLSCertFile != "" && a.cfg.TLSCACertFile != "" {
		p, ok := peer.FromContext(ctx)
		if !ok {
			return fmt.Errorf("no peer info")
		}
		tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
		if !ok {
			return fmt.Errorf("no TLS auth info")
		}
		if len(tlsInfo.State.VerifiedChains) == 0 {
			return fmt.Errorf("client certificate not verified")
		}
		return nil
	}

	if a.cfg.PreSharedToken != "" {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return fmt.Errorf("no metadata")
		}
		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return fmt.Errorf("no authorization token")
		}
		expected := "Bearer " + a.cfg.PreSharedToken
		if tokens[0] != expected {
			return fmt.Errorf("invalid token")
		}
		return nil
	}

	return nil
}

func (a *Authenticator) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.cfg.Insecure {
			next.ServeHTTP(w, r)
			return
		}

		if a.cfg.PreSharedToken != "" {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + a.cfg.PreSharedToken
			if auth != expected {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
		}

		if a.cfg.TLSCertFile != "" && a.cfg.TLSCACertFile != "" {
			if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
				http.Error(w, `{"error":"tls required"}`, http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) DialOption(agentAddr string) grpc.DialOption {
	if a.cfg.Insecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	if a.cfg.PreSharedToken != "" {
		return grpc.WithPerRPCCredentials(&tokenCredential{token: a.cfg.PreSharedToken})
	}

	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

type tokenCredential struct {
	token string
}

func (t *tokenCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t *tokenCredential) RequireTransportSecurity() bool {
	return false
}

func TokenFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return ""
	}
	return strings.TrimPrefix(tokens[0], "Bearer ")
}

type HTTPConfig struct {
	Token string
}

func HTTPTokenMiddleware(token *string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := *token
			if t == "" {
				next.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				auth := r.Header.Get("Authorization")
				if auth != "Bearer "+t {
					http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func LoadConfig() Config {
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	caFile := os.Getenv("TLS_CA_FILE")
	token := os.Getenv("AUTH_TOKEN")
	insecure := os.Getenv("INSECURE") == "true" || (certFile == "" && token == "")

	if insecure {
		log.Print("auth: running in insecure mode")
	}

	return Config{
		TLSCertFile:    certFile,
		TLSKeyFile:     keyFile,
		TLSCACertFile:  caFile,
		PreSharedToken: token,
		Insecure:       insecure,
	}
}
