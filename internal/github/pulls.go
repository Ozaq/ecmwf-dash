package github

import (
    "context"
    "fmt"
    "time"

    "github.com/google/go-github/v66/github"
)

func (c *Client) FetchPullRequests(ctx context.Context, org string, repos []string) ([]PullRequest, error) {
    var allPRs []PullRequest

    for _, repo := range repos {
        opts := &github.PullRequestListOptions{
            State: "open",
            ListOptions: github.ListOptions{
                PerPage: 100,
            },
        }

        for {
            prs, resp, err := c.gh.PullRequests.List(ctx, org, repo, opts)
            if err != nil {
                return nil, fmt.Errorf("fetching PRs for %s/%s: %w", org, repo, err)
            }

            for _, ghPR := range prs {
                pr := PullRequest{
                    Repository:   repo,
                    Number:       ghPR.GetNumber(),
                    Title:        ghPR.GetTitle(),
                    URL:          ghPR.GetHTMLURL(),
                    Author:       ghPR.GetUser().GetLogin(),
                    AuthorAvatar: ghPR.GetUser().GetAvatarURL(),
                    CreatedAt:    ghPR.GetCreatedAt().Time,
                    UpdatedAt:    ghPR.GetUpdatedAt().Time,
                    State:        ghPR.GetState(),
                    Draft:        ghPR.GetDraft(),
                    BaseBranch:   ghPR.GetBase().GetRef(),
                    HeadBranch:   ghPR.GetHead().GetRef(),
                    Comments:     ghPR.GetComments(),
                }

                // Set author association
                if ghPR.AuthorAssociation != nil {
                    pr.AuthorAssociation = ghPR.GetAuthorAssociation()
                    pr.IsExternal = !isInternal(pr.AuthorAssociation)
                }

                // Convert labels
                for _, label := range ghPR.Labels {
                    pr.Labels = append(pr.Labels, Label{
                        Name:  label.GetName(),
                        Color: label.GetColor(),
                    })
                }

                // Fetch additional details
                if err := c.fetchPRDetails(ctx, org, repo, ghPR.GetNumber(), &pr); err != nil {
                    // Log but continue
                    fmt.Printf("Error fetching PR details for %s/%s#%d: %v\n", org, repo, ghPR.GetNumber(), err)
                }

                allPRs = append(allPRs, pr)
            }

            if resp.NextPage == 0 {
                break
            }
            opts.Page = resp.NextPage
        }
    }

    return allPRs, nil
}

func (c *Client) fetchPRDetails(ctx context.Context, org, repo string, number int, pr *PullRequest) error {
    // Fetch reviews
    reviews, _, err := c.gh.PullRequests.ListReviews(ctx, org, repo, number, nil)
    if err != nil {
        return fmt.Errorf("fetching reviews: %w", err)
    }

    reviewMap := make(map[string]*Reviewer)
    reviewTimes := make(map[string]time.Time)
    approved := false
    changesRequested := false

    for _, review := range reviews {
        if review.GetState() == "" {
            continue
        }
        
        login := review.GetUser().GetLogin()
        // Keep only the latest review per user
        if _, ok := reviewMap[login]; !ok || review.GetSubmittedAt().Time.After(reviewTimes[login]) {
            reviewMap[login] = &Reviewer{
                Login:  login,
                Avatar: review.GetUser().GetAvatarURL(),
                State:  review.GetState(),
            }
            reviewTimes[login] = review.GetSubmittedAt().Time
        }

        if review.GetState() == "APPROVED" {
            approved = true
        } else if review.GetState() == "CHANGES_REQUESTED" {
            changesRequested = true
        }
    }

    for _, reviewer := range reviewMap {
        pr.Reviewers = append(pr.Reviewers, *reviewer)
    }

    // Set review status
    if changesRequested {
        pr.ReviewStatus = "changes_requested"
    } else if approved {
        pr.ReviewStatus = "approved"
    } else {
        pr.ReviewStatus = "pending"
    }

    // Fetch detailed PR info for mergeable state
    fullPR, _, err := c.gh.PullRequests.Get(ctx, org, repo, number)
    if err != nil {
        return fmt.Errorf("fetching PR details: %w", err)
    }

    pr.MergeableState = fullPR.GetMergeableState()
    pr.ReviewComments = fullPR.GetReviewComments()

    // Fetch check runs
    checkRuns, _, err := c.gh.Checks.ListCheckRunsForRef(ctx, org, repo, fullPR.GetHead().GetSHA(), &github.ListCheckRunsOptions{
        ListOptions: github.ListOptions{PerPage: 100},
    })
    if err != nil {
        return fmt.Errorf("fetching check runs: %w", err)
    }

    for _, check := range checkRuns.CheckRuns {
        // Skip skipped checks
        if check.GetConclusion() == "skipped" {
            continue
        }
        
        pr.Checks = append(pr.Checks, Check{
            Name:       check.GetName(),
            Status:     check.GetStatus(),
            Conclusion: check.GetConclusion(),
            URL:        check.GetHTMLURL(),
        })
        
        // Count checks
        if check.GetStatus() == "in_progress" {
            pr.ChecksRunning++
        } else if check.GetConclusion() == "success" {
            pr.ChecksSuccess++
        } else if check.GetConclusion() == "failure" {
            pr.ChecksFailure++
        }
    }

    return nil
}
