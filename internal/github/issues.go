package github

import (
    "context"
    "fmt"

    "github.com/google/go-github/v66/github"
)

func (c *Client) FetchIssues(ctx context.Context, org string, repos []string) ([]Issue, error) {
    var allIssues []Issue

    for _, repo := range repos {
        opts := &github.IssueListByRepoOptions{
            State: "open",
            ListOptions: github.ListOptions{
                PerPage: 100,
            },
        }

        for {
            issues, resp, err := c.gh.Issues.ListByRepo(ctx, org, repo, opts)
            if err != nil {
                return nil, fmt.Errorf("fetching issues for %s/%s: %w", org, repo, err)
            }

            for _, ghIssue := range issues {
                // Skip pull requests (they show up in issues API)
                if ghIssue.PullRequestLinks != nil {
                    continue
                }

                issue := Issue{
                    Repository:   repo,
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
                    issue.Labels = append(issue.Labels, Label{
                        Name:  label.GetName(),
                        Color: label.GetColor(),
                    })
                }

                allIssues = append(allIssues, issue)
            }

            if resp.NextPage == 0 {
                break
            }
            opts.Page = resp.NextPage
        }
    }

    return allIssues, nil
}

func isInternal(association string) bool {
    return association == "OWNER" || association == "MEMBER" || association == "COLLABORATOR"
}
