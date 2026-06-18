package archive

import (
	"context"
	"fmt"
	"os"
)

type S3Store struct {
	bucket string
	region string
}

func NewS3Store() *S3Store {
	return &S3Store{
		bucket: os.Getenv("AWS_BUCKET"),
		region: os.Getenv("AWS_REGION"),
	}
}

func (s *S3Store) Save(_ context.Context, runID, stepID string, _ []byte) error {
	return fmt.Errorf("AWS S3: requires github.com/aws/aws-sdk-go-v2 — stub only; bucket=%s key=%s/%s.zip", s.bucket, runID, stepID)
}

func (s *S3Store) Get(_ context.Context, runID, stepID string) ([]byte, error) {
	return nil, fmt.Errorf("AWS S3: requires github.com/aws/aws-sdk-go-v2 — stub only; bucket=%s key=%s/%s.zip", s.bucket, runID, stepID)
}

func (s *S3Store) Close() error {
	return nil
}
