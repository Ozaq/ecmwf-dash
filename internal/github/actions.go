package github

import (
	"context"
	"fmt"
	"log"

	gh "github.com/google/go-github/v83/github"
	"github.com/ozaq/ecmwf-dash/internal/config"
)

func (c *Client) FetchBranchChecks(ctx context.Context, org string, repos []config.RepositoryConfig) ChecksFetchResult {
	var result ChecksFetchResult
	successCount := 0

	for _, repo := range repos {
		if ctx.Err() != nil {
			break
		}

		repoFailed := true
		for _, branch := range repo.Branches {
			if ctx.Err() != nil {
				break
			}

			// Get the latest commit for this branch
			commits, resp, err := c.gh.Repositories.ListCommits(ctx, org, repo.Name, &gh.CommitsListOptions{
				SHA:         branch,
				ListOptions: gh.ListOptions{PerPage: 1},
			})
			if err != nil {
				log.Printf("Error fetching commits for %s/%s (branch: %s): %v", org, repo.Name, branch, err)
				continue
			}
			if resp != nil {
				result.Rate = rateFromResponse(resp)
			}

			if len(commits) == 0 {
				continue
			}

			latestCommit := commits[0]

			var allCheckRuns []*gh.CheckRun
			filterLatest := "latest"
			opts := &gh.ListCheckRunsOptions{
				Filter:      &filterLatest,
				ListOptions: gh.ListOptions{PerPage: 100},
			}

			for {
				if ctx.Err() != nil {
					break
				}

				checkRuns, resp, err := c.gh.Checks.ListCheckRunsForRef(ctx, org, repo.Name, latestCommit.GetSHA(), opts)
				if err != nil {
					log.Printf("Error fetching check runs for %s/%s (branch: %s, SHA: %s): %v", org, repo.Name, branch, latestCommit.GetSHA(), err)
					break
				}
				if resp != nil {
					result.Rate = rateFromResponse(resp)
				}
				allCheckRuns = append(allCheckRuns, checkRuns.CheckRuns...)
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}

			var checks []Check
			for _, check := range allCheckRuns {
				if check.GetConclusion() == "skipped" {
					continue
				}

				checks = append(checks, Check{
					Name:       check.GetName(),
					Status:     check.GetStatus(),
					Conclusion: check.GetConclusion(),
					URL:        check.GetHTMLURL(),
				})
			}

			branchCheck := BranchCheck{
				Repository: repo.Name,
				Branch:     branch,
				CommitSHA:  latestCommit.GetSHA(),
				CommitURL:  latestCommit.GetHTMLURL(),
				UpdatedAt:  latestCommit.GetCommit().GetCommitter().GetDate().Time,
				Checks:     checks,
			}

			result.BranchChecks = append(result.BranchChecks, branchCheck)
			successCount++
			repoFailed = false
		}

		if repoFailed {
			result.FailedRepos = append(result.FailedRepos, repo.Name)
		} else {
			result.SucceededRepos = append(result.SucceededRepos, repo.Name)
		}
	}

	// Any repos not attempted (due to context cancellation) count as failed
	attempted := len(result.SucceededRepos) + len(result.FailedRepos)
	for i := attempted; i < len(repos); i++ {
		result.FailedRepos = append(result.FailedRepos, repos[i].Name)
	}

	if successCount == 0 && len(repos) > 0 {
		if ctx.Err() != nil {
			result.Err = ctx.Err()
		} else {
			result.Err = fmt.Errorf("all branch check fetches failed")
		}
	}

	return result
}
