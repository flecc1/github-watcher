package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) GetOpenPullRequests(ctx context.Context, owner string, name string) (int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, name)
	total := 0

	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s?state=open&per_page=100&page=%d", endpoint, page)

		req, err := c.createAPIRequest(ctx, reqURL)
		if err != nil {
			return 0, fmt.Errorf("failed to request GitHub API: %w", err)
		}

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
