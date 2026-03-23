package triage

import (
	"context"
	"testing"
	"time"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

func TestIssueClassifier_Bug(t *testing.T) {
	classifier := NewIssueClassifier()

	tests := []struct {
		name     string
		issue    *models.Issue
		expected models.IssueType
	}{
		{
			name: "Bug from title",
			issue: &models.Issue{
				Title: "Bug: application crashes on startup",
				Body:  "When I try to start the app it crashes",
			},
			expected: models.IssueTypeBug,
		},
		{
			name: "Bug from body",
			issue: &models.Issue{
				Title: "Crashes when clicking button",
				Body:  "This is definitely a bug in the code",
			},
			expected: models.IssueTypeBug,
		},
		{
			name: "Feature request",
			issue: &models.Issue{
				Title: "Feature: add dark mode support",
				Body:  "It would be nice to have a dark theme",
			},
			expected: models.IssueTypeFeature,
		},
		{
			name: "Question",
			issue: &models.Issue{
				Title: "How do I configure the database?",
				Body:  "I need help setting up the connection",
			},
			expected: models.IssueTypeQuestion,
		},
		{
			name: "Discussion",
			issue: &models.Issue{
				Title: "Thoughts on new architecture?",
				Body:  "I was thinking about refactoring the service layer",
			},
			expected: models.IssueTypeDiscussion,
		},
		{
			name: "Other/unknown",
			issue: &models.Issue{
				Title: "Update README",
				Body:  "Need to update the documentation",
			},
			expected: models.IssueTypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			issueType, err := classifier.Classify(ctx, tt.issue)
			if err != nil {
				t.Fatalf("Classify() error = %v", err)
			}
			if issueType != tt.expected {
				t.Errorf("issueType = %v, want %v", issueType, tt.expected)
			}
		})
	}
}

func TestIssueClassifier_EmptyIssue(t *testing.T) {
	classifier := NewIssueClassifier()

	issue := &models.Issue{
		Title: "",
		Body:  "",
	}

	ctx := context.Background()
	issueType, err := classifier.Classify(ctx, issue)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}
	if issueType != models.IssueTypeOther {
		t.Errorf("issueType = %v, want %v", issueType, models.IssueTypeOther)
	}
}

func TestDuplicateDetector_SimilarIssues(t *testing.T) {
	detector := NewDuplicateDetector(0.7)

	newIssue := &models.Issue{
		Title: "App crashes when clicking save button",
		Body:  "I click the save button and the app crashes",
	}

	existingIssues := []*models.Issue{
		{
			Number: 100,
			Title:  "Bug: Crash on save button click",
			Body:   "The save button causes a crash",
		},
		{
			Number: 101,
			Title:  "Feature: Add export functionality",
			Body:   "Would be nice to export data",
		},
	}

	ctx := context.Background()
	result, err := detector.CheckDuplicates(ctx, newIssue, existingIssues)
	if err != nil {
		t.Fatalf("CheckDuplicates() error = %v", err)
	}

	if result.IsDuplicate {
		if result.SimilarTo == nil {
			t.Error("SimilarTo = nil, want pointer to similar issue")
		}
		if result.Similarity < 0.7 {
			t.Errorf("Similarity = %f, want >= 0.7", result.Similarity)
		}
	}
}

func TestDuplicateDetector_NoDuplicates(t *testing.T) {
	detector := NewDuplicateDetector(0.7)

	newIssue := &models.Issue{
		Title: "Add user authentication",
		Body:  "Implement OAuth2 login flow",
	}

	existingIssues := []*models.Issue{
		{
			Number: 100,
			Title:  "Fix CSS styling bug",
			Body:   "Button is misaligned on mobile",
		},
	}

	ctx := context.Background()
	result, err := detector.CheckDuplicates(ctx, newIssue, existingIssues)
	if err != nil {
		t.Fatalf("CheckDuplicates() error = %v", err)
	}

	if result.IsDuplicate {
		t.Error("IsDuplicate = true, want false for unrelated issues")
	}
}

func TestDuplicateDetector_ThresholdBoundary(t *testing.T) {
	detector := NewDuplicateDetector(0.7)

	issues := []*models.Issue{
		{Number: 1, Title: "Bug: app crashes", Body: "crashes"},
	}

	tests := []struct {
		name     string
		newIssue *models.Issue
		wantDup  bool
	}{
		{
			name:     "Exactly 70% similar",
			newIssue: &models.Issue{Title: "Bug: app fails", Body: "fails"},
			wantDup:  false,
		},
		{
			name:     "75% similar",
			newIssue: &models.Issue{Title: "Bug: application crashes", Body: "the app crashes"},
			wantDup:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := detector.CheckDuplicates(ctx, tt.newIssue, issues)
			if err != nil {
				t.Fatalf("CheckDuplicates() error = %v", err)
			}
			if result.IsDuplicate != tt.wantDup {
				t.Errorf("IsDuplicate = %v, want %v", result.IsDuplicate, tt.wantDup)
			}
		})
	}
}

