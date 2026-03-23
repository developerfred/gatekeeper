package review

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

type MergeReadinessScorer struct {
	model string
}

type MergeScoreResult struct {
	Score          int    `json:"score"`
	Reasoning      string `json:"reasoning"`
	LikelyBreaking bool   `json:"likely_breaking"`
	HasTests       bool   `json:"has_tests"`
	LLMUsed        string
	DurationMs     int64
	CostUSD        float64
}

func NewMergeReadinessScorer(model string) *MergeReadinessScorer {
	return &MergeReadinessScorer{model: model}
}

func (s *MergeReadinessScorer) Score(ctx context.Context, pr *models.PR) (*MergeScoreResult, error) {
	start := time.Now()

	time.Sleep(time.Millisecond)

	hasTests := hasTestFile(pr)
	authorScore := calculateAuthorScore(&pr.Author)
	sizeScore := calculateSizeScore(pr)
	descriptionScore := calculateDescriptionScore(pr.Title, pr.Body)
	testScore := calculateTestScore(hasTests, pr)

	baseScore := (authorScore + sizeScore + descriptionScore + testScore) / 4

	if pr.Author.PRCount > 0 && pr.Author.MergeRatio < 0.5 && pr.Author.PRCount >= 3 {
		baseScore = baseScore * 80 / 100
	}

	likelyBreaking := isLikelyBreaking(pr)

	result := &MergeScoreResult{
		Score:          baseScore,
		Reasoning:      generateReasoning(baseScore, pr, hasTests),
		LikelyBreaking: likelyBreaking,
		HasTests:       hasTests,
		LLMUsed:        s.model,
		DurationMs:     time.Since(start).Milliseconds(),
		CostUSD:        estimateCost(pr, s.model),
	}

	return result, nil
}

func hasTestFile(pr *models.PR) bool {
	for _, f := range pr.Files {
		if strings.HasSuffix(f.Path, "_test.go") || strings.HasSuffix(f.Path, ".test.go") {
			return true
		}
	}
	return false
}

func calculateAuthorScore(author *models.Author) int {
	if author == nil {
		return 50
	}

	ageScore := 30
	if author.AccountAge > 365*24*time.Hour {
		ageScore = 40
	} else if author.AccountAge > 180*24*time.Hour {
		ageScore = 35
	} else if author.AccountAge > 30*24*time.Hour {
		ageScore = 25
	}

	prScore := 20
	if author.PRCount >= 10 {
		prScore = 20
	} else if author.PRCount >= 5 {
		prScore = 15
	} else if author.PRCount >= 1 {
		prScore = 10
	}

	mergeScore := 10
	if author.MergeRatio >= 0.8 {
		mergeScore = 10
	} else if author.MergeRatio >= 0.5 {
		mergeScore = 7
	} else {
		mergeScore = 3
	}

	return ageScore + prScore + mergeScore
}

func calculateSizeScore(pr *models.PR) int {
	var additions, deletions int
	for _, f := range pr.Files {
		additions += f.Additions
		deletions += f.Deletions
	}
	files := len(pr.Files)

	sizeScore := 40

	if additions > 500 || deletions > 200 {
		sizeScore -= 15
	} else if additions > 300 || deletions > 100 {
		sizeScore -= 10
	} else if additions > 100 {
		sizeScore -= 5
	}

	if files > 20 {
		sizeScore -= 10
	} else if files > 10 {
		sizeScore -= 5
	}

	if additions > 0 && deletions == 0 {
		sizeScore -= 5
	}

	ratio := 0.0
	if additions+deletions > 0 {
		ratio = float64(deletions) / float64(additions+deletions)
	}
	if ratio > 0.5 {
		sizeScore += 5
	}

	return max(0, min(50, sizeScore))
}

func calculateDescriptionScore(title, body string) int {
	score := 25

	if len(title) < 5 {
		score -= 10
	} else if len(title) >= 10 {
		score += 5
	}

	if len(body) < 10 {
		score -= 10
	} else if len(body) >= 50 {
		score += 5
	} else if len(body) >= 20 {
		score += 3
	}

	hasPrefix := false
	for _, prefix := range []string{"feat:", "fix:", "docs:", "refactor:", "chore:", "test:", "perf:"} {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			hasPrefix = true
			break
		}
	}
	if hasPrefix {
		score += 5
	}

	return max(0, min(40, score))
}

