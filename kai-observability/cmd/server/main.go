package main

import (
	"flag"
	"log"
	"os"

	"kaiplatform.com/observability/internal/api"
	"kaiplatform.com/observability/internal/store"
)

func main() {
	port := flag.String("port", getEnv("PORT", "8082"), "HTTP port")
	pgConn := flag.String("pg", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	fileDump := flag.Bool("file-dump", os.Getenv("OBSERVABILITY_FILE_DUMP") == "true", "enable file dump")
	fileDumpDir := flag.String("file-dump-dir", getEnv("OBSERVABILITY_FILE_DUMP_PATH", "tmp/observability-logs"), "file dump directory")
	flag.Parse()

	var s store.Store

	if *pgConn != "" {
		pg, err := store.NewPostgresStore(*pgConn)
		if err != nil {
			log.Fatalf("postgres store: %v", err)
		}
		s = pg
		log.Print("store: postgres")
	} else {
		s = store.NewMemoryStore(50000)
		log.Print("store: in-memory")
	}

	if *fileDump {
		s = store.NewFileDumpStore(s, *fileDumpDir, 10000)
		log.Printf("file dump enabled: %s", *fileDumpDir)
	}

	hub := api.NewSSEHub()
	server := api.NewServer(*port, s, hub)
	server.Start()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
