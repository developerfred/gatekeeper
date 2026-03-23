package models

import (
	"testing"
	"time"
)

func TestSeverityValues(t *testing.T) {
	severities := []Severity{SeverityInfo, SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
	for _, s := range severities {
		if s == "" {
			t.Error("Severity should not be empty")
		}
	}
}

func TestReviewActionValues(t *testing.T) {
	actions := []ReviewAction{ActionApprove, ActionRequestChanges, ActionComment, ActionClose}
	for _, a := range actions {
		if a == "" {
			t.Errorf("ReviewAction should not be empty")
		}
	}
}

func TestAuthorMergeRatio(t *testing.T) {
	tests := []struct {
		name      string
		author    Author
		wantValid bool
	}{
		{
			name: "valid author with good ratio",
			author: Author{
				Login:       "testuser",
				AccountAge:  365 * 24 * time.Hour,
				PRCount:     10,
				MergeRatio:  0.8,
				HasBio:      true,
				HasLocation: true,
			},
			wantValid: true,
		},
		{
			name: "new author",
			author: Author{
				Login:       "newuser",
				AccountAge:  1 * 24 * time.Hour,
				PRCount:     1,
				MergeRatio:  0.0,
				HasBio:      false,
				HasLocation: false,
			},
			wantValid: true,
		},
		{
			name: "author with 100 percent merge rate",
			author: Author{
				Login:      "perfectuser",
				MergeRatio: 1.0,
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.author.MergeRatio < 0.0 || tt.author.MergeRatio > 1.0 {
				t.Errorf("MergeRatio should be between 0.0 and 1.0, got %v", tt.author.MergeRatio)
			}
		})
	}
}

func TestPRFilesAdditionsDeletions(t *testing.T) {
	pr := PR{
		Files: []File{
			{Path: "foo.go", Additions: 10, Deletions: 5},
			{Path: "bar.go", Additions: 0, Deletions: 20},
		},
	}

	totalAdditions := 0
	totalDeletions := 0
	for _, f := range pr.Files {
		totalAdditions += f.Additions
		totalDeletions += f.Deletions
	}

	if totalAdditions != 10 {
		t.Errorf("Expected 10 total additions, got %d", totalAdditions)
	}
	if totalDeletions != 25 {
		t.Errorf("Expected 25 total deletions, got %d", totalDeletions)
	}
}

func TestAntiSlopResultPassed(t *testing.T) {
	tests := []struct {
		name     string
		result   AntiSlopResult
		expected bool
	}{
		{
			name:     "passed with high score",
			result:   AntiSlopResult{Passed: true, Score: 95},
			expected: true,
		},
		{
			name:     "failed with low score",
			result:   AntiSlopResult{Passed: false, Score: 30, Failures: []string{"empty_description"}},
			expected: false,
		},
		{
			name:     "passed with threshold score",
			result:   AntiSlopResult{Passed: true, Score: 70},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Passed != tt.expected {
				t.Errorf("Expected Passed=%v, got %v", tt.expected, tt.result.Passed)
			}
		})
	}
}

func TestReviewResultMergeScore(t *testing.T) {
	tests := []struct {
		name   string
		result ReviewResult
		valid  bool
	}{
		{
			name:   "excellent merge readiness",
			result: ReviewResult{MergeScore: 92, Action: ActionApprove},
			valid:  true,
		},
		{
			name:   "needs changes",
			result: ReviewResult{MergeScore: 55, Action: ActionRequestChanges},
			valid:  true,
		},
		{
			name:   "blocked",
			result: ReviewResult{MergeScore: 25, Action: ActionClose},
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.MergeScore < 0 || tt.result.MergeScore > 100 {
				t.Errorf("MergeScore should be between 0 and 100, got %d", tt.result.MergeScore)
			}
		})
	}
}

func TestTriageResultDuplicateOf(t *testing.T) {
	result := TriageResult{
		Action:      "close",
		DuplicateOf: intPtr(1234),
	}

	if result.DuplicateOf == nil {
		t.Error("Expected DuplicateOf to be set")
	}
	if *result.DuplicateOf != 1234 {
		t.Errorf("Expected DuplicateOf to be 1234, got %d", *result.DuplicateOf)
	}
}

func TestPriorityValues(t *testing.T) {
	priorities := []Priority{PriorityNone, PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}
	for _, p := range priorities {
		if p == "" {
			t.Errorf("Priority should not be empty")
		}
	}
}

func TestIssueTypeValues(t *testing.T) {
	types := []IssueType{IssueTypeBug, IssueTypeFeature, IssueTypeQuestion, IssueTypeDiscussion, IssueTypeOther}
	for _, it := range types {
		if it == "" {
			t.Errorf("IssueType should not be empty")
		}
	}
}

func TestEventPRWorkflow(t *testing.T) {
	event := Event{
		Type:   EventTypePROpened,
		Source: "github",
		Repo: RepoRef{
			Owner: "testowner",
			Name:  "testrepo",
			Host:  "github.com",
		},
		PR: &PR{
			Number: 123,
			Title:  "Test PR",
			Body:   "Test body",
		},
	}

	if event.PR == nil {
		t.Error("PR should be set for pr_opened event")
	}
	if event.PR.Number != 123 {
		t.Errorf("Expected PR number 123, got %d", event.PR.Number)
	}
}

func TestEventIssueWorkflow(t *testing.T) {
	event := Event{
		Type:   EventTypeIssueOpened,
		Source: "github",
		Repo: RepoRef{
			Owner: "testowner",
			Name:  "testrepo",
			Host:  "github.com",
		},
		Issue: &Issue{
			Number: 456,
			Title:  "Bug: something broke",
			Body:   "Detailed description",
		},
	}

	if event.Issue == nil {
		t.Error("Issue should be set for issue_opened event")
	}
	if event.Issue.Number != 456 {
		t.Errorf("Expected Issue number 456, got %d", event.Issue.Number)
	}
}

func TestPipelineResultErrors(t *testing.T) {
	result := PipelineResult{
		Errors: []string{"llm_unavailable", "rate_limited"},
	}

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(result.Errors))
	}
}

func intPtr(i int) *int {
	return &i
}
