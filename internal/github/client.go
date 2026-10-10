package github

import (
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
