package github

import (
	"context"

	"github.com/google/go-github/v58/github"
)

type Client struct {
	client *github.Client
	ctx    context.Context
	token  string
}

func NewClient(token string) *Client {
	ctx := context.Background()
	var client *github.Client

	if token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	} else {
		client = github.NewClient(nil)
	}

	return &Client{
		client: client,
		ctx:    ctx,
		token:  token,
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
	c.client = github.NewClient(nil).WithAuthToken(token)
}

func (c *Client) GetClient() *github.Client {
	return c.client
}

func (c *Client) GetContext() context.Context {
	return c.ctx
}
