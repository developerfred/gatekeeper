package anti_slop

import (
	"testing"
	"time"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

func TestChecker_PassesLegitPR(t *testing.T) {
	c := NewChecker(70)

	patch := `+func ValidateToken(token string) error {
+	if token == "" {
+		return ErrEmptyToken
+	}
+	claims, err := ParseJWT(token)
+	if err != nil {
+		return err
+	}
+	return nil
+}`

	pr := &models.PR{
		Number: 1,
		Title:  "Add user authentication with JWT tokens",
		Body:   "This PR implements JWT-based authentication for the API endpoints. It includes token generation, validation, and refresh logic.",
		Author: models.Author{
			Login:       "senior-dev",
			AccountAge:  2 * 365 * 24 * time.Hour,
			PRCount:     50,
			MergeRatio:  0.85,
			HasBio:      true,
			HasLocation: true,
		},
		HeadBranch: "feat/jwt-auth",
		Files: []models.File{
			{Path: "auth/jwt.go", Additions: 150, Deletions: 10, Patch: patch},
			{Path: "auth/jwt_test.go", Additions: 80, Deletions: 0, Patch: "+func TestValidateToken(t *testing.T) {}\n"},
		},
		Commits: []models.Commit{
			{Message: "Add JWT token generation and validation"},
		},
	}

	result := c.Check(pr)

	if !result.Passed {
		t.Errorf("Expected legit PR to pass, got score %d, failures: %v", result.Score, result.Failures)
	}
	if result.Score < 70 {
		t.Errorf("Expected score >= 70 for legit PR, got %d", result.Score)
	}
}

func TestChecker_RejectsEmptyTitle(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title: "Fix",
		Body:  "This PR fixes something",
		Author: models.Author{
			Login:       "newuser",
			AccountAge:  2 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		HeadBranch: "fix-ai-bug",
	}

	result := c.Check(pr)

	if result.Passed {
		t.Error("Expected PR with short title to fail")
	}
	if !contains(result.Failures, "title_length") {
		t.Error("Expected title_length failure")
	}
}

func TestChecker_RejectsBlockedBranchName(t *testing.T) {
	c := NewChecker(70)

	patch := `+func FixAuth() error {
+	return nil
+}`

	pr := &models.PR{
		Title:      "Fix authentication issue",
		Body:       "Detailed description of the fix",
		HeadBranch: "fix-ai-auth-bug",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "auth.go", Additions: 50, Deletions: 10, Patch: patch},
		},
	}

	result := c.Check(pr)

	if result.Passed {
		t.Error("Expected PR with blocked branch name to fail")
	}
	if !contains(result.Failures, "branch_name") {
		t.Errorf("Expected branch_name failure, got %v", result.Failures)
	}
}

func TestChecker_RejectsEmptyDescription(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Add new feature",
		Body:       "",
		HeadBranch: "feat/new-feature",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 50},
		},
	}

	result := c.Check(pr)

	if !contains(result.Failures, "description_length") {
		t.Error("Expected description_length failure")
	}
	if result.Score > 80 {
		t.Errorf("Expected score <= 80 after description_length penalty, got %d", result.Score)
	}
}

func TestChecker_DetectsWhitespaceOnlyDiff(t *testing.T) {
	c := NewChecker(70)

	patch := ""
	for i := 0; i < 50; i++ {
		patch += "+   \n"
	}

	pr := &models.PR{
		Title:      "Fix formatting",
		Body:       "This PR fixes formatting issues",
		HeadBranch: "fix/whitespace",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "main.go", Additions: 500, Deletions: 500, Patch: patch},
		},
	}

	result := c.Check(pr)

	if len(result.Failures) == 0 {
		t.Error("Expected at least one failure for whitespace-only diff")
	}
}

func TestChecker_DetectsNewAccountWithLowMergeRatio(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Add feature",
		Body:       "This is a detailed description of the feature being added",
		HeadBranch: "feat/new-feature",
		Author: models.Author{
			Login:       "newuser123",
			AccountAge:  3 * 24 * time.Hour,
			PRCount:     5,
			MergeRatio:  0.1,
			HasBio:      false,
			HasLocation: false,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 50},
		},
	}

	result := c.Check(pr)

	if !contains(result.Warnings, "account_age") {
		t.Error("Expected account_age warning, got:", result.Warnings)
	}
	if !contains(result.Warnings, "merge_ratio") {
		t.Error("Expected merge_ratio warning, got:", result.Warnings)
	}
}

func TestChecker_DetectsHoneypotViolation(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Add feature",
		Body:       "This PR adds a new feature [[AI_GENERATED]]. Let me know if you have questions.",
		HeadBranch: "feat/new-feature",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 50},
		},
	}

	result := c.Check(pr)

	if result.Passed {
		t.Error("Expected PR with honeypot violation to fail")
	}
	if !contains(result.Failures, "honeypot") {
		t.Error("Expected honeypot failure")
	}
}

