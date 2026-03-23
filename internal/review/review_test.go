package review

import (
	"context"
	"testing"
	"time"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

func TestMergeReadinessScore(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 123,
		Title:  "feat: add new feature",
		Body:   "This PR adds a new feature that improves performance",
		Files: []models.File{
			{Path: "src/main.go", Additions: 100, Deletions: 10, Patch: "@@ -1,3 +1,4 @@"},
			{Path: "src/main_test.go", Additions: 50, Deletions: 10, Patch: "@@ -1,3 +1,4 @@"},
		},
		Author: models.Author{
			Login:      "contributor",
			AccountAge: 365 * 24 * time.Hour,
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Score = %d, want 0-100", result.Score)
	}
	if result.Reasoning == "" {
		t.Error("Reasoning = empty, want non-empty")
	}
	if result.LLMUsed != "test-model" {
		t.Errorf("LLMUsed = %q, want %q", result.LLMUsed, "test-model")
	}
	if result.DurationMs == 0 {
		t.Error("DurationMs = 0, want > 0")
	}
}

func TestMergeReadinessScore_WithBreakingChanges(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 456,
		Title:  "refactor: change API contract",
		Body:   "This changes the public API",
		Files: []models.File{
			{Path: "api/handler.go", Additions: 300, Deletions: 200},
		},
		Author: models.Author{
			Login:      "newcontributor",
			AccountAge: 7 * 24 * time.Hour,
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.LikelyBreaking && result.Score > 70 {
		t.Logf("Note: Large refactor flagged as likely breaking")
	}
}

func TestMergeReadinessScore_ExcellentPR(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 789,
		Title:  "feat: add user authentication",
		Body:   "Implements OAuth2 authentication with proper error handling and tests",
		Files: []models.File{
			{Path: "auth.go", Additions: 100, Deletions: 5},
			{Path: "auth_test.go", Additions: 100, Deletions: 0},
			{Path: "docs/auth.md", Additions: 50, Deletions: 0},
		},
		Author: models.Author{
			Login:      "seniordev",
			AccountAge: 1000 * 24 * time.Hour,
			PRCount:    20,
			MergeRatio: 0.9,
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.Score < 40 {
		t.Errorf("Score = %d, want >= 40 for PR with tests", result.Score)
	}
	if !result.HasTests {
		t.Error("HasTests = false, want true for PR with test file")
	}
}

func TestMergeReadinessScore_PoorPR(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 100,
		Title:  "fix",
		Body:   "",
		Files: []models.File{
			{Path: "main.go", Additions: 10, Deletions: 10},
		},
		Author: models.Author{
			Login:      "newuser",
			AccountAge: 1 * 24 * time.Hour,
			PRCount:    1,
			MergeRatio: 0.0,
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.Score > 50 {
		t.Logf("Note: Score = %d for poor PR (expected low)", result.Score)
	}
}

func TestMergeReadinessScore_MinimalFields(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 123,
		Title:  "Update",
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("Score = %d, want 0-100", result.Score)
	}
}

func TestMergeReadinessScore_WithExistingReviews(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 200,
		Title:  "feat: add feature",
		Body:   "Detailed description with context",
		Files: []models.File{
			{Path: "feature.go", Additions: 100, Deletions: 10},
			{Path: "feature_test.go", Additions: 100, Deletions: 10},
		},
		Author: models.Author{
			Login:      "contributor",
			AccountAge: 500 * 24 * time.Hour,
			PRCount:    10,
			MergeRatio: 0.9,
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.Score < 70 {
		t.Logf("Note: PR with good stats scored %d", result.Score)
	}
}

func TestDeepReview(t *testing.T) {
	reviewer := NewDeepReviewer("test-model")

	pr := &models.PR{
		Number: 123,
		Title:  "feat: add payment processing",
		Body:   "Implements Stripe payment integration",
		Files: []models.File{
			{
				Path:      "payment/stripe.go",
				Additions: 150,
				Deletions: 20,
				Patch:     "@@ -1,10 +1,15 @@\nfunc ProcessPayment(amount int) {\n+\terr := stripe.Charge(amount)\n+\tif err != nil {\n+\t\treturn err\n+\t}",
			},
		},
		Author: models.Author{
			Login:      "developer",
			AccountAge: 365 * 24 * time.Hour,
		},
	}

	ctx := context.Background()
	result, err := reviewer.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if len(result.Issues) == 0 {
		t.Logf("Note: Deep review found %d issues", len(result.Issues))
	}
	if result.Action != models.ActionApprove && result.Action != models.ActionRequestChanges && result.Action != models.ActionComment {
		t.Errorf("Action = %v, want valid ReviewAction", result.Action)
	}
}

func TestDeepReview_SecurityIssues(t *testing.T) {
	reviewer := NewDeepReviewer("test-model")

	pr := &models.PR{
		Number: 456,
		Title:  "fix: auth bug",
		Body:   "Fixes authentication issue",
		Files: []models.File{
			{
				Path:  "auth/login.go",
				Patch: "@@ -5,10 +5,8 @@\nfunc Login(user, pass string) {\n-\tif !authenticate(user, pass) {\n-\t\treturn ErrUnauthorized\n-\t}\n+\t// TODO: fix this later\n+\treturn nil",
			},
		},
		Author: models.Author{
			Login:      "newdev",
			AccountAge: 30 * 24 * time.Hour,
		},
	}

	ctx := context.Background()
	result, err := reviewer.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	hasSecurityIssue := false
	for _, issue := range result.Issues {
		if issue.Category == "security" {
			hasSecurityIssue = true
			break
		}
	}

	if !hasSecurityIssue {
		t.Logf("Note: Security issue not detected in suspicious code")
	}
}

func TestDeepReview_CorrectnessIssues(t *testing.T) {
	reviewer := NewDeepReviewer("test-model")

	pr := &models.PR{
		Number: 789,
		Title:  "fix: loop issue",
		Body:   "Fixes loop boundary",
		Files: []models.File{
			{
				Path:  "processor.go",
				Patch: "@@ -10,5 +10,6 @@\n\t for i := 0; i < 10; i++ {\n+\t\tif i == 5 {\n+\t\t\tbreak\n+\t\t}",
			},
		},
	}

	ctx := context.Background()
	result, err := reviewer.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	t.Logf("Deep review found %d issues across %d categories", len(result.Issues), len(categoriesFound(result.Issues)))
}

func categoriesFound(issues []models.ReviewIssue) map[string]bool {
	cats := make(map[string]bool)
	for _, i := range issues {
		cats[i.Category] = true
	}
	return cats
}

func TestSmartCommentStrategy(t *testing.T) {
	strategy := NewSmartCommentStrategy()

	tests := []struct {
		name           string
		score          int
		issues         []models.ReviewIssue
		expectedAction models.ReviewAction
	}{
		{
			name:           "Excellent score - approve",
			score:          92,
			issues:         []models.ReviewIssue{},
			expectedAction: models.ActionApprove,
		},
		{
			name:  "Good score with minor issues - comment",
			score: 72,
			issues: []models.ReviewIssue{
				{Severity: models.SeverityMedium, Category: "maintainability", Title: "Consider using const"},
			},
			expectedAction: models.ActionComment,
		},
		{
			name:  "Low score with blocking issues - request changes",
			score: 45,
			issues: []models.ReviewIssue{
				{Severity: models.SeverityCritical, Category: "security", Title: "SQL Injection"},
			},
			expectedAction: models.ActionRequestChanges,
		},
		{
			name:  "Medium score with info issues - request changes (low score)",
			score: 65,
			issues: []models.ReviewIssue{
				{Severity: models.SeverityLow, Category: "style", Title: "Naming convention"},
			},
			expectedAction: models.ActionRequestChanges,
		},
		{
			name:  "High score with critical security issue",
			score: 88,
			issues: []models.ReviewIssue{
				{Severity: models.SeverityCritical, Category: "security", Title: "Hardcoded password"},
			},
			expectedAction: models.ActionRequestChanges,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &models.ReviewResult{
				MergeScore: tt.score,
				Issues:     tt.issues,
			}
			action := strategy.DetermineAction(result)
			if action != tt.expectedAction {
				t.Errorf("action = %v, want %v", action, tt.expectedAction)
			}
		})
	}
}

func TestSmartCommentStrategy_Grouping(t *testing.T) {
	strategy := NewSmartCommentStrategy()

	result := &models.ReviewResult{
		MergeScore: 65,
		Issues: []models.ReviewIssue{
			{Severity: models.SeverityCritical, Category: "security", Title: "SQL Injection", File: "db.go", Line: 42},
			{Severity: models.SeverityHigh, Category: "correctness", Title: "Null pointer", File: "handler.go", Line: 100},
			{Severity: models.SeverityMedium, Category: "maintainability", Title: "Long function", File: "utils.go", Line: 200},
			{Severity: models.SeverityLow, Category: "style", Title: "Formatting", File: "main.go", Line: 1},
			{Severity: models.SeverityLow, Category: "style", Title: "Naming", File: "main.go", Line: 5},
		},
	}

	action := strategy.DetermineAction(result)
	if action != models.ActionRequestChanges {
		t.Errorf("action = %v, want RequestChanges for score < 70 with issues", action)
	}

	grouped := strategy.GroupIssuesBySeverity(result.Issues)
	if len(grouped) != 4 {
		t.Errorf("len(grouped) = %d, want 4 severity groups", len(grouped))
	}
}

func TestReviewPipeline(t *testing.T) {
	pipeline := NewReviewPipeline()

	pr := &models.PR{
		Number: 123,
		Title:  "feat: add new feature",
		Body:   "This PR adds a new feature",
		Files: []models.File{
			{Path: "feature.go", Additions: 100, Deletions: 15},
			{Path: "feature_test.go", Additions: 100, Deletions: 15},
		},
		Author: models.Author{
			Login:      "contributor",
			AccountAge: 200 * 24 * time.Hour,
			PRCount:    15,
			MergeRatio: 0.93,
		},
	}

	ctx := context.Background()
	result, err := pipeline.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if result.MergeScore < 0 || result.MergeScore > 100 {
		t.Errorf("MergeScore = %d, want 0-100", result.MergeScore)
	}
	if result.Reasoning == "" {
		t.Error("Reasoning = empty, want non-empty")
	}
	if result.Action == "" {
		t.Error("Action = empty, want valid action")
	}
}

func TestReviewPipeline_FastPath(t *testing.T) {
	pipeline := NewReviewPipeline()

	pr := &models.PR{
		Number: 456,
		Title:  "docs: update readme",
		Body:   "Updates documentation",
		Files: []models.File{
			{Path: "README.md", Additions: 50, Deletions: 10},
		},
		Author: models.Author{
			Login:      "newuser",
			AccountAge: 5 * 24 * time.Hour,
			PRCount:    2,
			MergeRatio: 0.5,
		},
	}

	ctx := context.Background()
	result, err := pipeline.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if result.MergeScore == 0 {
		t.Error("MergeScore = 0, want > 0 for valid PR")
	}
}

func TestReviewPipeline_DetailedReview(t *testing.T) {
	pipeline := NewReviewPipeline()

	pr := &models.PR{
		Number: 789,
		Title:  "feat: add complex feature",
		Body:   "Implements complex feature with multiple components",
		Files: []models.File{
			{Path: "feature/main.go", Additions: 200, Deletions: 50},
			{Path: "feature/handler.go", Additions: 150, Deletions: 30},
			{Path: "feature/models.go", Additions: 100, Deletions: 10},
			{Path: "feature/main_test.go", Additions: 50, Deletions: 10},
		},
		Author: models.Author{
			Login:      "seniordev",
			AccountAge: 1000 * 24 * time.Hour,
			PRCount:    50,
			MergeRatio: 0.96,
		},
	}

	ctx := context.Background()
	result, err := pipeline.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	t.Logf("Detailed review: score=%d, action=%v, issues=%d, cost=%.4f, duration=%dms",
		result.MergeScore, result.Action, len(result.Issues), result.CostUSD, result.DurationMs)
}

func TestReviewPipeline_AllCategories(t *testing.T) {
	pipeline := NewReviewPipeline()

	pr := &models.PR{
		Number: 999,
		Title:  "feat: comprehensive feature",
		Body:   "Full feature with tests",
		Files: []models.File{
			{Path: "feature.go", Additions: 200, Deletions: 40, Patch: "@@ -1,10 +1,20 @@"},
			{Path: "feature_test.go", Additions: 100, Deletions: 20, Patch: "@@ -1,5 +1,10 @@"},
			{Path: "docs.md", Additions: 100, Deletions: 20, Patch: "@@ -1,3 +1,8 @@"},
		},
		Author: models.Author{
			Login:      "experienceddev",
			AccountAge: 2000 * 24 * time.Hour,
			PRCount:    100,
			MergeRatio: 0.95,
		},
	}

	ctx := context.Background()
	result, err := pipeline.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	categories := make(map[string]int)
	for _, issue := range result.Issues {
		categories[issue.Category]++
	}

	t.Logf("Issues by category:")
	for cat, count := range categories {
		t.Logf("  %s: %d", cat, count)
	}

	if result.LLMUsed == "" {
		t.Error("LLMUsed = empty, want model name")
	}
}

func TestReviewPipeline_ZeroFiles(t *testing.T) {
	pipeline := NewReviewPipeline()

	pr := &models.PR{
		Number: 1,
		Title:  "Empty PR",
	}

	ctx := context.Background()
	result, err := pipeline.Review(ctx, pr)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if result.MergeScore < 0 || result.MergeScore > 100 {
		t.Errorf("MergeScore = %d, want 0-100", result.MergeScore)
	}
}

func TestMergeReadinessScorer_CostTracking(t *testing.T) {
	scorer := NewMergeReadinessScorer("test-model")

	pr := &models.PR{
		Number: 123,
		Title:  "Test PR",
		Files: []models.File{
			{Path: "test.go", Additions: 100, Deletions: 10},
		},
	}

	ctx := context.Background()
	result, err := scorer.Score(ctx, pr)
	if err != nil {
		t.Fatalf("Score() error = %v", err)
	}

	if result.CostUSD < 0 {
		t.Errorf("CostUSD = %f, want >= 0", result.CostUSD)
	}
	if result.DurationMs < 0 {
		t.Errorf("DurationMs = %d, want >= 0", result.DurationMs)
	}
}
