package github

import (
	"context"
	"log"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/ozaq/ecmwf-dash/internal/config"
)

func (c *Client) FetchBranchChecks(ctx context.Context, org string, repos []config.RepositoryConfig) ([]BranchCheck, error) {
	var allChecks []BranchCheck

	for _, repo := range repos {

		for _, branch := range repo.Branches {
			// Get the latest commit for this branch
			commits, _, err := c.gh.Repositories.ListCommits(ctx, org, repo.Name, &github.CommitsListOptions{
				SHA:         branch,
				ListOptions: github.ListOptions{PerPage: 1},
			})
			if err != nil {
				log.Printf("Error fetching commits for %s/%s (branch: %s): %v", org, repo, branch, err)
				continue
			}

			if len(commits) == 0 {
				continue
			}

			latestCommit := commits[0]

			var allCheckRuns []*github.CheckRun
			opts := &github.ListCheckRunsOptions{
				ListOptions: github.ListOptions{PerPage: 100},
			}

			for {
				checkRuns, resp, err := c.gh.Checks.ListCheckRunsForRef(ctx, org, repo.Name, latestCommit.GetSHA(), opts)
				if err != nil {
					log.Printf("Error fetching check runs for %s/%s (branch: %s, SHA: %s): %v", org, repo, branch, latestCommit.GetSHA(), err)
					continue
				}
				allCheckRuns = append(allCheckRuns, checkRuns.CheckRuns...)
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}

			var checks []Check
			for _, check := range allCheckRuns {
				// Skip skipped checks
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
				UpdatedAt:  time.Now(),
				Checks:     checks,
			}

			allChecks = append(allChecks, branchCheck)
		}
	}

	return allChecks, nil
}
