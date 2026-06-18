package secrets

import (
	"context"
	"fmt"
	"os"
)

type GCPSecretManager struct {
	projectID string
}

func NewGCPSecretManager() *GCPSecretManager {
	return &GCPSecretManager{
		projectID: os.Getenv("GCP_PROJECT_ID"),
	}
}

func (m *GCPSecretManager) GetSecret(ctx context.Context, path, key string) (string, error) {
	return "", fmt.Errorf("GCP Secret Manager: requires cloud.google.com/go/secretmanager — stub only; project=%s secret=%s/%s", m.projectID, path, key)
}

func (m *GCPSecretManager) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	return nil, fmt.Errorf("GCP Secret Manager: requires cloud.google.com/go/secretmanager — stub only; project=%s path=%s", m.projectID, path)
}
