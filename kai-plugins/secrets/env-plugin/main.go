package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <get|list> [flags]\n", os.Args[0])
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = os.Args[1:]

	switch cmd {
	case "get":
		runGet()
	case "list":
		runList()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runGet() {
	secretPath := flag.String("path", "", "secret path")
	secretKey := flag.String("key", "", "secret key")
	flag.Parse()

	if *secretPath == "" || *secretKey == "" {
		fmt.Fprintf(os.Stderr, "--path and --key are required\n")
		os.Exit(1)
	}

	envKey := fmt.Sprintf("KAI_SECRET_%s_%s",
		strings.ToUpper(strings.ReplaceAll(*secretPath, "/", "_")),
		strings.ToUpper(*secretKey))

	val := os.Getenv(envKey)
	if val == "" {
		fmt.Fprintf(os.Stderr, "secret %s/%s not found (env %s not set)\n", *secretPath, *secretKey, envKey)
		os.Exit(1)
	}

	fmt.Print(val)
}

func runList() {
	secretPath := flag.String("path", "", "secret path")
	flag.Parse()

	if *secretPath == "" {
		fmt.Fprintf(os.Stderr, "--path is required\n")
		os.Exit(1)
	}

	prefix := fmt.Sprintf("KAI_SECRET_%s_",
		strings.ToUpper(strings.ReplaceAll(*secretPath, "/", "_")))

	result := make(map[string]string)
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		keyName := strings.TrimPrefix(k, prefix)
		keyName = strings.ToLower(keyName)
		result[keyName] = v
	}

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
		os.Exit(1)
	}
}
