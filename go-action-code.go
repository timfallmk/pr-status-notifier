package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type CheckStatus struct {
	Name       string
	Status     string
	Conclusion string
}

// GitHub Actions output commands
func notice(msg string) {
	fmt.Printf("\n::notice::%s\n", msg)
}

func warning(msg string) {
	fmt.Printf("\n::warning::%s\n", msg)
}

func fail(msg string) {
	fmt.Printf("\n::error::%s\n", msg)
	os.Exit(1)
}

func getInput(name string) string {
	return os.Getenv("INPUT_" + strings.ToUpper(name))
}

func main() {
	// Get action inputs
	token := getInput("github-token")
	fmt.Printf("Token: %s\n", token)
	if token == "" {
		fail("github-token input is required")
	}
	excludedChecks := strings.Split(getInput("excluded-checks"), ",")
	notificationTemplate := getInput("notification-message")

	// Create GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get GitHub context
	eventName := os.Getenv("GITHUB_EVENT_NAME")
	repo := os.Getenv("GITHUB_REPOSITORY")
	repoParts := strings.Split(repo, "/")
	owner, repoName := repoParts[0], repoParts[1]

	notice(fmt.Sprintf("Starting PR status check for %s/%s", owner, repoName))

	// Get SHA and PR number based on event type
	var sha string
	var prNumber int
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	eventFile, err := os.ReadFile(eventPath)
	if err != nil {
		fail(fmt.Sprintf("Error reading event file: %v", err))
	}

	switch eventName {
	case "pull_request":
		var event github.PullRequestEvent
		if err := json.Unmarshal(eventFile, &event); err != nil {
			fail(fmt.Sprintf("Error parsing pull request event: %v", err))
		}
		sha = *event.PullRequest.Head.SHA
		prNumber = *event.PullRequest.Number
		notice(fmt.Sprintf("PR event detected: SHA=%s, PR=#%d", sha, prNumber))

	case "status":
		var event github.StatusEvent
		if err := json.Unmarshal(eventFile, &event); err != nil {
			fail(fmt.Sprintf("Error parsing status event: %v", err))
		}
		sha = *event.SHA
		notice(fmt.Sprintf("Status event detected: SHA=%s", sha))

	default:
		fail(fmt.Sprintf("Unsupported event type: %s", eventName))
	}

	// If we don't have a PR number yet, find it from the SHA
	if prNumber == 0 {
		prs, _, err := client.PullRequests.ListPullRequestsWithCommit(ctx, owner, repoName, sha, nil)
		if err != nil {
			fail(fmt.Sprintf("Error finding PR for SHA: %v", err))
		}
		if len(prs) == 0 {
			warning(fmt.Sprintf("No PR found for SHA: %s", sha))
			return
		}
		prNumber = *prs[0].Number
	}

	// Get PR details
	pr, _, err := client.PullRequests.Get(ctx, owner, repoName, prNumber)
	if err != nil {
		fail(fmt.Sprintf("Error getting PR details: %v", err))
	}
	notice(fmt.Sprintf("Found PR #%d: %s", prNumber, *pr.Title))

	// Get combined status
	combinedStatus, _, err := client.Repositories.GetCombinedStatus(ctx, owner, repoName, sha, nil)
	if err != nil {
		fail(fmt.Sprintf("Error getting combined status: %v", err))
	}

	// Get check runs
	checkRuns, _, err := client.Checks.ListCheckRunsForRef(ctx, owner, repoName, sha, nil)
	if err != nil {
		fail(fmt.Sprintf("Error getting check runs: %v", err))
	}

	// Combine and filter checks
	var relevantChecks []CheckStatus
	for _, status := range combinedStatus.Statuses {
		if !isExcluded(*status.Context, excludedChecks) {
			relevantChecks = append(relevantChecks, CheckStatus{
				Name:       *status.Context,
				Status:     "completed",
				Conclusion: *status.State,
			})
		}
	}

	for _, check := range checkRuns.CheckRuns {
		if !isExcluded(*check.Name, excludedChecks) {
			relevantChecks = append(relevantChecks, CheckStatus{
				Name:       *check.Name,
				Status:     *check.Status,
				Conclusion: *check.Conclusion,
			})
		}
	}

	notice(fmt.Sprintf("Found %d relevant checks", len(relevantChecks)))

	// Check if all passed
	allPassed := true
	for _, check := range relevantChecks {
		isPassed := false
		if check.Status == "completed" {
			switch check.Conclusion {
			case "success", "neutral", "skipped":
				isPassed = true
			}
		}
		if !isPassed {
			allPassed = false
		}
		status := "❌"
		if isPassed {
			status = "✅"
		}
		log.Printf("Check %q: %s/%s %s", check.Name, check.Status, check.Conclusion, status)
	}

	if allPassed && len(relevantChecks) > 0 {
		notice("✅ All non-excluded checks have passed!")

		// Check for existing notification
		comments, _, err := client.Issues.ListComments(ctx, owner, repoName, prNumber, nil)
		if err != nil {
			fail(fmt.Sprintf("Error checking comments: %v", err))
		}

		hasNotification := false
		for _, comment := range comments {
			if strings.Contains(*comment.Body, "All checks have passed!") {
				hasNotification = true
				break
			}
		}

		if !hasNotification {
			message := strings.ReplaceAll(notificationTemplate, "{user}", *pr.User.Login)
			_, _, err = client.Issues.CreateComment(ctx, owner, repoName, prNumber, &github.IssueComment{
				Body: &message,
			})
			if err != nil {
				fail(fmt.Sprintf("Error creating comment: %v", err))
			}
			notice("Notification comment created successfully")
		}
	} else {
		warning("⏳ Some checks are still pending or failed")
	}
}

func isExcluded(name string, excludedChecks []string) bool {
	nameLower := strings.ToLower(name)
	for _, excluded := range excludedChecks {
		if strings.Contains(nameLower, strings.ToLower(strings.TrimSpace(excluded))) {
			return true
		}
	}
	return false
}
