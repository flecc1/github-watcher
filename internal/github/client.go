package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type RepositoryInfo struct {
	FullName      string `json:"full_name"`
	Stars         int    `json:"stargazers_count"`
	Forks         int    `json:"forks_count"`
	OpenIssues    int    `json:"open_issues_count"`
	Description   string `json:"description"`
	HasIssues     bool   `json:"has_issues"`
	DefaultBranch string `json:"default_branch"`
}

func (c *Client) GetRepository(ctx context.Context, owner string, name string) (RepositoryInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-watcher-app")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("failed to request GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RepositoryInfo{}, fmt.Errorf("GitHub API return status code: %d for %s/%s",
			resp.StatusCode, owner, name)
	}

	var repoInfo RepositoryInfo

	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return RepositoryInfo{}, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return repoInfo, nil
}

func (c *Client) HasReadme(ctx context.Context, owner string, name string) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create README request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-watcher-app")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to request README: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("GitHub API returned code: %d for README of %s/%s",
			resp.StatusCode, owner, name)
	}
}