func TestChecker_GenericCommitMessage(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Update",
		Body:       "Made some updates to the codebase",
		HeadBranch: "update/something",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Commits: []models.Commit{
			{Message: "update"},
		},
		Files: []models.File{
			{Path: "file.go", Additions: 50},
		},
	}

	result := c.Check(pr)

	if !contains(result.Warnings, "commit_message") {
		t.Error("Expected commit_message warning for generic message")
	}
}

func TestChecker_ThresholdBoundary(t *testing.T) {
	c := NewChecker(70)

	patch := `+func ProcessData(input string) error {
+	result := transform(input)
+	return save(result)
+}`

	pr := &models.PR{
		Title:      "Reasonable title here",
		Body:       "This is a reasonable description with enough content to pass basic checks",
		HeadBranch: "feature/my-feature",
		Author: models.Author{
			Login:       "trusteduser",
			AccountAge:  3 * 365 * 24 * time.Hour,
			PRCount:     100,
			MergeRatio:  0.9,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 100, Deletions: 20, Patch: patch},
		},
		Commits: []models.Commit{
			{Message: "Add comprehensive feature implementation"},
		},
	}

	result := c.Check(pr)

	if result.Score < 70 && result.Passed {
		t.Errorf("Expected PR to pass with score >= 70, got score %d", result.Score)
	}
}

func TestChecker_EmptyPR(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "",
		Body:       "",
		HeadBranch: "",
		Author:     models.Author{},
		Files:      []models.File{},
	}

	result := c.Check(pr)

	if result.Passed {
		t.Error("Expected empty PR to fail")
	}
	if result.Score >= 50 {
		t.Errorf("Expected low score for empty PR, got %d", result.Score)
	}
}

func TestChecker_NoCodeFiles(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Update README",
		Body:       "Updated the documentation with new instructions",
		HeadBranch: "docs/readme",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "README.md", Additions: 50, Deletions: 10},
		},
	}

	result := c.Check(pr)

	if !contains(result.Warnings, "files_code") {
		t.Error("Expected files_code warning for README-only PR")
	}
}

func TestChecker_DiffImportsOnly(t *testing.T) {
	c := NewChecker(70)

	patch := `+import "fmt"
+import "os"
+import "time"`

	pr := &models.PR{
		Title:      "Add dependencies",
		Body:       "Added new imports for the dependencies",
		HeadBranch: "deps/add-lib",
		Author: models.Author{
			Login:       "testuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			MergeRatio:  0.5,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "main.go", Additions: 3, Deletions: 0, Patch: patch},
		},
	}

	result := c.Check(pr)

	if !contains(result.Failures, "diff_imports_only") {
		t.Error("Expected diff_imports_only failure, got:", result.Failures)
	}
}

func TestChecker_IncompleteProfile(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Add feature",
		Body:       "This is a detailed description of the feature being added to the codebase",
		HeadBranch: "feat/new-feature",
		Author: models.Author{
			Login:       "mysteryuser",
			AccountAge:  1 * 365 * 24 * time.Hour,
			PRCount:     10,
			MergeRatio:  0.5,
			HasBio:      false,
			HasLocation: false,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 50},
		},
		Commits: []models.Commit{
			{Message: "Add new feature"},
		},
	}

	result := c.Check(pr)

	if !contains(result.Warnings, "author_profile") {
		t.Error("Expected author_profile warning")
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func BenchmarkChecker_Check(b *testing.B) {
	c := NewChecker(70)

	pr := &models.PR{
		Number:     1,
		Title:      "Add user authentication with JWT tokens",
		Body:       "This PR implements JWT-based authentication for the API endpoints.",
		HeadBranch: "feat/jwt-auth",
		Author: models.Author{
			Login:       "senior-dev",
			AccountAge:  2 * 365 * 24 * time.Hour,
			PRCount:     50,
			MergeRatio:  0.85,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "auth/jwt.go", Additions: 150, Deletions: 10, Patch: "code changes"},
		},
		Commits: []models.Commit{
			{Message: "Add JWT token generation"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(pr)
	}
}

func TestNewChecker(t *testing.T) {
	c := NewChecker(70)
	if c == nil {
		t.Error("NewChecker should not return nil")
	}
	if len(c.rules) == 0 {
		t.Error("Checker should have default rules")
	}
	if c.threshold != 70 {
		t.Errorf("Expected threshold 70, got %d", c.threshold)
	}
}

func TestCheckerWithCustomThreshold(t *testing.T) {
	c := NewChecker(50)
	if c.threshold != 50 {
		t.Errorf("Expected threshold 50, got %d", c.threshold)
	}
}

func TestChecker_AddRule(t *testing.T) {
	c := NewChecker(70)
	initialCount := len(c.rules)

	c.AddRule(&branchNameRule{})

	if len(c.rules) != initialCount+1 {
		t.Error("Expected AddRule to increase rule count by 1")
	}
}

func TestChecker_ScoreClamping(t *testing.T) {
	c := NewChecker(0)

	pr := &models.PR{
		Title:      "",
		Body:       "",
		HeadBranch: "fix-ai-xxx",
		Author:     models.Author{},
		Files:      []models.File{},
		Commits:    []models.Commit{},
	}

	result := c.Check(pr)

	if result.Score > 100 {
		t.Errorf("Score should not exceed 100, got %d", result.Score)
	}
	if result.Score < 0 {
		t.Errorf("Score should not be negative, got %d", result.Score)
	}
}

func TestChecker_FillerWords(t *testing.T) {
	c := NewChecker(70)

	testCases := []struct {
		name string
		body string
		fail bool
	}{
		{"normal description", "This PR adds user authentication with JWT tokens and proper error handling.", false},
		{"filler phrase", "This PR I made some changes to the codebase", true},
		{"another filler", "Here are the changes for the pull request", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pr := &models.PR{
				Title:      "Add feature",
				Body:       tc.body,
				HeadBranch: "feat/feature",
				Author: models.Author{
					Login:       "testuser",
					AccountAge:  1 * 365 * 24 * time.Hour,
					MergeRatio:  0.5,
					HasBio:      true,
					HasLocation: true,
				},
				Files: []models.File{
					{Path: "feature.go", Additions: 50},
				},
			}

			result := c.Check(pr)
			hasWarning := contains(result.Warnings, "description_template")
			if tc.fail && !hasWarning {
				t.Error("Expected description_template warning")
			}
		})
	}
}

func BenchmarkChecker_CheckSlop(b *testing.B) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Fix",
		Body:       "Fix bug",
		HeadBranch: "fix-ai-bug",
		Author: models.Author{
			Login:       "newuser",
			AccountAge:  1 * 24 * time.Hour,
			PRCount:     1,
			MergeRatio:  0.0,
			HasBio:      false,
			HasLocation: false,
		},
		Files: []models.File{
			{Path: "README.md", Additions: 100},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Check(pr)
	}
}

