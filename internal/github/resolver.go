package github

import (
	"fmt"
	"strings"
)

type ResolvedAction struct {
	SHA     string
	Version string
	Error   error
}

func (c *Client) ResolveAction(owner, repo, ref string) ResolvedAction {
	ctx := c.GetContext()
	client := c.GetClient()

	if strings.HasPrefix(ref, "v") {
		tagRef := fmt.Sprintf("tags/%s", ref)
		gitRef, resp, err := client.Git.GetRef(ctx, owner, repo, tagRef)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				return c.resolveAsBranch(owner, repo, ref)
			}
			return ResolvedAction{Error: err}
		}

		if gitRef.Object != nil && gitRef.Object.SHA != nil {
			return ResolvedAction{
				SHA:     *gitRef.Object.SHA,
				Version: ref,
			}
		}
	}

	return c.resolveAsBranch(owner, repo, ref)
}

func (c *Client) resolveAsBranch(owner, repo, branch string) ResolvedAction {
	ctx := c.GetContext()
	client := c.GetClient()

	commit, _, err := client.Repositories.GetCommit(ctx, owner, repo, branch, nil)
	if err != nil {
		return ResolvedAction{Error: err}
	}

	if commit.SHA != nil {
		return ResolvedAction{
			SHA:     *commit.SHA,
			Version: branch,
		}
	}

	return ResolvedAction{Error: fmt.Errorf("unable to resolve reference")}
}
