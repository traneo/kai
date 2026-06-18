package secrets

import (
	"context"
	"fmt"
	"os"
)

type AzureKeyVault struct {
	vaultURL string
	tenantID string
}

func NewAzureKeyVault() *AzureKeyVault {
	return &AzureKeyVault{
		vaultURL: os.Getenv("AZURE_KEY_VAULT_URL"),
		tenantID: os.Getenv("AZURE_TENANT_ID"),
	}
}

func (m *AzureKeyVault) GetSecret(ctx context.Context, path, key string) (string, error) {
	return "", fmt.Errorf("Azure Key Vault: requires github.com/Azure/azure-sdk-for-go — stub only; vault=%s secret=%s/%s", m.vaultURL, path, key)
}

func (m *AzureKeyVault) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	return nil, fmt.Errorf("Azure Key Vault: requires github.com/Azure/azure-sdk-for-go — stub only; vault=%s path=%s", m.vaultURL, path)
}
