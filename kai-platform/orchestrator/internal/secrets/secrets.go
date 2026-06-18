package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Manager interface {
	GetSecret(ctx context.Context, path, key string) (string, error)
	GetSecrets(ctx context.Context, path string) (map[string]string, error)
}

type MemoryManager struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func NewMemoryManager() *MemoryManager {
	m := &MemoryManager{secrets: map[string]string{
		"llm/openai-key": os.Getenv("OPENAI_API_KEY"),
		"llm/anthropic":  os.Getenv("ANTHROPIC_API_KEY"),
		"git/token":      os.Getenv("GIT_TOKEN"),
	}}
	return m
}

func (m *MemoryManager) GetSecret(_ context.Context, path, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	full := path + "/" + key
	if v, ok := m.secrets[full]; ok {
		return v, nil
	}
	return "", fmt.Errorf("secret %s not found", full)
}

func (m *MemoryManager) GetSecrets(_ context.Context, path string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range m.secrets {
		if len(k) >= len(path) && k[:len(path)] == path {
			result[k[len(path)+1:]] = v
		}
	}
	return result, nil
}

type VaultConfig struct {
	Addr      string
	Token     string
	MountPath string
	Insecure  bool
}

type VaultManager struct {
	cfg    VaultConfig
	client *http.Client
	cache  sync.Map
}

func NewVaultManager(cfg VaultConfig) *VaultManager {
	return &VaultManager{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func VaultFromEnv() *VaultManager {
	cfg := VaultConfig{
		Addr:      os.Getenv("VAULT_ADDR"),
		Token:     os.Getenv("VAULT_TOKEN"),
		MountPath: os.Getenv("VAULT_MOUNT_PATH"),
		Insecure:  os.Getenv("VAULT_INSECURE") == "true",
	}
	if cfg.Addr == "" {
		return nil
	}
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}
	return NewVaultManager(cfg)
}

func (v *VaultManager) GetSecret(ctx context.Context, path, key string) (string, error) {
	secrets, err := v.GetSecrets(ctx, path)
	if err != nil {
		return "", err
	}
	if val, ok := secrets[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("secret key %q not found in %s", key, path)
}

func (v *VaultManager) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	if cached, ok := v.cache.Load(path); ok {
		return cached.(map[string]string), nil
	}

	url := fmt.Sprintf("%s/v1/%s/data/%s", v.cfg.Addr, v.cfg.MountPath, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("vault request: %w", err)
	}
	req.Header.Set("X-Vault-Token", v.cfg.Token)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault returned %d", resp.StatusCode)
	}

	var vaultResp struct {
		Data struct {
			Data map[string]string `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&vaultResp); err != nil {
		return nil, fmt.Errorf("vault decode: %w", err)
	}

	v.cache.Store(path, vaultResp.Data.Data)
	return vaultResp.Data.Data, nil
}

func NewManagerFromEnv(ctx context.Context) Manager {
	if vault := VaultFromEnv(); vault != nil {
		log.Printf("secrets: using vault at %s", vault.cfg.Addr)
		return vault
	}

	pluginLoader := NewPluginLoader(DefaultPluginDir())
	if err := pluginLoader.Discover(); err == nil {
		pluginMgrs := pluginLoader.LoadManagers(ctx)
		if len(pluginMgrs) > 0 {
			return &FallbackManager{managers: pluginMgrs}
		}
	}

	if os.Getenv("AWS_REGION") != "" || os.Getenv("AWS_PROFILE") != "" {
		log.Print("secrets: using AWS Secrets Manager (stub)")
		return NewAWSSecretsManager()
	}

	if os.Getenv("AZURE_KEY_VAULT_URL") != "" {
		log.Print("secrets: using Azure Key Vault (stub)")
		return NewAzureKeyVault()
	}

	if os.Getenv("GCP_PROJECT_ID") != "" {
		log.Print("secrets: using GCP Secret Manager (stub)")
		return NewGCPSecretManager()
	}

	mem := NewMemoryManager()
	log.Print("secrets: using in-memory store")
	return mem
}

type FallbackManager struct {
	managers []Manager
}

func (f *FallbackManager) GetSecret(ctx context.Context, path, key string) (string, error) {
	for _, m := range f.managers {
		val, err := m.GetSecret(ctx, path, key)
		if err == nil {
			return val, nil
		}
	}
	return "", fmt.Errorf("secret %s/%s not found in any plugin backend", path, key)
}

func (f *FallbackManager) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	for _, m := range f.managers {
		vals, err := m.GetSecrets(ctx, path)
		if err == nil && len(vals) > 0 {
			return vals, nil
		}
	}
	return nil, fmt.Errorf("no secrets found at %s in any plugin backend", path)
}

type InjectableSecrets struct {
	LLMEndpoint string
	LLMModel    string
	LLMAPIKey   string
	GitToken    string
}

func Resolve(ctx context.Context, mgr Manager) *InjectableSecrets {
	secrets := &InjectableSecrets{
		LLMEndpoint: os.Getenv("LLM_ENDPOINT"),
		LLMModel:    os.Getenv("LLM_MODEL"),
	}

	if key, err := mgr.GetSecret(ctx, "llm", "openai-key"); err == nil {
		secrets.LLMAPIKey = key
	} else if key, err := mgr.GetSecret(ctx, "llm", "anthropic"); err == nil {
		secrets.LLMAPIKey = key
	}

	if secrets.LLMAPIKey == "" {
		secrets.LLMAPIKey = os.Getenv("LLM_API_KEY")
	}

	if token, err := mgr.GetSecret(ctx, "git", "token"); err == nil {
		secrets.GitToken = token
	} else {
		secrets.GitToken = os.Getenv("GIT_TOKEN")
	}

	return secrets
}
