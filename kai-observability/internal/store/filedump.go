package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"kaiplatform.com/observability/internal/models"
)

type FileDumpStore struct {
	inner Store
	ch    chan models.LogEntry
	done  chan struct{}
	dir   string
	wg    sync.WaitGroup
}

func NewFileDumpStore(inner Store, dir string, queueSize int) *FileDumpStore {
	if queueSize <= 0 {
		queueSize = 10000
	}
	os.MkdirAll(dir, 0755)
	s := &FileDumpStore{
		inner: inner,
		ch:    make(chan models.LogEntry, queueSize),
		done:  make(chan struct{}),
		dir:   dir,
	}
	s.wg.Add(1)
	go s.loop()
	return s
}

func (s *FileDumpStore) Append(ctx context.Context, entries []models.LogEntry) error {
	for _, e := range entries {
		select {
		case s.ch <- e:
		default:
		}
	}
	return s.inner.Append(ctx, entries)
}

func (s *FileDumpStore) Query(ctx context.Context, filter models.QueryFilter) ([]models.LogEntry, error) {
	return s.inner.Query(ctx, filter)
}

func (s *FileDumpStore) GetByID(ctx context.Context, id string) (*models.LogEntry, error) {
	return s.inner.GetByID(ctx, id)
}

func (s *FileDumpStore) Close() error {
	close(s.done)
	s.wg.Wait()
	return s.inner.Close()
}

func (s *FileDumpStore) loop() {
	defer s.wg.Done()
	var buf []byte
	var f *os.File
	var fDay int

	for {
		select {
		case <-s.done:
			if f != nil {
				f.Close()
			}
			return
		case entry, ok := <-s.ch:
			if !ok {
				continue
			}
			now := time.Now()
			day := now.YearDay()
			if day != fDay || f == nil {
				if f != nil {
					f.Close()
				}
				name := fmt.Sprintf("logs_%s.jsonl", now.Format("2006-01-02"))
				path := filepath.Join(s.dir, name)
				var err error
				f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					log.Printf("filedump: open %s: %v", path, err)
					f = nil
					continue
				}
				fDay = day
			}
			buf, _ = json.Marshal(entry)
			buf = append(buf, '\n')
			if _, err := f.Write(buf); err != nil {
				log.Printf("filedump: write: %v", err)
			}
		}
	}
}
