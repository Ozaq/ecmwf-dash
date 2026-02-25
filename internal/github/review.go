package github

// DeriveReviewStatus computes an aggregate review status from a set of
// per-reviewer states. The map keys are reviewer logins, values are their
// latest review state (APPROVED, CHANGES_REQUESTED, etc.).
//
// Priority: any CHANGES_REQUESTED -> "changes_requested",
// else any APPROVED -> "approved", else "pending".
func DeriveReviewStatus(reviewers map[string]*Reviewer) string {
	status := "pending"
	for _, reviewer := range reviewers {
		switch reviewer.State {
		case "CHANGES_REQUESTED":
			return "changes_requested"
		case "APPROVED":
			status = "approved"
		}
	}
	return status
}