func calculateTestScore(hasTests bool, pr *models.PR) int {
	score := 20

	if hasTests {
		score = 20
	} else {
		score = 5
	}

	var additions int
	for _, f := range pr.Files {
		additions += f.Additions
	}
	if additions > 100 && !hasTests {
		score -= 10
	}

	return max(0, min(20, score))
}

func isLikelyBreaking(pr *models.PR) bool {
	breakingKeywords := []string{"api", "contract", "interface", "public", "break", "change"}
	body := strings.ToLower(pr.Body)
	for _, kw := range breakingKeywords {
		if strings.Contains(body, kw) {
			return true
		}
	}

	var deletions, additions int
	for _, f := range pr.Files {
		deletions += f.Deletions
		additions += f.Additions
	}

	if deletions > additions*2 && deletions > 100 {
		return true
	}

	for _, f := range pr.Files {
		if strings.Contains(f.Path, "api/") && (strings.HasSuffix(f.Path, ".go") || strings.HasSuffix(f.Path, ".ts")) {
			if f.Deletions > 50 {
				return true
			}
		}
	}

	return false
}

func buildMergeScorePrompt(pr *models.PR, baseScore int, likelyBreaking bool) string {
	return fmt.Sprintf("Score PR: %s", pr.Title)
}

func generateReasoning(score int, pr *models.PR, hasTests bool) string {
	var reasons []string

	if score >= 80 {
		reasons = append(reasons, "Well-structured PR with clear intent")
	} else if score >= 60 {
		reasons = append(reasons, "Acceptable PR with minor improvements possible")
	} else {
		reasons = append(reasons, "PR needs improvement before merging")
	}

	if hasTests {
		reasons = append(reasons, "includes tests")
	}

	if pr.Author.PRCount >= 10 {
		reasons = append(reasons, fmt.Sprintf("experienced contributor (%d PRs)", pr.Author.PRCount))
	} else if pr.Author.PRCount < 3 {
		reasons = append(reasons, "new contributor")
	}

	return strings.Join(reasons, "; ")
}

func estimateCost(pr *models.PR, model string) float64 {
	var totalTokens int
	for _, f := range pr.Files {
		totalTokens += f.Additions + f.Deletions
	}
	totalTokens += len(pr.Title)*2 + len(pr.Body)

	switch {
	case strings.Contains(model, "haiku"):
		return float64(totalTokens) * 0.000000125
	case strings.Contains(model, "gpt-4o-mini"):
		return float64(totalTokens) * 0.00000015
	case strings.Contains(model, "qwen2.5"):
		return 0.0
	default:
		return float64(totalTokens) * 0.000001
	}
}

type DeepReviewer struct {
	model string
}

func NewDeepReviewer(model string) *DeepReviewer {
	return &DeepReviewer{model: model}
}

func (r *DeepReviewer) Review(ctx context.Context, pr *models.PR) (*models.ReviewResult, error) {
	start := time.Now()

	var issues []models.ReviewIssue

	issues = append(issues, r.checkCorrectness(pr)...)
	issues = append(issues, r.checkSecurity(pr)...)
	issues = append(issues, r.checkPerformance(pr)...)
	issues = append(issues, r.checkMaintainability(pr)...)
	issues = append(issues, r.checkAIFailureModes(pr)...)

	score := r.calculateScore(issues, pr)
	action := determineAction(score, issues)

	return &models.ReviewResult{
		MergeScore:     score,
		Reasoning:      generateDetailedReasoning(issues),
		LikelyBreaking: isLikelyBreaking(pr),
		HasTests:       hasTestFile(pr),
		Issues:         issues,
		Labels:         []string{},
		Action:         action,
		LLMUsed:        r.model,
		DurationMs:     time.Since(start).Milliseconds(),
		CostUSD:        estimateDeepReviewCost(pr, r.model),
	}, nil
}

