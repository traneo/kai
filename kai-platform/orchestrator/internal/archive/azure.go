package archive

import (
	"context"
	"fmt"
	"os"
)

type AzureBlobStore struct {
	storageAccount string
	container      string
}

func NewAzureBlobStore() *AzureBlobStore {
	return &AzureBlobStore{
		storageAccount: os.Getenv("AZURE_STORAGE_ACCOUNT"),
		container:      os.Getenv("AZURE_STORAGE_CONTAINER"),
	}
}

func (s *AzureBlobStore) Save(_ context.Context, runID, stepID string, _ []byte) error {
	return fmt.Errorf("Azure Blob Storage: requires github.com/Azure/azure-sdk-for-go — stub only; account=%s container=%s blob=%s/%s.zip", s.storageAccount, s.container, runID, stepID)
}

func (s *AzureBlobStore) Get(_ context.Context, runID, stepID string) ([]byte, error) {
	return nil, fmt.Errorf("Azure Blob Storage: requires github.com/Azure/azure-sdk-for-go — stub only; account=%s container=%s blob=%s/%s.zip", s.storageAccount, s.container, runID, stepID)
}

func (s *AzureBlobStore) Close() error {
	return nil
}
