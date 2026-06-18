package gitprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type gitlabProvider struct{}

func NewGitLab() Provider {
	return &gitlabProvider{}
}

func (p *gitlabProvider) Name() string { return "gitlab" }

func (p *gitlabProvider) CreatePR(ctx context.Context, req CreatePRRequest) (*CreatePRResult, error) {
	projectID := url.PathEscape(req.Owner + "/" + req.Repo)
	apiURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/merge_requests", projectID)

	body := map[string]any{
		"title":              req.Title,
		"description":        req.Body,
		"source_branch":      req.HeadBranch,
		"target_branch":      req.BaseBranch,
	}
	data, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gitlab create request: %w", err)
	}
	httpReq.Header.Set("PRIVATE-TOKEN", req.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gitlab api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab api returned %d", resp.StatusCode)
	}

	var result struct {
		WebURL string `json:"web_url"`
		IID    int    `json:"iid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gitlab decode response: %w", err)
	}

	return &CreatePRResult{URL: result.WebURL, Number: result.IID}, nil
}
