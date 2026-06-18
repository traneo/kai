package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kaiplatform/config/internal/config"
)

type Store struct {
	mu              sync.RWMutex
	dir             string
	orchestratorURL string
	httpClient      *http.Client
}

func New(dir, orchestratorURL string) (*Store, error) {
	s := &Store{
		dir:             dir,
		orchestratorURL: strings.TrimRight(orchestratorURL, "/"),
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
	if err := os.MkdirAll(filepath.Join(dir, "versions"), 0755); err != nil {
		return nil, fmt.Errorf("create versions dir: %w", err)
	}
	initCounter(dir)
	initActive(dir)

	active, err := s.getActiveVersion()
	if err != nil {
		v, cerr := s.createDraft(json.RawMessage(`{"runner":"kai"}`), "initial config", "system")
		if cerr != nil {
			return nil, fmt.Errorf("create initial config: %w", cerr)
		}
		if _, cerr := s.doPublish(v.ID); cerr != nil {
			return nil, fmt.Errorf("publish initial config: %w", cerr)
		}
		if _, cerr := s.doActivate(v.ID); cerr != nil {
			return nil, fmt.Errorf("activate initial config: %w", cerr)
		}
	} else {
		_ = active
	}
	return s, nil
}

func (s *Store) ListVersions(status string) ([]*config.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(s.dir, "versions"))
	if err != nil {
		return nil, err
	}

	var versions []*config.ConfigVersion
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		v, err := readVersion(s.dir, strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		if status != "" && v.Status != status {
			continue
		}
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (s *Store) GetVersion(id string) (*config.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return readVersion(s.dir, id)
}

func (s *Store) GetActiveVersion() (*config.ConfigVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getActiveVersion()
}

func (s *Store) CreateDraft() (*config.ConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, err := s.getActiveVersion()
	if err != nil {
		return s.createDraft(json.RawMessage(`{"runner":"kai"}`), "", "")
	}

	// Clone the active config as the base for the new draft
	cfg := make(json.RawMessage, len(active.Config))
	copy(cfg, active.Config)
	return s.createDraft(cfg, "", "")
}

func (s *Store) UpdateDraft(id string, cfg json.RawMessage, message string) (*config.ConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, err := readVersion(s.dir, id)
	if err != nil {
		return nil, err
	}
	if v.Status != "draft" {
		return nil, fmt.Errorf("version %s is %s, not draft", id, v.Status)
	}

	v.Config = cfg
	v.Message = message
	v.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := writeVersion(s.dir, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) Publish(id string) (*config.ConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doPublish(id)
}

func (s *Store) Activate(id string) (*config.ReloadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, err := readVersion(s.dir, id)
	if err != nil {
		return nil, err
	}
	if v.Status == "draft" {
		return nil, fmt.Errorf("cannot activate draft, publish first")
	}

	prevActive, _ := s.getActiveVersion()

	writeActive(s.dir, v.ID)

	result, err := s.pushToOrchestrator(v)

	if err != nil {
		if prevActive != nil {
			writeActive(s.dir, prevActive.ID)
		}
		return result, fmt.Errorf("push to orchestrator: %w", err)
	}

	v.Status = "active"
	v.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	writeVersion(s.dir, v)

	if prevActive != nil && prevActive.ID != v.ID {
		prevActive.Status = "published"
		prevActive.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		writeVersion(s.dir, prevActive)
	}

	return result, nil
}

func (s *Store) Rollback(id string) (*config.ConfigVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	target, err := readVersion(s.dir, id)
	if err != nil {
		return nil, err
	}

	// Clone target config for draft
	cfg := make(json.RawMessage, len(target.Config))
	copy(cfg, target.Config)
	v, err := s.createDraft(cfg, fmt.Sprintf("rollback to v%d", target.Version), "admin")
	if err != nil {
		return nil, err
	}

	if _, err := s.doPublish(v.ID); err != nil {
		return nil, err
	}

	if _, err := s.doActivate(v.ID); err != nil {
		return nil, err
	}

	return readVersion(s.dir, v.ID)
}

func (s *Store) createDraft(cfg json.RawMessage, message, createdBy string) (*config.ConfigVersion, error) {
	next := nextVersion(s.dir)
	id := fmt.Sprintf("%010d", next)

	v := &config.ConfigVersion{
		ID:        id,
		Version:   next,
		Status:    "draft",
		Config:    cfg,
		Message:   message,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := writeVersion(s.dir, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) doPublish(id string) (*config.ConfigVersion, error) {
	v, err := readVersion(s.dir, id)
	if err != nil {
		return nil, err
	}
	if v.Status != "draft" {
		return nil, fmt.Errorf("version %s is %s, not draft", id, v.Status)
	}

	v.Status = "published"
	v.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := writeVersion(s.dir, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) doActivate(id string) (*config.ReloadResult, error) {
	v, err := readVersion(s.dir, id)
	if err != nil {
		return nil, err
	}
	if v.Status == "draft" {
		return nil, fmt.Errorf("cannot activate draft version %s, publish first", id)
	}

	writeActive(s.dir, v.ID)

	v.Status = "active"
	v.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	writeVersion(s.dir, v)

	return &config.ReloadResult{Status: "activated"}, nil
}

func (s *Store) pushToOrchestrator(v *config.ConfigVersion) (*config.ReloadResult, error) {
	if s.orchestratorURL == "" {
		return &config.ReloadResult{Status: "no_orchestrator"}, nil
	}

	body, err := json.Marshal(v.Config)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Post(
		s.orchestratorURL+"/api/platform/config/reload",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return &config.ReloadResult{
			Status: "push_failed",
			Errors: []string{fmt.Sprintf("orchestrator unreachable: %v", err)},
		}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result config.ReloadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return &config.ReloadResult{Status: "pushed", HotReloaded: []string{"config_sent"}}, nil
	}

	return &result, nil
}

func (s *Store) getActiveVersion() (*config.ConfigVersion, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, "active"))
	if err != nil {
		return nil, err
	}
	return readVersion(s.dir, strings.TrimSpace(string(data)))
}

func readVersion(dir, id string) (*config.ConfigVersion, error) {
	data, err := os.ReadFile(filepath.Join(dir, "versions", id+".json"))
	if err != nil {
		return nil, err
	}
	var v config.ConfigVersion
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func writeVersion(dir string, v *config.ConfigVersion) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "versions", v.ID+".json"), data, 0644)
}

func writeActive(dir, id string) {
	os.WriteFile(filepath.Join(dir, "active"), []byte(id), 0644)
}

func nextVersion(dir string) int {
	data, err := os.ReadFile(filepath.Join(dir, "counter"))
	if err != nil {
		os.WriteFile(filepath.Join(dir, "counter"), []byte("1"), 0644)
		return 1
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	n++
	os.WriteFile(filepath.Join(dir, "counter"), []byte(strconv.Itoa(n)), 0644)
	return n
}

func initCounter(dir string) {
	os.ReadFile(filepath.Join(dir, "counter"))
}

func initActive(dir string) {
	os.ReadFile(filepath.Join(dir, "active"))
}
