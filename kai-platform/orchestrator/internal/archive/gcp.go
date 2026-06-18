package archive

import (
	"context"
	"fmt"
	"os"
)

type GCSStore struct {
	bucket string
}

func NewGCSStore() *GCSStore {
	return &GCSStore{
		bucket: os.Getenv("GCS_BUCKET"),
	}
}

func (s *GCSStore) Save(_ context.Context, runID, stepID string, _ []byte) error {
	return fmt.Errorf("Google Cloud Storage: requires cloud.google.com/go/storage — stub only; bucket=%s object=%s/%s.zip", s.bucket, runID, stepID)
}

func (s *GCSStore) Get(_ context.Context, runID, stepID string) ([]byte, error) {
	return nil, fmt.Errorf("Google Cloud Storage: requires cloud.google.com/go/storage — stub only; bucket=%s object=%s/%s.zip", s.bucket, runID, stepID)
}

func (s *GCSStore) Close() error {
	return nil
}
