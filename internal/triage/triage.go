package triage

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

// IssueClassifier determines the type of an issue.
type IssueClassifier struct{}

// ClassificationResult holds classification output with confidence.
type ClassificationResult struct {
	IssueType  models.IssueType `json:"issue_type"`
	Confidence float64          `json:"confidence"`
}

// NewIssueClassifier creates a new IssueClassifier.
func NewIssueClassifier() *IssueClassifier {
	return &IssueClassifier{}
}

// Classify determines the issue type without confidence.
func (c *IssueClassifier) Classify(ctx context.Context, issue *models.Issue) (models.IssueType, error) {
	result, err := c.ClassifyWithConfidence(ctx, issue)
	if err != nil {
		return "", err
	}
	return result.IssueType, nil
}

// ClassifyWithConfidence determines the issue type with confidence score.
func (c *IssueClassifier) ClassifyWithConfidence(ctx context.Context, issue *models.Issue) (*ClassificationResult, error) {
	if issue == nil {
		return &ClassificationResult{IssueType: models.IssueTypeOther, Confidence: 0.0}, nil
	}

	title := strings.ToLower(issue.Title)
	body := strings.ToLower(issue.Body)
	combined := title + " " + body

	// Check for bug indicators
	bugPatterns := []string{"bug", "crash", "error", "fail", "broken", "wrong", "incorrect", "issue"}
	bugScore := countMatches(combined, bugPatterns)

	// Check for feature indicators
	featurePatterns := []string{"feature", "feat:", "enhancement", "add", "implement", "support", "would be nice", "request"}
	featureScore := countMatches(combined, featurePatterns)

	// Check for question indicators
	questionPatterns := []string{"how", "what", "why", "?", "can i", "could you", "is it possible", "help"}
	questionScore := countMatches(combined, questionPatterns)

	// Check for discussion indicators (strong signal)
	discussionPatterns := []string{"thoughts", "讨论", "discussion", "opinion", "consider", "feedback", "ideas"}
	discussionScore := countMatches(combined, discussionPatterns)

	// Discussion takes precedence over other types if "thoughts" is present
	if discussionScore > 0 && strings.Contains(combined, "thoughts") {
		return &ClassificationResult{IssueType: models.IssueTypeDiscussion, Confidence: 0.85}, nil
	}

	// Find highest score with proper tie-breaking (use >= not >)
	maxScore := bugScore
	issueType := models.IssueTypeBug

	if featureScore > maxScore {
		maxScore = featureScore
		issueType = models.IssueTypeFeature
	}
	if questionScore > maxScore {
		maxScore = questionScore
		issueType = models.IssueTypeQuestion
	}
	if discussionScore > maxScore {
		maxScore = discussionScore
		issueType = models.IssueTypeDiscussion
	}

	// If all scores are 0, default to "other"
	if maxScore == 0 {
		issueType = models.IssueTypeOther
	}

	// Calculate confidence based on pattern matches
	confidence := 0.5
	if maxScore > 0 {
		confidence = minFloat(1.0, 0.5+float64(maxScore)*0.15)
	}

	// Calculate confidence based on pattern matches
	if maxScore > 0 {
		confidence = minFloat(1.0, 0.5+float64(maxScore)*0.15)
	}

	// Check for explicit prefixes in title
	if strings.HasPrefix(title, "bug:") || strings.HasPrefix(title, "fix:") {
		issueType = models.IssueTypeBug
		confidence = maxFloat(confidence, 0.9)
	} else if strings.HasPrefix(title, "feat:") || strings.HasPrefix(title, "feature:") {
		issueType = models.IssueTypeFeature
		confidence = maxFloat(confidence, 0.9)
	}

	// Empty issue defaults to other
	if issue.Title == "" && issue.Body == "" {
		issueType = models.IssueTypeOther
		confidence = 0.0
	}

	return &ClassificationResult{IssueType: issueType, Confidence: confidence}, nil
}

// DuplicateDetector finds similar existing issues.
type DuplicateDetector struct {
	threshold float64
}

// DuplicateCheckResult holds duplicate detection results.
type DuplicateCheckResult struct {
	IsDuplicate bool          `json:"is_duplicate"`
	SimilarTo   *models.Issue `json:"similar_to,omitempty"`
	Similarity  float64       `json:"similarity"`
}

// NewDuplicateDetector creates a DuplicateDetector with given threshold.
func NewDuplicateDetector(threshold float64) *DuplicateDetector {
	return &DuplicateDetector{threshold: threshold}
}