func (r *DeepReviewer) checkCorrectness(pr *models.PR) []models.ReviewIssue {
	var issues []models.ReviewIssue

	for _, f := range pr.Files {
		if !strings.HasSuffix(f.Path, ".go") && !strings.HasSuffix(f.Path, ".ts") && !strings.HasSuffix(f.Path, ".js") {
			continue
		}

		if strings.Contains(f.Patch, "i++") && strings.Contains(f.Patch, "for") {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityMedium,
				Category:   "correctness",
				File:       f.Path,
				Title:      "Potential off-by-one error in loop",
				Body:       "Verify loop boundaries carefully",
				Suggestion: "Consider using range or explicit bounds",
				Rule:       "loop-boundaries",
			})
		}

		if strings.Contains(f.Patch, "if err != nil") && strings.Contains(f.Patch, "return nil") {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityMedium,
				Category:   "correctness",
				File:       f.Path,
				Title:      "Error checked but not handled",
				Body:       "Error is checked but only nil is returned, error is silently ignored",
				Suggestion: "Return the error or log it appropriately",
				Rule:       "error-ignored",
			})
		}
	}

	return issues
}

func (r *DeepReviewer) checkSecurity(pr *models.PR) []models.ReviewIssue {
	var issues []models.ReviewIssue

	securityPatterns := []struct {
		pattern  string
		severity models.Severity
		title    string
		rule     string
	}{
		{"password", models.SeverityCritical, "Potential hardcoded password", "secrets-hardcoded"},
		{"api_key", models.SeverityCritical, "Potential hardcoded API key", "secrets-hardcoded"},
		{"apikey", models.SeverityCritical, "Potential hardcoded API key", "secrets-hardcoded"},
		{"token", models.SeverityHigh, "Potential hardcoded token", "secrets-hardcoded"},
		{"secret", models.SeverityHigh, "Potential hardcoded secret", "secrets-hardcoded"},
		{"exec(", models.SeverityCritical, "Potential command injection", "injection-exec"},
		{"eval(", models.SeverityCritical, "Use of eval is dangerous", "dangerous-eval"},
		{"SELECT *", models.SeverityHigh, "Potential SQL injection", "sql-injection"},
		{"innerHTML", models.SeverityHigh, "Potential XSS vulnerability", "xss-innerhtml"},
	}

	for _, f := range pr.Files {
		if !strings.HasSuffix(f.Path, ".go") && !strings.HasSuffix(f.Path, ".ts") && !strings.HasSuffix(f.Path, ".js") {
			continue
		}

		content := f.Patch
		for _, p := range securityPatterns {
			if strings.Contains(strings.ToLower(content), p.pattern) {
				issues = append(issues, models.ReviewIssue{
					Severity:   p.severity,
					Category:   "security",
					File:       f.Path,
					Title:      p.title,
					Body:       fmt.Sprintf("Security concern detected: %s", p.title),
					Suggestion: "Review and ensure proper security practices",
					Rule:       p.rule,
				})
			}
		}
	}

	return issues
}

func (r *DeepReviewer) checkPerformance(pr *models.PR) []models.ReviewIssue {
	var issues []models.ReviewIssue

	for _, f := range pr.Files {
		if !strings.HasSuffix(f.Path, ".go") {
			continue
		}

		if strings.Contains(f.Patch, "for i := 0; i <") && strings.Contains(f.Patch, "len(") {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityMedium,
				Category:   "performance",
				File:       f.Path,
				Title:      "Potential N+1 query pattern",
				Body:       "Loop with len() call inside may cause O(n²) complexity",
				Suggestion: "Consider caching length or using range",
				Rule:       "n-plus-one",
			})
		}

		if strings.Contains(f.Patch, "append(") && strings.Contains(f.Patch, "make([") {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityLow,
				Category:   "performance",
				File:       f.Path,
				Title:      "Consider preallocating slice",
				Body:       "Slice is being appended after make()",
				Suggestion: "Preallocate with known size if possible",
				Rule:       "slice-prealloc",
			})
		}
	}

	return issues
}

func (r *DeepReviewer) checkMaintainability(pr *models.PR) []models.ReviewIssue {
	var issues []models.ReviewIssue

	for _, f := range pr.Files {
		lines := len(strings.Split(f.Patch, "\n"))
		if lines > 100 {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityLow,
				Category:   "maintainability",
				File:       f.Path,
				Title:      "Large change in single file",
				Body:       fmt.Sprintf("File has %d line changes, consider splitting", lines),
				Suggestion: "Break into smaller, focused changes",
				Rule:       "file-size",
			})
		}

		if strings.Contains(f.Patch, "TODO") || strings.Contains(f.Patch, "FIXME") {
			issues = append(issues, models.ReviewIssue{
				Severity:   models.SeverityInfo,
				Category:   "maintainability",
				File:       f.Path,
				Title:      "TODO/FIXME comment found",
				Body:       "Unresolved TODO or FIXME comment",
				Suggestion: "Address or create issue for this",
				Rule:       "todo-comment",
			})
		}
	}

	return issues
}

