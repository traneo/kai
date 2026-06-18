package api

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var (
	debugFile   *os.File
	debugMu     sync.Mutex
	debugInit   sync.Once
	debugLogger *log.Logger
)

func debugLogf(format string, args ...any) {
	debugInit.Do(func() {
		path := os.Getenv("kai_DEBUG_LOG")
		if path == "" {
			return
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("debug log: open %s: %v", path, err)
			return
		}
		debugFile = f
		debugLogger = log.New(f, "", log.Ltime|log.Lmicroseconds)
		fmt.Fprintf(f, "=== debug log started %s ===\n", time.Now().UTC().Format(time.RFC3339))
	})
	if debugLogger == nil {
		return
	}
	debugMu.Lock()
	defer debugMu.Unlock()
	debugLogger.Printf(format, args...)
}