func TestChecker_MultipleFailures(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "a",
		Body:       "x",
		HeadBranch: "fix-ai-xxx",
		Author: models.Author{
			Login:       "new",
			AccountAge:  1 * 24 * time.Hour,
			MergeRatio:  0.1,
			HasBio:      false,
			HasLocation: false,
		},
		Files: []models.File{},
	}

	result := c.Check(pr)

	if len(result.Failures) < 2 {
		t.Errorf("Expected multiple failures, got %v", result.Failures)
	}
}

func TestChecker_ScoreCalculation(t *testing.T) {
	c := NewChecker(70)

	basePR := &models.PR{
		Title:      "Add comprehensive feature with detailed implementation",
		Body:       "This PR implements a new feature with full test coverage and documentation.",
		HeadBranch: "feat/new-feature",
		Author: models.Author{
			Login:       "senior",
			AccountAge:  5 * 365 * 24 * time.Hour,
			PRCount:     200,
			MergeRatio:  0.95,
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "feature.go", Additions: 200, Deletions: 20, Patch: "substantial code"},
		},
		Commits: []models.Commit{
			{Message: "Implement new feature with tests and docs"},
		},
	}

	cleanResult := c.Check(basePR)

	newPR := *basePR
	newPR.Title = "a"
	badResult := c.Check(&newPR)

	if badResult.Score >= cleanResult.Score {
		t.Error("Bad PR should have lower score than clean PR")
	}
}

func TestChecker_HiddenMarkdownMarkers(t *testing.T) {
	c := NewChecker(70)

	markers := []string{"[[", "]]", "<!--", "-->", "{{", "}}"}

	for _, marker := range markers {
		pr := &models.PR{
			Title:      "Add feature",
			Body:       "Description " + marker + " hidden text",
			HeadBranch: "feat/feature",
			Author: models.Author{
				Login:       "testuser",
				AccountAge:  1 * 365 * 24 * time.Hour,
				MergeRatio:  0.5,
				HasBio:      true,
				HasLocation: true,
			},
			Files: []models.File{
				{Path: "feature.go", Additions: 50},
			},
		}

		result := c.Check(pr)

		if !contains(result.Failures, "honeypot") {
			t.Errorf("Expected honeypot failure for marker %s", marker)
		}
	}
}

func TestResultReasonsMap(t *testing.T) {
	c := NewChecker(70)

	pr := &models.PR{
		Title:      "Fix",
		Body:       "Fix",
		HeadBranch: "fix-ai-x",
		Author:     models.Author{},
		Files:      []models.File{},
	}

	result := c.Check(pr)

	if result.Reasons == nil {
		t.Error("Reasons map should not be nil")
	}

	for _, failure := range result.Failures {
		if _, ok := result.Reasons[failure]; !ok {
			t.Errorf("Failure %s should have a reason", failure)
		}
	}
}