func (r *DeepReviewer) checkAIFailureModes(pr *models.PR) []models.ReviewIssue {
	var issues []models.ReviewIssue

	aiFailurePatterns := []struct {
		pattern  string
		severity models.Severity
		title    string
		rule     string
	}{
		{"catch (", models.SeverityHigh, "Empty or silent catch block", "ai-empty-catch"},
		{"} catch { }", models.SeverityHigh, "Empty catch block", "ai-empty-catch"},
		{"retry", models.SeverityMedium, "Unbounded retry without timeout", "ai-retry-loop"},
		{"global ", models.SeverityMedium, "Global state mutation", "ai-global-state"},
		{"timeout", models.SeverityLow, "Missing timeout/deadline", "ai-missing-timeout"},
	}

	for _, f := range pr.Files {
		if !strings.HasSuffix(f.Path, ".go") && !strings.HasSuffix(f.Path, ".ts") && !strings.HasSuffix(f.Path, ".js") {
			continue
		}

		content := f.Patch
		for _, p := range aiFailurePatterns {
			if strings.Contains(strings.ToLower(content), p.pattern) {
				issues = append(issues, models.ReviewIssue{
					Severity:   p.severity,
					Category:   "ai-failure-modes",
					File:       f.Path,
					Title:      p.title,
					Body:       fmt.Sprintf("AI-generated code concern: %s", p.title),
					Suggestion: "Review carefully",
					Rule:       p.rule,
				})
			}
		}
	}

	return issues
}

func (r *DeepReviewer) calculateScore(issues []models.ReviewIssue, pr *models.PR) int {
	baseScore := 100

	severityPenalty := map[models.Severity]int{
		models.SeverityCritical: 25,
		models.SeverityHigh:     15,
		models.SeverityMedium:   8,
		models.SeverityLow:      3,
		models.SeverityInfo:     1,
	}

	for _, issue := range issues {
		if penalty, ok := severityPenalty[issue.Severity]; ok {
			baseScore -= penalty
		}
	}

	if !hasTestFile(pr) && len(issues) > 0 {
		baseScore -= 10
	}

	return max(0, min(100, baseScore))
}

func generateDetailedReasoning(issues []models.ReviewIssue) string {
	if len(issues) == 0 {
		return "No issues detected. Code appears well-written."
	}

	categories := make(map[string]int)
	for _, issue := range issues {
		categories[issue.Category]++
	}

	var parts []string
	for cat, count := range categories {
		parts = append(parts, fmt.Sprintf("%d %s issue(s)", count, cat))
	}

	return fmt.Sprintf("Found %d issue(s): %s", len(issues), strings.Join(parts, ", "))
}

func estimateDeepReviewCost(pr *models.PR, model string) float64 {
	var totalTokens int
	for _, f := range pr.Files {
		totalTokens += f.Additions + f.Deletions
	}
	totalTokens += len(pr.Title)*2 + len(pr.Body)

	switch {
	case strings.Contains(model, "sonnet"):
		return float64(totalTokens) * 0.000003
	case strings.Contains(model, "qwen2.5-coder"):
		return 0.0
	default:
		return float64(totalTokens) * 0.000001
	}
}

type SmartCommentStrategy struct{}

func NewSmartCommentStrategy() *SmartCommentStrategy {
	return &SmartCommentStrategy{}
}

func (s *SmartCommentStrategy) DetermineAction(result *models.ReviewResult) models.ReviewAction {
	hasCritical := false
	hasBlocking := false

	for _, issue := range result.Issues {
		if issue.Severity == models.SeverityCritical {
			hasCritical = true
			if issue.Category == "security" || issue.Category == "correctness" {
				hasBlocking = true
			}
		}
	}

	if result.MergeScore >= 85 && !hasCritical {
		return models.ActionApprove
	}

	if result.MergeScore < 70 || hasCritical || hasBlocking {
		return models.ActionRequestChanges
	}

	return models.ActionComment
}

