package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestHTTPTokenMiddleware_NoToken(t *testing.T) {
	handler := HTTPTokenMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no token configured, got %d", w.Code)
	}
}

func TestHTTPTokenMiddleware_ValidToken(t *testing.T) {
	handler := HTTPTokenMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", w.Code)
	}
}

func TestHTTPTokenMiddleware_InvalidToken(t *testing.T) {
	handler := HTTPTokenMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid token, got %d", w.Code)
	}
}

func TestHTTPTokenMiddleware_SkipsNonAPI(t *testing.T) {
	handler := HTTPTokenMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-API path, got %d", w.Code)
	}
}

func TestAuthenticate_TokenSuccess(t *testing.T) {
	a := New(Config{PreSharedToken: "mytoken"})
	md := metadata.New(map[string]string{"authorization": "Bearer mytoken"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := a.authenticate(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAuthenticate_TokenFailure(t *testing.T) {
	a := New(Config{PreSharedToken: "mytoken"})
	md := metadata.New(map[string]string{"authorization": "Bearer wrong"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	err := a.authenticate(ctx)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestAuthenticate_TokenMissing(t *testing.T) {
	a := New(Config{PreSharedToken: "mytoken"})
	ctx := context.Background()
	err := a.authenticate(ctx)
	if err == nil {
		t.Error("expected error for missing token")
	}
}

func TestAuthenticate_Insecure(t *testing.T) {
	a := New(Config{Insecure: true})
	err := a.authenticate(context.Background())
	if err != nil {
		t.Errorf("expected no error in insecure mode, got %v", err)
	}
}

func TestTokenFromContext(t *testing.T) {
	md := metadata.New(map[string]string{"authorization": "Bearer tok123"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	if tok := TokenFromContext(ctx); tok != "tok123" {
		t.Errorf("expected tok123, got %s", tok)
	}
}

func TestTokenFromContext_Empty(t *testing.T) {
	if tok := TokenFromContext(context.Background()); tok != "" {
		t.Errorf("expected empty, got %s", tok)
	}
}

func TestLoadConfig_DefaultsToInsecure(t *testing.T) {
	cfg := LoadConfig()
	if !cfg.Insecure {
		t.Error("expected insecure to be true by default")
	}
}
