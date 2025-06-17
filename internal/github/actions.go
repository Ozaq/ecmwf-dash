package github

import (
    "context"
    "fmt"

    "github.com/google/go-github/v66/github"
)

func (c *Client) FetchWorkflowRuns(ctx context.Context, org string, repos []string) ([]WorkflowRun, error) {
    var allRuns []WorkflowRun

    for _, repo := range repos {
        // Fetch workflow runs for main/master branches
        branches := []string{"main", "master", "develop"}
        
        for _, branch := range branches {
            opts := &github.ListWorkflowRunsOptions{
                Branch: branch,
                Status: "completed", // Get completed runs to see latest status
                ListOptions: github.ListOptions{
                    PerPage: 10, // Get recent runs
                },
            }

            runs, _, err := c.gh.Actions.ListRepositoryWorkflowRuns(ctx, org, repo, opts)
            if err != nil {
                // Log error but continue with other repos/branches
                fmt.Printf("Error fetching workflow runs for %s/%s (branch: %s): %v\n", org, repo, branch, err)
                continue
            }

            for _, run := range runs.WorkflowRuns {
                workflowRun := WorkflowRun{
                    Repository:    repo,
                    Branch:        branch,
                    WorkflowName:  run.GetName(),
                    WorkflowID:    run.GetWorkflowID(),
                    RunID:         run.GetID(),
                    RunNumber:     run.GetRunNumber(),
                    Status:        run.GetStatus(),
                    Conclusion:    run.GetConclusion(),
                    CreatedAt:     run.GetCreatedAt().Time,
                    UpdatedAt:     run.GetUpdatedAt().Time,
                    URL:           run.GetHTMLURL(),
                    HeadBranch:    run.GetHeadBranch(),
                    HeadSHA:       run.GetHeadSHA(),
                    Event:         run.GetEvent(),
                    TriggerActor:  run.GetTriggeringActor().GetLogin(),
                }

                allRuns = append(allRuns, workflowRun)
            }
        }
    }

    return allRuns, nil
}
