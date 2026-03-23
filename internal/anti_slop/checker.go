package anti_slop

import (
	"regexp"
	"strings"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

type Rule interface {
	Name() string
	Check(pr *models.PR, result *models.AntiSlopResult)
}

type Checker struct {
	rules     []Rule
	threshold int
}

func NewChecker(threshold int) *Checker {
	c := &Checker{
		threshold: threshold,
		rules:     []Rule{},
	}
	c.rules = append(c.rules,
		newBranchNameRule(),
		&titleLengthRule{},
		&descriptionLengthRule{},
		&descriptionTemplateRule{},
		&diffWhitespaceOnlyRule{},
		&diffImportsOnlyRule{},
		&accountAgeRule{},
		&mergeRatioRule{},
		&authorProfileRule{},
		&pastPRQualityRule{},
		&filesCodeRule{},
		newHoneypotRule(),
		&commitMessageRule{},
	)
	return c
}

func (c *Checker) Check(pr *models.PR) *models.AntiSlopResult {
	result := &models.AntiSlopResult{
		Score:   100,
		Reasons: make(map[string]string),
	}

	for _, rule := range c.rules {
		rule.Check(pr, result)
	}

	result.Passed = result.Score >= c.threshold
	return result
}

func (c *Checker) AddRule(rule Rule) {
	c.rules = append(c.rules, rule)
}

type branchNameRule struct {
	blockedPatterns []*regexp.Regexp
}

func newBranchNameRule() *branchNameRule {
	return &branchNameRule{
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)^(fix|update)-ai-.*`),
			regexp.MustCompile(`(?i)^ai-.*-fix$`),
			regexp.MustCompile(`(?i)^patch-\d+$`),
		},
	}
}

func (r *branchNameRule) Name() string { return "branch_name" }

func (r *branchNameRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	for _, pattern := range r.blockedPatterns {
		if pattern.MatchString(pr.HeadBranch) {
			result.Failures = append(result.Failures, r.Name())
			result.Reasons[r.Name()] = "Branch name matches blocked AI-generated pattern"
			result.Score -= 30
			return
		}
	}
}

type titleLengthRule struct{}

func (r *titleLengthRule) Name() string { return "title_length" }

func (r *titleLengthRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if len(pr.Title) < 10 {
		result.Failures = append(result.Failures, r.Name())
		result.Reasons[r.Name()] = "Title too short (less than 10 characters)"
		result.Score -= 25
		return
	}

	if len(pr.Title) > 200 {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Title unusually long"
		result.Score -= 5
	}
}

type descriptionLengthRule struct{}

func (r *descriptionLengthRule) Name() string { return "description_length" }

func (r *descriptionLengthRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if len(pr.Body) < 20 {
		result.Failures = append(result.Failures, r.Name())
		result.Reasons[r.Name()] = "Description is empty or too short"
		result.Score -= 20
		return
	}

	wordCount := len(strings.Fields(pr.Body))
	if wordCount < 10 {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Description has very few words"
		result.Score -= 10
	}
}

type descriptionTemplateRule struct{}

func (r *descriptionTemplateRule) Name() string { return "description_template" }

func (r *descriptionTemplateRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	lowerBody := strings.ToLower(pr.Body)
	fillerWords := []string{"this pr", "this pull request", "here are the changes", "i made some changes", "some updates"}
	for _, fw := range fillerWords {
		if strings.Contains(lowerBody, fw) {
			result.Warnings = append(result.Warnings, r.Name())
			result.Reasons[r.Name()] = "Description contains filler words"
			result.Score -= 5
			return
		}
	}

	bulletCount := strings.Count(pr.Body, "•") + strings.Count(pr.Body, "-") + strings.Count(pr.Body, "*")
	totalLines := strings.Count(pr.Body, "\n") + 1
	if totalLines > 5 && float64(bulletCount) > float64(totalLines)*0.8 {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Description is mostly bullet points"
		result.Score -= 5
	}
}

type diffWhitespaceOnlyRule struct{}

func (r *diffWhitespaceOnlyRule) Name() string { return "diff_whitespace_only" }

func (r *diffWhitespaceOnlyRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if len(pr.Files) == 0 {
		return
	}

	totalChanges := 0
	whitespaceChanges := 0

	for _, f := range pr.Files {
		totalChanges += f.Additions + f.Deletions
		wsMatches := regexp.MustCompile(`^[ \t]+$`).FindAllString(f.Patch, -1)
		whitespaceChanges += len(wsMatches)
	}

	if totalChanges > 10 && float64(whitespaceChanges)/float64(totalChanges) > 0.5 {
		result.Failures = append(result.Failures, r.Name())
		result.Reasons[r.Name()] = "Diff contains mostly whitespace changes"
		result.Score -= 40
	}
}

type diffImportsOnlyRule struct{}

func (r *diffImportsOnlyRule) Name() string { return "diff_imports_only" }

func (r *diffImportsOnlyRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if len(pr.Files) == 0 {
		return
	}

	for _, f := range pr.Files {
		if f.Patch == "" {
			continue
		}

		lines := strings.Split(f.Patch, "\n")
		if len(lines) < 2 {
			continue
		}

		nonImportLines := 0

		for _, line := range lines {
			if strings.HasPrefix(line, "+") && len(line) > 1 {
				content := strings.TrimPrefix(line, "+")

				if strings.HasPrefix(content, "import") || strings.TrimSpace(content) == "\"" || strings.HasPrefix(content, "\"") {
					continue
				}

				if strings.TrimSpace(content) == "" || strings.HasPrefix(content, "//") {
					continue
				}

				nonImportLines++
			}
		}

		if nonImportLines == 0 && f.Additions > 0 {
			result.Failures = append(result.Failures, r.Name())
			result.Reasons[r.Name()] = "Diff only adds imports without code changes"
			result.Score -= 35
			return
		}
	}
}

type accountAgeRule struct{}

func (r *accountAgeRule) Name() string { return "account_age" }

func (r *accountAgeRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	minAgeDays := 7
	if pr.Author.AccountAge.Hours()/24 < float64(minAgeDays) {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Author account is less than 7 days old"
		result.Score -= 10
	}
}

type mergeRatioRule struct{}

func (r *mergeRatioRule) Name() string { return "merge_ratio" }

func (r *mergeRatioRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if pr.Author.PRCount > 3 && pr.Author.MergeRatio < 0.3 {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Author has less than 30% PR merge rate"
		result.Score -= 15
	}
}

type authorProfileRule struct{}

func (r *authorProfileRule) Name() string { return "author_profile" }

func (r *authorProfileRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	if !pr.Author.HasBio && !pr.Author.HasLocation {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "Author has incomplete profile"
		result.Score -= 5
	}
}

type pastPRQualityRule struct{}

func (r *pastPRQualityRule) Name() string { return "past_pr_quality" }

func (r *pastPRQualityRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	// This would be populated from historical data
	// For now, skip if we don't have data
	if pr.Author.PRCount == 0 {
		return
	}
}

type filesCodeRule struct{}

func (r *filesCodeRule) Name() string { return "files_code" }

func (r *filesCodeRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	codeExtensions := []string{".go", ".js", ".ts", ".tsx", ".py", ".java", ".rs", ".cpp", ".c", ".h"}

	hasCode := false
	for _, f := range pr.Files {
		for _, ext := range codeExtensions {
			if strings.HasSuffix(f.Path, ext) {
				hasCode = true
				break
			}
		}
		if hasCode {
			break
		}
	}

	if !hasCode && len(pr.Files) > 0 {
		result.Warnings = append(result.Warnings, r.Name())
		result.Reasons[r.Name()] = "No code files in diff"
		result.Score -= 10
	}
}

type honeypotRule struct{}

func newHoneypotRule() *honeypotRule {
	return &honeypotRule{}
}

func (r *honeypotRule) Name() string { return "honeypot" }

func (r *honeypotRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	// Honeypot words that AI agents tend to include but humans don't see
	honeypotTerms := []string{"[[", "]]", "<!--", "-->", "{{", "}}"}
	for _, term := range honeypotTerms {
		if strings.Contains(pr.Body, term) {
			result.Failures = append(result.Failures, r.Name())
			result.Reasons[r.Name()] = "Description contains hidden markdown markers"
			result.Score -= 50
			return
		}
	}
}

type commitMessageRule struct{}

func (r *commitMessageRule) Name() string { return "commit_message" }

func (r *commitMessageRule) Check(pr *models.PR, result *models.AntiSlopResult) {
	genericPatterns := []string{
		`^update$`,
		`^fix$`,
		`^changes$`,
		`^fix bug$`,
		`^update files$`,
		`(?i)^ai[- ]generated`,
	}

	for _, commit := range pr.Commits {
		for _, pattern := range genericPatterns {
			if matched, _ := regexp.MatchString(pattern, commit.Message); matched {
				result.Warnings = append(result.Warnings, r.Name())
				result.Reasons[r.Name()] = "Contains generic commit message: " + commit.Message
				result.Score -= 5
				return
			}
		}
	}
}
