package archive

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type LocalFileStore struct {
	baseDir string
}

func NewLocalFileStore(baseDir string) *LocalFileStore {
	return &LocalFileStore{baseDir: baseDir}
}

func (s *LocalFileStore) Save(_ context.Context, runID, stepID string, data []byte) error {
	dir := filepath.Join(s.baseDir, runID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create archive dir %s: %w", dir, err)
	}

	path := filepath.Join(dir, stepID+".zip")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write archive %s: %w", path, err)
	}

	log.Printf("state archive saved: %s (%d bytes)", path, len(data))
	return nil
}

func (s *LocalFileStore) Get(_ context.Context, runID, stepID string) ([]byte, error) {
	path := filepath.Join(s.baseDir, runID, stepID+".zip")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read archive %s: %w", path, err)
	}
	return data, nil
}

func (s *LocalFileStore) Close() error {
	return nil
}