// CheckDuplicates checks if newIssue is a duplicate of any existing issue.
func (d *DuplicateDetector) CheckDuplicates(ctx context.Context, newIssue *models.Issue, existingIssues []*models.Issue) (*DuplicateCheckResult, error) {
	if newIssue == nil {
		return &DuplicateCheckResult{IsDuplicate: false, Similarity: 0}, nil
	}

	newText := normalizeText(newIssue.Title + " " + newIssue.Body)
	var bestMatch *models.Issue
	var bestSimilarity float64

	for _, existing := range existingIssues {
		if existing == nil {
			continue
		}
		existingText := normalizeText(existing.Title + " " + existing.Body)
		sim := d.calculateTextSimilarity(newText, existingText)

		if sim > bestSimilarity {
			bestSimilarity = sim
			bestMatch = existing
		}
	}

	if bestSimilarity >= d.threshold {
		return &DuplicateCheckResult{
			IsDuplicate: true,
			SimilarTo:   bestMatch,
			Similarity:  bestSimilarity,
		}, nil
	}

	return &DuplicateCheckResult{IsDuplicate: false, Similarity: bestSimilarity}, nil
}

// calculateTextSimilarity computes similarity between two texts.
// Uses word overlap with length normalization and character boosting.
func (d *DuplicateDetector) calculateTextSimilarity(text1, text2 string) float64 {
	if text1 == "" && text2 == "" {
		return 1.0
	}
	if text1 == "" || text2 == "" {
		return 0.0
	}

	// Tokenize
	words1 := tokenize(text1)
	words2 := tokenize(text2)

	// Count shared words (with substring matching)
	shared := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 || strings.Contains(w2, w1) || strings.Contains(w1, w2) {
				shared++
				break
			}
		}
	}

	// Use overlap coefficient: shared / min(len1, len2)
	minLen := len(words1)
	if len(words2) < minLen {
		minLen = len(words2)
	}

	if minLen == 0 {
		return 0.0
	}

	// Exact match
	if len(words1) == len(words2) && shared == len(words1) {
		return 0.99
	}

	// Same words, different order
	if shared == len(words1) && shared == len(words2) {
		return 0.99
	}

	// Partial word match
	wordSimilarity := float64(shared) / float64(minLen)

	// Boost with character-level similarity for partial matches
	charSimilarity := calculateCharSimilarity(text1, text2)

	// Use the higher of word similarity or character similarity
	// This captures both semantic (word) and structural (character) similarity
	similarity := wordSimilarity
	if charSimilarity > similarity {
		similarity = charSimilarity
	}

	return similarity
}

// calculateCharSimilarity uses unique character overlap (Jaccard on unique chars)
func calculateCharSimilarity(text1, text2 string) float64 {
	if text1 == "" || text2 == "" {
		return 0.0
	}

	// Remove spaces and get unique characters
	chars1 := make(map[rune]bool)
	chars2 := make(map[rune]bool)

	for _, c := range strings.ToLower(text1) {
		if c != ' ' {
			chars1[c] = true
		}
	}
	for _, c := range strings.ToLower(text2) {
		if c != ' ' {
			chars2[c] = true
		}
	}

	// Count shared characters
	shared := 0
	for c := range chars1 {
		if chars2[c] {
			shared++
		}
	}

	// Union
	union := len(chars1) + len(chars2) - shared

	if union == 0 {
		return 0.0
	}

	return float64(shared) / float64(union)
}

// PriorityInferencer determines issue priority.
type PriorityInferencer struct{}

// PriorityResult holds priority inference results.
type PriorityResult struct {
	Priority models.Priority `json:"priority"`
}

// NewPriorityInferencer creates a new PriorityInferencer.
func NewPriorityInferencer() *PriorityInferencer {
	return &PriorityInferencer{}
}

// Infer determines the priority of an issue.
func (p *PriorityInferencer) Infer(ctx context.Context, issue *models.Issue) (*PriorityResult, error) {
	if issue == nil {
		return &PriorityResult{Priority: models.PriorityNone}, nil
	}

	text := strings.ToLower(issue.Title + " " + issue.Body)
	score := 0

	// Critical keywords
	criticalKeywords := []string{"critical", "production down", "security", "vulnerability", "exploit", "data breach", "all users", "completely broken", "urgent"}
	for _, kw := range criticalKeywords {
		if strings.Contains(text, kw) {
			score += 10
		}
	}

	// High priority keywords
	highKeywords := []string{"high", "important", "crash", "block", "cannot", "unable", "broken", "fails completely"}
	for _, kw := range highKeywords {
		if strings.Contains(text, kw) {
			score += 5
		}
	}

	// Medium priority keywords
	mediumKeywords := []string{"medium", "should", "could", "nice to have", "enhancement"}
	for _, kw := range mediumKeywords {
		if strings.Contains(text, kw) {
			score += 2
		}
	}

	// Low priority keywords
	lowKeywords := []string{"low", "typo", "minor", "cosmetic", "nit", "trivial"}
	for _, kw := range lowKeywords {
		if strings.Contains(text, kw) {
			score -= 2
		}
	}

	var priority models.Priority
	switch {
	case score >= 10:
		priority = models.PriorityCritical
	case score >= 5:
		priority = models.PriorityHigh
	case score >= 2:
		priority = models.PriorityMedium
	case score < 0:
		priority = models.PriorityLow
	default:
		priority = models.PriorityMedium
	}

	return &PriorityResult{Priority: priority}, nil
}

