package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) GetCommitsLast7Days(ctx context.Context, owner string, name string) (int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, name)

	since := time.Now().UTC().AddDate(0, 0, -7).Format(time.RFC3339)
	total := 0

	for page := 1; ; page++ {
		reqUrl := fmt.Sprintf("%s?since=%s&per_page=100&page=%d", endpoint, url.QueryEscape(since), page)

		req, err := c.createAPIRequest(ctx, reqUrl)
		if err != nil {
			return 0, fmt.Errorf("failed to request commits last 7 days: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to request commits: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
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

func (c *Client) GetLastCommitAt(ctx context.Context, owner string, name string) (*time.Time, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits?per_page=1", owner, name)

	req, err := c.createAPIRequest(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to request last commit at: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request last commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API return status code: %d for last commit of %s of %s",
			resp.StatusCode, owner, name)
	}

	var commits []struct {
		Commit struct {
			Author struct {
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("failed to decode last commit response: %w", err)
	}

	if len(commits) == 0 {
		return nil, nil
	}

	lastCommitAt := commits[0].Commit.Author.Date
	return &lastCommitAt, nil
}
