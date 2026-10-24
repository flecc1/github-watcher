package github

import (
	"bytes"
	"context"
	"fmt"
	"github-watcher/internal/cache"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	cache      *cache.Cache
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: cache.NewCache(60 * time.Second),
	}
}

func (c *Client) createAPIRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-watcher-app")
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
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		c.cache.Set(cache.URL(stringReq), respBody, resp.StatusCode)
	}

	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	return resp, nil
}