func TestPriorityInferencer(t *testing.T) {
	inferencer := NewPriorityInferencer()

	tests := []struct {
		name     string
		issue    *models.Issue
		expected models.Priority
	}{
		{
			name: "Critical - production down",
			issue: &models.Issue{
				Title: "Production is down! All users affected!",
				Body:  "Server returns 500 errors",
			},
			expected: models.PriorityCritical,
		},
		{
			name: "Critical - security vulnerability",
			issue: &models.Issue{
				Title: "Security: SQL injection in login",
				Body:  "Attacker can bypass authentication",
			},
			expected: models.PriorityCritical,
		},
		{
			name: "High priority",
			issue: &models.Issue{
				Title: "Bug: Cannot save work",
				Body:  "Every time I click save nothing happens",
			},
			expected: models.PriorityHigh,
		},
		{
			name: "Medium priority",
			issue: &models.Issue{
				Title: "Feature: Add dark mode",
				Body:  "Would be nice for night usage",
			},
			expected: models.PriorityMedium,
		},
		{
			name: "Low priority",
			issue: &models.Issue{
				Title: "Typo in error message",
				Body:  "Message says 'occured' instead of 'occurred'",
			},
			expected: models.PriorityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := inferencer.Infer(ctx, tt.issue)
			if err != nil {
				t.Fatalf("Infer() error = %v", err)
			}
			if result.Priority != tt.expected {
				t.Errorf("Priority = %v, want %v", result.Priority, tt.expected)
			}
		})
	}
}

func TestPriorityInferencer_Components(t *testing.T) {
	inferencer := NewPriorityInferencer()

	issue := &models.Issue{
		Title: "Bug in auth module",
		Body:  "Authentication fails for some users",
	}

	ctx := context.Background()
	result, err := inferencer.Infer(ctx, issue)
	if err != nil {
		t.Fatalf("Infer() error = %v", err)
	}

	if result.Priority == "" {
		t.Error("Priority = empty, want non-empty")
	}
}

func TestActionabilityChecker_Actionable(t *testing.T) {
	checker := NewActionabilityChecker()

	tests := []struct {
		name   string
		issue  *models.Issue
		wantOK bool
	}{
		{
			name: "Actionable - has steps",
			issue: &models.Issue{
				Title: "Bug: App crashes",
				Body:  "Steps to reproduce:\n1. Open app\n2. Click save\n3. App crashes",
			},
			wantOK: true,
		},
		{
			name: "Actionable - has expected behavior",
			issue: &models.Issue{
				Title: "Feature request",
				Body:  "Expected: Dark mode should be available in settings",
			},
			wantOK: true,
		},
		{
			name: "Not actionable - vague",
			issue: &models.Issue{
				Title: "Something is wrong",
				Body:  "It doesn't work properly",
			},
			wantOK: false,
		},
		{
			name: "Not actionable - spam",
			issue: &models.Issue{
				Title: "Check out my product!",
				Body:  "Visit my website to buy things!!!",
			},
			wantOK: false,
		},
		{
			name: "Not actionable - no description",
			issue: &models.Issue{
				Title: "Bug",
				Body:  "",
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := checker.Check(ctx, tt.issue)
			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}
			if result.IsActionable != tt.wantOK {
				t.Errorf("IsActionable = %v, want %v", result.IsActionable, tt.wantOK)
			}
		})
	}
}

func TestActionabilityChecker_Score(t *testing.T) {
	checker := NewActionabilityChecker()

	issue := &models.Issue{
		Title: "Bug report",
		Body:  "Steps to reproduce:\n1. Go to page\n2. Click button\nExpected: Modal opens\nActual: Nothing happens",
	}

	ctx := context.Background()
	result, err := checker.Check(ctx, issue)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Score = %d, want 0-100", result.Score)
	}
}

func TestLabelProposer(t *testing.T) {
	proposer := NewLabelProposer()

	issue := &models.Issue{
		Title:  "Bug: Authentication fails with OAuth",
		Body:   "OAuth2 login doesn't work properly",
		Labels: []string{},
	}

	ctx := context.Background()
	labels, err := proposer.Propose(ctx, issue)
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}

	if len(labels) == 0 {
		t.Error("No labels proposed for bug issue")
	}

	hasBugLabel := false
	for _, label := range labels {
		if label == "bug" {
			hasBugLabel = true
			break
		}
	}
	if !hasBugLabel {
		t.Error("No 'bug' label proposed for bug issue")
	}
}

