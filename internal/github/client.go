package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github-watcher/internal/cache"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	cache      *cache.Cache
	token      string
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: cache.NewCache(60 * time.Second),
		token: token,
	}
}

func (c *Client) createAPIRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-watcher-app")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	stringReq := fmt.Sprintf("%s %s", req.Method, req.URL)
	body, code, inCache := c.cache.Get(cache.URL(stringReq))
	if inCache {
		resp := http.Response{
			StatusCode: code,
			Body:       io.NopCloser(bytes.NewBuffer(body)),
		}
		return &resp, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from GitHub API: %w", err)
	}
	if resp.StatusCode == http.StatusOK {
		c.cache.Set(cache.URL(stringReq), respBody, resp.StatusCode)
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	return resp, nil
}

func lastPage(linkHeader string) (int, bool) {
	parts := strings.Split(linkHeader, ",")

	for _, part := range parts {
		if strings.Contains(part, "rel=\"last\"") {
			if idx := strings.Index(part, ";"); idx != -1 {
				part = part[:idx]
			}
			part = strings.Trim(part, " <>")
			parsed, err := url.Parse(part)
			if err != nil {
				return 0, false
			}
			param := parsed.Query().Get("page")
			res, err := strconv.Atoi(param)
			if err != nil {
				return 0, false
			}
			return res, true
		}
	}
	return 0, false
}

func (c *Client) countItems(ctx context.Context, reqURL string) (int, error) {
	req, err := c.createAPIRequest(ctx, reqURL)
	if err != nil {
		return 0, fmt.Errorf("failed to create count request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code from GitHub API: %d", resp.StatusCode)
	}

	num, ok := lastPage(resp.Header.Get("Link"))
	if ok {
		return num, nil
	}

	var items []struct{}

	err = json.NewDecoder(resp.Body).Decode(&items)
	if err != nil {
		return 0, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}
	return len(items), nil
}
