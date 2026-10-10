package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

func (c *Client) GetOpenPullRequests(ctx context.Context, owner string, name string) (int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, name)
	total := 0

	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s?state=open&per_page=100&page=%d", endpoint, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create pull requests request: %w", err)
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "github-watcher-app")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to request pull requests: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return 0, fmt.Errorf("GitHub API return status code: %d for pull requests of %s/%s",
				resp.StatusCode, owner, name)
		}

		var pullRequests []struct{}
		err = json.NewDecoder(resp.Body).Decode(&pullRequests)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("failed to decode pull requests response: %w", err)
		}
		total += len(pullRequests)
		if len(pullRequests) < 100 {
			break
		}
	}
	return total, nil
}

func (c *Client) GetCommitsLast7Days(ctx context.Context, owner string, name string) (int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, name)

	since := time.Now().UTC().AddDate(0, 0, -7).Format(time.RFC3339)
	total := 0

	for page := 1; ; page++ {
		reqUrl := fmt.Sprintf("%s?since=%s&per_page=100&page=%d", endpoint, url.QueryEscape(since), page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
		if err != nil {
			return 0, fmt.Errorf("failed to create commits last 7 days request: %w", err)
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "github-watcher-app")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to request commits: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			req.Body.Close()
			return 0, fmt.Errorf("GitHub API return status code: %d for commits of %s of %s",
				resp.StatusCode, owner, name)
		}

		var commits []struct{}
		err = json.NewDecoder(resp.Body).Decode(&commits)
		resp.Body.Close()
		if err != nil {
			return 0, fmt.Errorf("failed to decode commits response: %w", err)
		}
		total += len(commits)
		if len(commits) < 100 {
			break
		}
	}
	return total, nil
}