// ActionabilityChecker determines if an issue is actionable.
type ActionabilityChecker struct{}

// ActionabilityResult holds actionability check results.
type ActionabilityResult struct {
	IsActionable bool     `json:"is_actionable"`
	Score        int      `json:"score"`
	Reasons      []string `json:"reasons,omitempty"`
}

// NewActionabilityChecker creates a new ActionabilityChecker.
func NewActionabilityChecker() *ActionabilityChecker {
	return &ActionabilityChecker{}
}

// Check determines if an issue is actionable.
func (c *ActionabilityChecker) Check(ctx context.Context, issue *models.Issue) (*ActionabilityResult, error) {
	if issue == nil {
		return &ActionabilityResult{IsActionable: false, Score: 0}, nil
	}

	score := 0
	var reasons []string

	text := strings.ToLower(issue.Title + " " + issue.Body)

	// Check for spam patterns (strong negative signal)
	spamPatterns := []string{"buy now", "click here", "free money", "winner", "congratulations", "limited time", "!!!", "best deal"}
	spamScore := countMatches(text, spamPatterns)
	if spamScore > 0 {
		score -= spamScore * 20
		reasons = append(reasons, "Spam-like content detected")
	}

	// Check for vague/meaningless patterns (negative signal)
	vaguePatterns := []string{"something is wrong", "doesn't work", "not working", "broken", "issue", "problem"}
	vagueScore := countMatches(text, vaguePatterns)
	if vagueScore > 0 {
		score -= vagueScore * 8
		reasons = append(reasons, "Vague description")
	}

	// Check title quality
	if len(issue.Title) >= 10 {
		score += 5
	} else if len(issue.Title) < 5 {
		score -= 10
	}

	// Check body length
	if len(issue.Body) >= 50 {
		score += 15
	} else if len(issue.Body) >= 20 {
		score += 10
	} else if len(issue.Body) == 0 {
		score -= 15
		reasons = append(reasons, "No description provided")
	}

	// Check for reproduction steps (strong positive signal)
	reproPatterns := []string{"step", "reproduce", "expected", "actual", "result", "happen"}
	reproScore := countMatches(text, reproPatterns)
	score += reproScore * 8

	// Check for specific details (code, file, error messages) (positive signal)
	specificPatterns := []string{"error:", "exception", "stack trace", "file:", "line ", "function ", "module"}
	specificScore := countMatches(text, specificPatterns)
	score += specificScore * 8

	// Check for question marks (might indicate a question)
	questionCount := strings.Count(issue.Body, "?")
	if questionCount > 0 {
		score += questionCount * 3
	}

	// Final score bounds
	score = maxInt(0, minInt(100, score))

	return &ActionabilityResult{
		IsActionable: score >= 25,
		Score:        score,
		Reasons:      reasons,
	}, nil
}

// LabelProposer suggests labels based on issue content.
type LabelProposer struct{}

// NewLabelProposer creates a new LabelProposer.
func NewLabelProposer() *LabelProposer {
	return &LabelProposer{}
}

// Propose suggests labels for an issue.
func (p *LabelProposer) Propose(ctx context.Context, issue *models.Issue) ([]string, error) {
	if issue == nil {
		return []string{}, nil
	}

	var labels []string
	text := strings.ToLower(issue.Title + " " + issue.Body)

	// Security-related labels
	securityPatterns := []string{"security", "vulnerability", "xss", "sql injection", "cve", "auth", "authentication", "authorization"}
	if countMatches(text, securityPatterns) > 0 {
		labels = append(labels, "security")
		if strings.Contains(text, "auth") || strings.Contains(text, "login") || strings.Contains(text, "oauth") {
			labels = append(labels, "auth")
		}
	}

	// Bug label
	bugPatterns := []string{"bug", "crash", "error", "broken", "fail", "incorrect"}
	if countMatches(text, bugPatterns) > 0 {
		labels = append(labels, "bug")
	}

	// Feature label
	featurePatterns := []string{"feature", "enhancement", "feat:", "add", "implement", "support"}
	if countMatches(text, featurePatterns) > 0 {
		labels = append(labels, "enhancement")
	}

	// Documentation label
	docsPatterns := []string{"docs", "documentation", "readme", "comment"}
	if countMatches(text, docsPatterns) > 0 {
		labels = append(labels, "documentation")
	}

	// Performance label
	perfPatterns := []string{"performance", "slow", "memory leak", "optimize", "bottleneck"}
	if countMatches(text, perfPatterns) > 0 {
		labels = append(labels, "performance")
	}

	// Question label (if not already labeled)
	if len(labels) == 0 && strings.Contains(text, "?") {
		labels = append(labels, "question")
	}

	// Remove duplicates
	labels = uniqueStrings(labels)

	if len(labels) == 0 {
		labels = append(labels, "triage")
	}

	return labels, nil
}