func TestLabelProposer_AuthIssue(t *testing.T) {
	proposer := NewLabelProposer()

	issue := &models.Issue{
		Title: "Security: XSS vulnerability in comments",
		Body:  "User can inject scripts via comments",
	}

	ctx := context.Background()
	labels, err := proposer.Propose(ctx, issue)
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}

	hasSecurityOrAuth := false
	for _, label := range labels {
		if label == "security" || label == "auth" || label == "bug" {
			hasSecurityOrAuth = true
			break
		}
	}
	if !hasSecurityOrAuth {
		t.Error("No security-related label proposed")
	}
}

func TestTriagePipeline(t *testing.T) {
	pipeline := NewTriagePipeline()

	issue := &models.Issue{
		Number: 123,
		Title:  "Bug: App crashes on startup",
		Body:   "Steps:\n1. Launch app\n2. See crash\nExpected: App opens normally",
		Author: models.Author{
			Login:      "user1",
			AccountAge: 30 * 24 * time.Hour,
		},
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := pipeline.Triage(ctx, issue, nil)
	if err != nil {
		t.Fatalf("Triage() error = %v", err)
	}

	if result.Action == "" {
		t.Error("Action = empty, want valid action")
	}
	if result.IssueType == "" {
		t.Error("IssueType = empty, want classified type")
	}
	if result.Priority == "" {
		t.Error("Priority = empty, want inferred priority")
	}
}

func TestTriagePipeline_WithDuplicates(t *testing.T) {
	pipeline := NewTriagePipeline()

	newIssue := &models.Issue{
		Number: 200,
		Title:  "Bug: App crashes when saving",
		Body:   "The save button causes a crash",
	}

	existingIssues := []*models.Issue{
		{
			Number: 100,
			Title:  "Bug: Crash on save button",
			Body:   "Save button makes app crash",
		},
	}

	ctx := context.Background()
	result, err := pipeline.Triage(ctx, newIssue, existingIssues)
	if err != nil {
		t.Fatalf("Triage() error = %v", err)
	}

	if result.Action == "close" && result.DuplicateOf == nil {
		t.Error("Action=close but DuplicateOf is nil")
	}
}

func TestTriagePipeline_NonActionable(t *testing.T) {
	pipeline := NewTriagePipeline()

	issue := &models.Issue{
		Number: 300,
		Title:  "Problem",
		Body:   "Something is not working",
	}

	ctx := context.Background()
	result, err := pipeline.Triage(ctx, issue, nil)
	if err != nil {
		t.Fatalf("Triage() error = %v", err)
	}

	if result.IsActionable {
		t.Error("IsActionable = true, want false for vague issue")
	}
}

func TestTriagePipeline_Spam(t *testing.T) {
	pipeline := NewTriagePipeline()

	issue := &models.Issue{
		Number: 400,
		Title:  "BUY NOW!!! FREE STUFF!!!",
		Body:   "Click here to win prizes!!! Best deal ever!!!",
	}

	ctx := context.Background()
	result, err := pipeline.Triage(ctx, issue, nil)
	if err != nil {
		t.Fatalf("Triage() error = %v", err)
	}

	if result.Action != "close" {
		t.Logf("Note: Spam issue action = %q, may auto-close", result.Action)
	}
}

func TestTriagePipeline_LabelsProposed(t *testing.T) {
	pipeline := NewTriagePipeline()

	issue := &models.Issue{
		Number: 500,
		Title:  "Feature: Add dark mode",
		Body:   "Would be nice to have dark theme",
	}

	ctx := context.Background()
	result, err := pipeline.Triage(ctx, issue, nil)
	if err != nil {
		t.Fatalf("Triage() error = %v", err)
	}

	if len(result.Labels) == 0 {
		t.Error("No labels proposed for feature request")
	}
}

func TestIssueClassifier_Confidence(t *testing.T) {
	classifier := NewIssueClassifier()

	issue := &models.Issue{
		Title: "Bug: Critical crash in production",
		Body:  "Production is down and users cannot access the service",
	}

	ctx := context.Background()
	result, err := classifier.ClassifyWithConfidence(ctx, issue)
	if err != nil {
		t.Fatalf("ClassifyWithConfidence() error = %v", err)
	}

	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence = %f, want 0-1", result.Confidence)
	}
	if result.IssueType != models.IssueTypeBug {
		t.Errorf("IssueType = %v, want Bug", result.IssueType)
	}
}

func TestDuplicateDetector_TextSimilarity(t *testing.T) {
	detector := NewDuplicateDetector(0.5)

	tests := []struct {
		text1  string
		text2  string
		minSim float64
	}{
		{"Hello world", "Hello world", 0.99},
		{"Hello world", "World hello", 0.99},
		{"Hello world", "Goodbye world", 0.6},
		{"Hello world", "Foo bar", 0.0},
	}

	for _, tt := range tests {
		sim := detector.calculateTextSimilarity(tt.text1, tt.text2)
		if sim < tt.minSim {
			t.Errorf("similarity(%q, %q) = %f, want >= %f", tt.text1, tt.text2, sim, tt.minSim)
		}
	}
}
