package github

import (
	"context"
	"fmt"
)

func (c *Client) GetOpenPullRequests(ctx context.Context, owner string, name string) (int, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=open&per_page=1", owner, name)

	count, err := c.countItems(ctx, endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to count pull requests of %s/%s: %w", owner, name, err)
	}
	return count, nil
}
