package github

// ClassifyCheck returns "running", "success", or "failure" for a check run.
// Skipped checks should be pre-filtered by the caller.
//
// Classification:
//   - in_progress/queued/waiting/pending → "running"
//   - conclusion == "success"            → "success"
//   - everything else (failure, timed_out, cancelled, action_required,
//     neutral, stale, unknown)           → "failure"
func ClassifyCheck(status, conclusion string) string {
	switch status {
	case "in_progress", "queued", "waiting", "pending":
		return "running"
	}
	if conclusion == "success" {
		return "success"
	}
	return "failure"
}
