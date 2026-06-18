package archive

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Store interface {
	Save(ctx context.Context, runID, stepID string, data []byte) error
	Get(ctx context.Context, runID, stepID string) ([]byte, error)
	Close() error
}

func NewStoreFromEnv(ctx context.Context) Store {
	pluginLoader := NewPluginLoader(DefaultPluginDir())
	if err := pluginLoader.Discover(); err == nil {
		stores := pluginLoader.LoadStores(ctx)
		if len(stores) > 0 {
			log.Printf("archive: using plugin backend")
			return &FallbackStore{stores: stores}
		}
	}

	if os.Getenv("AWS_BUCKET") != "" || os.Getenv("AWS_REGION") != "" {
		log.Print("archive: using AWS S3 (stub)")
		return NewS3Store()
	}

	if os.Getenv("AZURE_STORAGE_ACCOUNT") != "" || os.Getenv("AZURE_STORAGE_CONTAINER") != "" {
		log.Print("archive: using Azure Blob Storage (stub)")
		return NewAzureBlobStore()
	}

	if os.Getenv("GCS_BUCKET") != "" {
		log.Print("archive: using Google Cloud Storage (stub)")
		return NewGCSStore()
	}

	dir := getEnv("ARCHIVE_DIR", filepath.Join("data", "state-archives"))
	log.Printf("archive: using local filesystem at %s", dir)
	return NewLocalFileStore(dir)
}

type FallbackStore struct {
	stores []Store
}

func (f *FallbackStore) Save(ctx context.Context, runID, stepID string, data []byte) error {
	var lastErr error
	for _, s := range f.stores {
		if err := s.Save(ctx, runID, stepID, data); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("archive: all backends failed: %w", lastErr)
}

func (f *FallbackStore) Get(ctx context.Context, runID, stepID string) ([]byte, error) {
	var lastErr error
	for _, s := range f.stores {
		data, err := s.Get(ctx, runID, stepID)
		if err != nil {
			lastErr = err
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive: all backends failed: %w", lastErr)
}

func (f *FallbackStore) Close() error {
	var lastErr error
	for _, s := range f.stores {
		if err := s.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
