package secrets

import (
	"context"
	"fmt"
	"os"
)

type AWSSecretsManager struct {
	region  string
	profile string
}

func NewAWSSecretsManager() *AWSSecretsManager {
	return &AWSSecretsManager{
		region:  os.Getenv("AWS_REGION"),
		profile: os.Getenv("AWS_PROFILE"),
	}
}

func (m *AWSSecretsManager) GetSecret(ctx context.Context, path, key string) (string, error) {
	return "", fmt.Errorf("AWS Secrets Manager: requires github.com/aws/aws-sdk-go-v2 — stub only; path=%s key=%s", path, key)
}

func (m *AWSSecretsManager) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	return nil, fmt.Errorf("AWS Secrets Manager: requires github.com/aws/aws-sdk-go-v2 — stub only; path=%s", path)
}
