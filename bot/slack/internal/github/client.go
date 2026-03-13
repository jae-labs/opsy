package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v68/github"
)

type Client struct {
	client *gh.Client
	owner  string
	repo   string
}

func NewClient(token, owner, repo string) *Client {
	client := gh.NewClient(nil).WithAuthToken(token)
	return &Client{client: client, owner: owner, repo: repo}
}

// GetFileContent fetches a file from the default branch.
func (c *Client) GetFileContent(ctx context.Context, path string) ([]byte, string, error) {
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, path, nil)
	if err != nil {
		return nil, "", fmt.Errorf("get contents: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return nil, "", fmt.Errorf("decode content: %w", err)
	}
	return []byte(content), fileContent.GetSHA(), nil
}

// CreateBranchFromMain creates a new branch from the HEAD of main.
func (c *Client) CreateBranchFromMain(ctx context.Context, branchName string) error {
	ref, _, err := c.client.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/main")
	if err != nil {
		return fmt.Errorf("get main ref: %w", err)
	}
	newRef := &gh.Reference{
		Ref:    gh.Ptr("refs/heads/" + branchName),
		Object: &gh.GitObject{SHA: ref.Object.SHA},
	}
	_, _, err = c.client.Git.CreateRef(ctx, c.owner, c.repo, newRef)
	if err != nil {
		// branch already exists — delete it and retry with fresh main HEAD
		_, delErr := c.client.Git.DeleteRef(ctx, c.owner, c.repo, "refs/heads/"+branchName)
		if delErr != nil {
			return fmt.Errorf("create ref: %w (delete old branch also failed: %v)", err, delErr)
		}
		_, _, err = c.client.Git.CreateRef(ctx, c.owner, c.repo, newRef)
		if err != nil {
			return fmt.Errorf("create ref after retry: %w", err)
		}
	}
	return nil
}

// UpdateFile creates or updates a file on a branch.
func (c *Client) UpdateFile(ctx context.Context, branch, path string, content []byte, fileSHA, commitMsg string) error {
	opts := &gh.RepositoryContentFileOptions{
		Message: gh.Ptr(commitMsg),
		Content: content,
		Branch:  gh.Ptr(branch),
		SHA:     gh.Ptr(fileSHA),
		Author: &gh.CommitAuthor{
			Name:  gh.Ptr("Opsy Bot"),
			Email: gh.Ptr("luiz@justanother.engineer"),
		},
	}
	_, _, err := c.client.Repositories.UpdateFile(ctx, c.owner, c.repo, path, opts)
	if err != nil {
		return fmt.Errorf("update file: %w", err)
	}
	return nil
}

// ApprovePR submits an approval review on a pull request.
func (c *Client) ApprovePR(ctx context.Context, prNumber int, body string) error {
	review := &gh.PullRequestReviewRequest{
		Event: gh.Ptr("APPROVE"),
		Body:  gh.Ptr(body),
	}
	_, _, err := c.client.PullRequests.CreateReview(ctx, c.owner, c.repo, prNumber, review)
	if err != nil {
		return fmt.Errorf("approve PR: %w", err)
	}
	return nil
}

// CreatePR opens a pull request.
func (c *Client) CreatePR(ctx context.Context, branch, title, body string) (string, error) {
	pr := &gh.NewPullRequest{
		Title:               gh.Ptr(title),
		Head:                gh.Ptr(branch),
		Base:                gh.Ptr("main"),
		Body:                gh.Ptr(body),
		MaintainerCanModify: gh.Ptr(true),
	}
	created, _, err := c.client.PullRequests.Create(ctx, c.owner, c.repo, pr)
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}
	return created.GetHTMLURL(), nil
}
