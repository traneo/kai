package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Result struct {
	URL    string `json:"url"`
	Number int    `json:"number"`
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "create-pr" {
		writeError("usage: forgejo-provider create-pr --flags")
	}

	fs := flag.NewFlagSet("create-pr", flag.ExitOnError)
	repoURL := fs.String("repo-url", "", "repository URL")
	owner := fs.String("owner", "", "repository owner/namespace")
	repo := fs.String("repo", "", "repository name")
	base := fs.String("base", "", "base branch (target)")
	head := fs.String("head", "", "head branch (source)")
	title := fs.String("title", "", "PR title")
	body := fs.String("body", "", "PR description")
	token := fs.String("token", "", "Forgejo access token")
	forgejoURL := fs.String("forgejo_url", "", "Forgejo instance URL (overrides auto-detect)")

	fs.Parse(os.Args[2:])

	if *repoURL == "" || *token == "" || *title == "" {
		writeError("repo-url, token, and title are required")
	}

	forgejoBase := *forgejoURL
	if forgejoBase == "" {
		forgejoBase = detectForgejoURL(*repoURL)
	}
	forgejoBase = strings.TrimRight(forgejoBase, "/")

	r := *repo
	o := *owner
	if o == "" || r == "" {
		parsedOwner, parsedRepo := parseRepoURL(*repoURL)
		if o == "" {
			o = parsedOwner
		}
		if r == "" {
			r = parsedRepo
		}
	}

	apiURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", forgejoBase, o, r)

	prBody := map[string]string{
		"base":  *base,
		"head":  *head,
		"title": *title,
		"body":  *body,
	}
	data, _ := json.Marshal(prBody)

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
	if err != nil {
		writeError("create request: %s", err)
	}
	req.Header.Set("Authorization", "token "+*token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError("api call: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		writeError("api returned %d (expected 201)", resp.StatusCode)
	}

	var forgejoResp struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&forgejoResp); err != nil {
		writeError("decode response: %s", err)
	}

	result, _ := json.Marshal(Result{
		URL:    forgejoResp.HTMLURL,
		Number: forgejoResp.Number,
	})
	fmt.Println(string(result))
}

func detectForgejoURL(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "https://codeberg.org"
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host)
}

func parseRepoURL(rawURL string) (string, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	path := strings.TrimSuffix(u.Path, ".git")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", ""
	}
	repo := parts[len(parts)-1]
	owner := strings.Join(parts[:len(parts)-1], "/")
	return owner, repo
}

func writeError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
