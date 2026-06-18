package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type Result struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var (
	conventionalPattern = regexp.MustCompile(`^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)(\(.+\))?!?:\s.+`)
	breakingPattern     = regexp.MustCompile(`BREAKING CHANGE|!:`)
)

func main() {
	repoDir := flag.String("repo-dir", "", "repository directory")
	flag.Parse()

	if *repoDir == "" {
		writeResult(Result{Status: "skipped", Message: "no repo directory provided"})
		return
	}

	gitCmd := exec.Command("git", "log", "--oneline", "-10", "--format=%s")
	gitCmd.Dir = *repoDir
	out, err := gitCmd.CombinedOutput()
	if err != nil {
		writeResult(Result{Status: "skipped", Message: fmt.Sprintf("not a git repo: %s", err)})
		return
	}

	messages := strings.Split(strings.TrimSpace(string(out)), "\n")
	var violations []string

	for _, msg := range messages {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			continue
		}
		if !conventionalPattern.MatchString(msg) {
			violations = append(violations, msg)
		}
	}

	if len(violations) > 0 {
		var b strings.Builder
		b.WriteString(fmt.Sprintf("%d commit(s) do not follow conventional commits format:\n", len(violations)))
		b.WriteString("Expected: type(scope)!: description\n")
		b.WriteString("Types: feat, fix, chore, docs, style, refactor, perf, test, build, ci, revert\n")
		for _, v := range violations {
			b.WriteString(fmt.Sprintf("  - %s\n", truncate(v, 80)))
		}

		hasBreaking := false
		for _, msg := range messages {
			if breakingPattern.MatchString(msg) {
				hasBreaking = true
				break
			}
		}
		if hasBreaking {
			b.WriteString("(breaking changes detected — major version bump implied)\n")
		}

		writeResult(Result{Status: "failed", Message: b.String()})
		return
	}

	writeResult(Result{Status: "passed", Message: "all recent commits follow conventional commits format"})
}

func writeResult(r Result) {
	data, _ := json.Marshal(r)
	fmt.Println(string(data))
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
