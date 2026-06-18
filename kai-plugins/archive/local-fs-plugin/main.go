package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <save|get> [flags]\n", os.Args[0])
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = os.Args[1:]

	switch cmd {
	case "save":
		runSave()
	case "get":
		runGet()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func flags() (*string, *string, *string) {
	runID := flag.String("run-id", "", "run identifier")
	stepID := flag.String("step-id", "", "step identifier")
	baseDir := flag.String("base-dir", "/tmp/kai-archives", "base directory for archives")
	flag.Parse()
	return runID, stepID, baseDir
}

func runSave() {
	runID, stepID, baseDir := flags()
	if *runID == "" || *stepID == "" {
		fmt.Fprintf(os.Stderr, "--run-id and --step-id are required\n")
		os.Exit(1)
	}

	dir := filepath.Join(*baseDir, *runID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create dir %s: %v\n", dir, err)
		os.Exit(1)
	}

	path := filepath.Join(dir, *stepID+".zip")
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}

	fmt.Printf("saved %s (%d bytes)\n", path, len(data))
}

func runGet() {
	runID, stepID, baseDir := flags()
	if *runID == "" || *stepID == "" {
		fmt.Fprintf(os.Stderr, "--run-id and --step-id are required\n")
		os.Exit(1)
	}

	path := filepath.Join(*baseDir, *runID, *stepID+".zip")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}

	os.Stdout.Write(data)
}
