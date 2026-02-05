package github

import (
	"fmt"
	"strings"
)

type RateLimitError struct {
	Remaining int
	Limit     int
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("GitHub API rate limit exceeded: %d/%d remaining", e.Remaining, e.Limit)
}

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "403")
}

func (c *Client) CheckRateLimit() (*RateLimitStatus, error) {
	ctx := c.GetContext()
	client := c.GetClient()

	rateLimit, _, err := client.RateLimit.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &RateLimitStatus{
		Limit:     rateLimit.Core.Limit,
		Remaining: rateLimit.Core.Remaining,
		Reset:     rateLimit.Core.Reset.Time,
	}, nil
}

type RateLimitStatus struct {
	Limit     int
	Remaining int
	Reset     interface{}
}
