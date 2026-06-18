package gitprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type bitbucketProvider struct{}

func NewBitbucket() Provider {
	return &bitbucketProvider{}
}

func (p *bitbucketProvider) Name() string { return "bitbucket" }

func (p *bitbucketProvider) CreatePR(ctx context.Context, req CreatePRRequest) (*CreatePRResult, error) {
	apiURL := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/pullrequests", req.Owner, req.Repo)

	body := map[string]any{
		"title": req.Title,
		"description": req.Body,
		"source": map[string]any{
			"branch": map[string]string{"name": req.HeadBranch},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": req.BaseBranch},
		},
	}
	data, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("bitbucket create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bitbucket api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("bitbucket api returned %d", resp.StatusCode)
	}

	var result struct {
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bitbucket decode response: %w", err)
	}

	return &CreatePRResult{URL: result.Links.HTML.Href, Number: result.ID}, nil
}