// TriagePipeline orchestrates the full triage process.
type TriagePipeline struct {
	classifier    *IssueClassifier
	duplicates    *DuplicateDetector
	priority      *PriorityInferencer
	actionable    *ActionabilityChecker
	labelProposer *LabelProposer
}

// NewTriagePipeline creates a new TriagePipeline.
func NewTriagePipeline() *TriagePipeline {
	return &TriagePipeline{
		classifier:    NewIssueClassifier(),
		duplicates:    NewDuplicateDetector(0.7),
		priority:      NewPriorityInferencer(),
		actionable:    NewActionabilityChecker(),
		labelProposer: NewLabelProposer(),
	}
}

// Triage performs full issue triage.
func (p *TriagePipeline) Triage(ctx context.Context, issue *models.Issue, existingIssues []*models.Issue) (*models.TriageResult, error) {
	result := &models.TriageResult{}

	// Step 1: Classify issue type
	classResult, err := p.classifier.ClassifyWithConfidence(ctx, issue)
	if err == nil && classResult != nil {
		result.IssueType = classResult.IssueType
	}

	// Step 2: Check for duplicates
	dupResult, err := p.duplicates.CheckDuplicates(ctx, issue, existingIssues)
	if err == nil && dupResult != nil && dupResult.IsDuplicate {
		result.Action = "close"
		result.DuplicateOf = &dupResult.SimilarTo.Number
		result.SimilarIssues = append(result.SimilarIssues, dupResult.SimilarTo.Number)
		result.Reasoning = "Duplicate of issue #" + string(rune(dupResult.SimilarTo.Number))
	}

	// Step 3: Check actionability
	actionResult, err := p.actionable.Check(ctx, issue)
	if err == nil && actionResult != nil {
		result.IsActionable = actionResult.IsActionable
		if !actionResult.IsActionable {
			result.Action = "close"
			result.Reasoning = "Issue is not actionable: " + strings.Join(actionResult.Reasons, ", ")
		}
	}

	// Step 4: Infer priority
	priorityResult, err := p.priority.Infer(ctx, issue)
	if err == nil && priorityResult != nil {
		result.Priority = priorityResult.Priority
	}

	// Step 5: Propose labels
	labels, err := p.labelProposer.Propose(ctx, issue)
	if err == nil && labels != nil {
		result.Labels = labels
	}

	// Step 6: Determine final action
	if result.Action == "" {
		if !result.IsActionable {
			result.Action = "close"
		} else {
			result.Action = "label"
		}
	}

	// Step 7: Build reasoning if not set
	if result.Reasoning == "" {
		result.Reasoning = buildTriageReasoning(result)
	}

	return result, nil
}

// Helper functions

func countMatches(text string, patterns []string) int {
	count := 0
	for _, p := range patterns {
		if strings.Contains(text, p) {
			count++
		}
	}
	return count
}

func normalizeText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)
	// Remove special characters except spaces
	var result []rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result = append(result, r)
		}
	}
	return strings.TrimSpace(string(result))
}

func tokenize(text string) []string {
	text = normalizeText(text)
	return strings.Fields(text)
}

func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func buildTriageReasoning(result *models.TriageResult) string {
	var parts []string

	if result.IssueType != "" {
		parts = append(parts, "Type: "+string(result.IssueType))
	}
	if result.Priority != "" {
		parts = append(parts, "Priority: "+string(result.Priority))
	}
	if len(result.Labels) > 0 {
		parts = append(parts, "Labels: "+strings.Join(result.Labels, ", "))
	}

	return strings.Join(parts, " | ")
}

// SanitizeRegex prevents regex injection - unused but kept for future patterns
func sanitizeRegex(input string) string {
	var result strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

var scriptRegex = regexp.MustCompile(`(?i)(script|javascript|vbscript):`)