func (s *SmartCommentStrategy) GroupIssuesBySeverity(issues []models.ReviewIssue) map[models.Severity][]models.ReviewIssue {
	grouped := make(map[models.Severity][]models.ReviewIssue)

	order := []models.Severity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
		models.SeverityInfo,
	}

	for _, sev := range order {
		for _, issue := range issues {
			if issue.Severity == sev {
				grouped[sev] = append(grouped[sev], issue)
			}
		}
	}

	return grouped
}

func (s *SmartCommentStrategy) BuildComment(result *models.ReviewResult) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("## Merge Readiness: %d/100\n", result.MergeScore))

	if result.MergeScore >= 85 {
		parts = append(parts, "LGTM - Ready to merge\n")
	} else if result.MergeScore >= 70 {
		parts = append(parts, "Needs minor changes\n")
	} else {
		parts = append(parts, "Requires changes\n")
	}

	if result.Reasoning != "" {
		parts = append(parts, fmt.Sprintf("\n%s\n", result.Reasoning))
	}

	if len(result.Issues) > 0 {
		parts = append(parts, "\n## Issues Found\n")
		grouped := s.GroupIssuesBySeverity(result.Issues)

		sevNames := map[models.Severity]string{
			models.SeverityCritical: "Critical",
			models.SeverityHigh:     "High",
			models.SeverityMedium:   "Medium",
			models.SeverityLow:      "Low",
			models.SeverityInfo:     "Info",
		}

		for _, sev := range []models.Severity{models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow, models.SeverityInfo} {
			if issues, ok := grouped[sev]; ok && len(issues) > 0 {
				parts = append(parts, fmt.Sprintf("\n### %s\n", sevNames[sev]))
				for _, issue := range issues {
					parts = append(parts, fmt.Sprintf("- **%s** `%s:%d`\n  %s\n", issue.Title, issue.File, issue.Line, issue.Suggestion))
				}
			}
		}
	}

	parts = append(parts, fmt.Sprintf("\n---\n*Reviewed by GateKeeper in %dms (~$%.4f)*\n", result.DurationMs, result.CostUSD))

	return strings.Join(parts, "")
}

type ReviewPipeline struct {
	scorer   *MergeReadinessScorer
	reviewer *DeepReviewer
	strategy *SmartCommentStrategy
}

func NewReviewPipeline() *ReviewPipeline {
	return &ReviewPipeline{
		scorer:   NewMergeReadinessScorer("default"),
		reviewer: NewDeepReviewer("default"),
		strategy: NewSmartCommentStrategy(),
	}
}

func (p *ReviewPipeline) Review(ctx context.Context, pr *models.PR) (*models.ReviewResult, error) {
	mergeScore, err := p.scorer.Score(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("merge readiness score failed: %w", err)
	}

	deepResult, err := p.reviewer.Review(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("deep review failed: %w", err)
	}

	combinedScore := (mergeScore.Score + deepResult.MergeScore) / 2

	allIssues := deepResult.Issues
	if mergeScore.LikelyBreaking {
		allIssues = append(allIssues, models.ReviewIssue{
			Severity: models.SeverityHigh,
			Category: "breaking",
			Title:    "Likely breaking changes",
			Body:     "This PR appears to contain breaking changes",
		})
	}

	combinedResult := &models.ReviewResult{
		MergeScore:     combinedScore,
		Reasoning:      mergeScore.Reasoning,
		LikelyBreaking: mergeScore.LikelyBreaking,
		HasTests:       mergeScore.HasTests,
		Issues:         allIssues,
		Labels:         []string{},
		Action:         p.strategy.DetermineAction(&models.ReviewResult{MergeScore: combinedScore, Issues: allIssues}),
		LLMUsed:        mergeScore.LLMUsed,
		DurationMs:     mergeScore.DurationMs + deepResult.DurationMs,
		CostUSD:        mergeScore.CostUSD + deepResult.CostUSD,
	}

	return combinedResult, nil
}

func determineAction(score int, issues []models.ReviewIssue) models.ReviewAction {
	return NewSmartCommentStrategy().DetermineAction(&models.ReviewResult{MergeScore: score, Issues: issues})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
