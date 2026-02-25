package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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

	// context.Background() is appropriate here: StaticTokenSource doesn't make
	// network calls, so the context is only used by the oauth2 HTTP transport
	// for per-request contexts (which we supply via the fetch methods).
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	tc.Timeout = 30 * time.Second

	return &Client{
		gh: github.NewClient(tc),
	}, nil
}

// RateInfo holds GitHub API rate limit state without leaking the go-github dependency.
type RateInfo struct {
	Remaining int
	Limit     int
	Reset     time.Time
}

// LogRate logs the current rate limit status. Call with the RateInfo returned
// by fetch functions instead of making a separate API call.
func (c *Client) LogRate(r RateInfo) {
	log.Printf("GitHub API rate: %d/%d remaining, resets at %s", r.Remaining, r.Limit, r.Reset.Format(time.RFC3339))
	if r.Remaining < 100 {
		log.Printf("WARNING: GitHub API rate limit nearly exhausted (%d remaining)", r.Remaining)
	}
}

// rateFromResponse extracts RateInfo from a GitHub API response.
func rateFromResponse(resp *github.Response) RateInfo {
	return RateInfo{
		Remaining: resp.Rate.Remaining,
		Limit:     resp.Rate.Limit,
		Reset:     resp.Rate.Reset.Time,
	}
}
