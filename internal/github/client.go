package github

import (
    "context"
    "fmt"
    "os"

    "github.com/google/go-github/v66/github"
    "golang.org/x/oauth2"
)

type Client struct {
    gh *github.Client
}

func NewClient() (*Client, error) {
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
    }

    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)

    return &Client{
        gh: github.NewClient(tc),
    }, nil
}
