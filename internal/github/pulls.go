package github

import (
	"context"
	"log"
	"time"

	gh "github.com/google/go-github/v66/github"
	"github.com/ozaq/ecmwf-dash/internal/config"
)

func (c *Client) FetchPullRequests(ctx context.Context, org string, repos []config.RepositoryConfig) ([]PullRequest, RateInfo, error) {
	var allPRs []PullRequest
	var lastRate RateInfo
	successCount := 0

	for _, repo := range repos {
		if ctx.Err() != nil {
			break
		}

		opts := &gh.PullRequestListOptions{
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

			prs, resp, err := c.gh.PullRequests.List(ctx, org, repo.Name, opts)
			if err != nil {
				log.Printf("Error fetching PRs for %s/%s: %v", org, repo.Name, err)
				repoFailed = true
				break
			}
			if resp != nil {
				lastRate = rateFromResponse(resp)
			}

			for _, ghPR := range prs {
				pr := PullRequest{
					Repository:   repo.Name,
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
					color := sanitizeLabelColor(label.GetColor())
					pr.Labels = append(pr.Labels, Label{
						Name:       label.GetName(),
						Color:      color,
						LabelStyle: computeLabelStyle(color),
					})
				}

				// Fetch additional details
				rate, err := c.fetchPRDetails(ctx, org, repo.Name, ghPR.GetNumber(), &pr)
				lastRate = rate
				if err != nil {
					log.Printf("Error fetching PR details for %s/%s#%d: %v", org, repo.Name, ghPR.GetNumber(), err)
				}

				allPRs = append(allPRs, pr)
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		if !repoFailed {
			successCount++
		}
	}

	if successCount == 0 && len(repos) > 0 {
		return allPRs, lastRate, ctx.Err()
	}

	return allPRs, lastRate, nil
}

func (c *Client) fetchPRDetails(ctx context.Context, org, repo string, number int, pr *PullRequest) (RateInfo, error) {
	var lastRate RateInfo

	// Fetch reviews with pagination
	reviewOpts := &gh.ListOptions{PerPage: 100}
	reviewMap := make(map[string]*Reviewer)
	reviewTimes := make(map[string]time.Time)

	for {
		if ctx.Err() != nil {
			return lastRate, ctx.Err()
		}

		reviews, resp, err := c.gh.PullRequests.ListReviews(ctx, org, repo, number, reviewOpts)
		if err != nil {
			return lastRate, err
		}
		if resp != nil {
			lastRate = rateFromResponse(resp)
		}

		for _, review := range reviews {
			state := review.GetState()
			if state == "" || state == "COMMENTED" {
				continue
			}

			login := review.GetUser().GetLogin()

			// DISMISSED reviews remove the reviewer's previous state
			if state == "DISMISSED" {
				delete(reviewMap, login)
				delete(reviewTimes, login)
				continue
			}

			if _, ok := reviewMap[login]; !ok || review.GetSubmittedAt().Time.After(reviewTimes[login]) {
				reviewMap[login] = &Reviewer{
					Login:  login,
					Avatar: review.GetUser().GetAvatarURL(),
					State:  state,
				}
				reviewTimes[login] = review.GetSubmittedAt().Time
			}
		}

		if resp.NextPage == 0 {
			break
		}
		reviewOpts.Page = resp.NextPage
	}

	for _, reviewer := range reviewMap {
		pr.Reviewers = append(pr.Reviewers, *reviewer)
	}

	pr.ReviewStatus = DeriveReviewStatus(reviewMap)

	// Fetch detailed PR info for mergeable state
	fullPR, resp, err := c.gh.PullRequests.Get(ctx, org, repo, number)
	if err != nil {
		return lastRate, err
	}
	if resp != nil {
		lastRate = rateFromResponse(resp)
	}

	pr.MergeableState = fullPR.GetMergeableState()
	pr.ReviewComments = fullPR.GetReviewComments()

	// Fetch check runs with pagination and explicit filter
	filterLatest := "latest"
	checkOpts := &gh.ListCheckRunsOptions{
		Filter:      &filterLatest,
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		if ctx.Err() != nil {
			return lastRate, ctx.Err()
		}

		checkRuns, resp, err := c.gh.Checks.ListCheckRunsForRef(ctx, org, repo, fullPR.GetHead().GetSHA(), checkOpts)
		if err != nil {
			return lastRate, err
		}
		if resp != nil {
			lastRate = rateFromResponse(resp)
		}

		for _, check := range checkRuns.CheckRuns {
			if check.GetConclusion() == "skipped" {
				continue
			}

			pr.Checks = append(pr.Checks, Check{
				Name:       check.GetName(),
				Status:     check.GetStatus(),
				Conclusion: check.GetConclusion(),
				URL:        check.GetHTMLURL(),
			})

			// Count check results — only explicit success counts as passed;
			// all other completed states (failure, timed_out, cancelled,
			// action_required, neutral, stale) count as failures.
			switch {
			case check.GetStatus() == "in_progress" || check.GetStatus() == "queued" ||
				check.GetStatus() == "waiting" || check.GetStatus() == "pending":
				pr.ChecksRunning++
			case check.GetConclusion() == "success":
				pr.ChecksSuccess++
			default:
				pr.ChecksFailure++
			}
		}

		if resp.NextPage == 0 {
			break
		}
		checkOpts.Page = resp.NextPage
	}

	return lastRate, nil
}
