package gitprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type githubProvider struct{}

func NewGitHub() Provider {
	return &githubProvider{}
}

func (p *githubProvider) Name() string { return "github" }

func (p *githubProvider) CreatePR(ctx context.Context, req CreatePRRequest) (*CreatePRResult, error) {
	body := map[string]any{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.HeadBranch,
		"base":  req.BaseBranch,
	}
	data, _ := json.Marshal(body)

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", req.Owner, req.Repo)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("github create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("github api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var result struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("github decode response: %w", err)
	}

	return &CreatePRResult{URL: result.HTMLURL, Number: result.Number}, nil
}
