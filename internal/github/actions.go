package github

import (
    "context"
    "fmt"
    "time"

    "github.com/google/go-github/v66/github"
)

func (c *Client) FetchBranchChecks(ctx context.Context, org string, repos []string) ([]BranchCheck, error) {
    var allChecks []BranchCheck

    for _, repo := range repos {
        branches := []string{"main", "master", "develop"}
        
        for _, branch := range branches {
            // Get the latest commit for this branch
            commits, _, err := c.gh.Repositories.ListCommits(ctx, org, repo, &github.CommitsListOptions{
                SHA: branch,
                ListOptions: github.ListOptions{PerPage: 1},
            })
            if err != nil {
                fmt.Printf("Error fetching commits for %s/%s (branch: %s): %v\n", org, repo, branch, err)
                continue
            }
            
            if len(commits) == 0 {
                continue
            }
            
            latestCommit := commits[0]
            
            // Fetch check runs for the latest commit
            checkRuns, _, err := c.gh.Checks.ListCheckRunsForRef(ctx, org, repo, latestCommit.GetSHA(), &github.ListCheckRunsOptions{
                ListOptions: github.ListOptions{PerPage: 100},
            })
            if err != nil {
                fmt.Printf("Error fetching check runs for %s/%s (branch: %s, SHA: %s): %v\n", org, repo, branch, latestCommit.GetSHA(), err)
                continue
            }

            var checks []Check
            for _, check := range checkRuns.CheckRuns {
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
                Repository: repo,
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