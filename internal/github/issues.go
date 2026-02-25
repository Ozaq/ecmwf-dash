package github

import (
	"context"
	"fmt"
	"log"

	gh "github.com/google/go-github/v66/github"
	"github.com/ozaq/ecmwf-dash/internal/config"
)

func (c *Client) FetchIssues(ctx context.Context, org string, repos []config.RepositoryConfig) IssuesFetchResult {
	var result IssuesFetchResult
	successCount := 0

	for _, repo := range repos {
		if ctx.Err() != nil {
			break
		}

		opts := &gh.IssueListByRepoOptions{
			State: "open",
			ListOptions: gh.ListOptions{
				PerPage: 100,
			},
		}

		repoFailed := false
		for {
			if ctx.Err() != nil {
				break
			}

			issues, resp, err := c.gh.Issues.ListByRepo(ctx, org, repo.Name, opts)
			if err != nil {
				log.Printf("Error fetching issues for %s/%s: %v", org, repo.Name, err)
				repoFailed = true
				break
			}
			if resp != nil {
				result.Rate = rateFromResponse(resp)
			}

			for _, ghIssue := range issues {
				// Skip pull requests (they show up in issues API)
				if ghIssue.PullRequestLinks != nil {
					continue
				}

				issue := Issue{
					Repository:   repo.Name,
					Number:       ghIssue.GetNumber(),
					Title:        ghIssue.GetTitle(),
					URL:          ghIssue.GetHTMLURL(),
					Author:       ghIssue.GetUser().GetLogin(),
					AuthorAvatar: ghIssue.GetUser().GetAvatarURL(),
					CreatedAt:    ghIssue.GetCreatedAt().Time,
					UpdatedAt:    ghIssue.GetUpdatedAt().Time,
				}

				// Set author association and external flag
				if ghIssue.AuthorAssociation != nil {
					issue.AuthorAssociation = ghIssue.GetAuthorAssociation()
					issue.IsExternal = !isInternal(issue.AuthorAssociation)
				}

				// Convert labels
				for _, label := range ghIssue.Labels {
					color := sanitizeLabelColor(label.GetColor())
					issue.Labels = append(issue.Labels, Label{
						Name:       label.GetName(),
						Color:      color,
						LabelStyle: computeLabelStyle(color),
					})
				}

				result.Issues = append(result.Issues, issue)
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		if repoFailed {
			result.FailedRepos = append(result.FailedRepos, repo.Name)
		} else {
			result.SucceededRepos = append(result.SucceededRepos, repo.Name)
			successCount++
		}
	}

	// Any repos not attempted (due to context cancellation) count as failed
	// so the fetcher preserves their old data via Merge.
	attempted := len(result.SucceededRepos) + len(result.FailedRepos)
	for i := attempted; i < len(repos); i++ {
		result.FailedRepos = append(result.FailedRepos, repos[i].Name)
	}

	if successCount == 0 && len(repos) > 0 {
		if ctx.Err() != nil {
			result.Err = ctx.Err()
		} else {
			result.Err = fmt.Errorf("all %d repos failed", len(repos))
		}
	}

	return result
}
