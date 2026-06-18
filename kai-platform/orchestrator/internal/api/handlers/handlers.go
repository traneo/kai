package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/api/coordinator"
	"kaiplatform.com/orchestrator/internal/cost"
)

type ServerIface interface {
	Pool() *agentpool.Pool
	CostTracker() *cost.Tracker
}

type Deps struct {
	Coordinator  *coordinator.Coordinator
	Server       ServerIface
	SecretStore  coordinator.SecretStore
	AuthToken    *string
	PublishEvent func(any)
	Mu           *sync.RWMutex
	Subscribers  *map[chan any]struct{}
}

func writeError(w http.ResponseWriter, code int, format string, args ...any) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf(format, args...),
	})
}

func newRunID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("run-%s-%d", hex.EncodeToString(b), time.Now().UnixMilli())
}
