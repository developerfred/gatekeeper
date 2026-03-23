package commands

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gatekeeper/gatekeeper/internal/anti_slop"
	"github.com/gatekeeper/gatekeeper/internal/core/models"
	"github.com/gatekeeper/gatekeeper/internal/llm"
	"github.com/gatekeeper/gatekeeper/internal/output"
	"github.com/gatekeeper/gatekeeper/internal/review"
	"github.com/spf13/cobra"
)

// ReviewCmd handles the `gatekeeper review` command
type ReviewCmd struct {
	cmd      *cobra.Command
	pipeline *ReviewPipeline
}

// ReviewPipeline orchestrates the full review flow
type ReviewPipeline struct {
	antiSlopChecker *anti_slop.Checker
	llmRouter       *llm.Router
	reviewPipeline  *review.ReviewPipeline
	outputFormatter *output.CLIFormatter
}

// NewReviewCmd creates a new review command
func NewReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Run AI review on a pull request",
		Long: `Run full AI-powered review on a pull request.

Supports:
  - GitHub PRs via --pr flag
  - Multiple LLM providers (ollama, openai)

Examples:
  gatekeeper review --pr https://github.com/owner/repo/pull/123
  gatekeeper review --pr owner/repo#123 --provider openai
  gatekeeper review --pr owner/repo#123 --model gpt-4o`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(cmd)
		},
	}

	// Command flags
	cmd.Flags().StringP("pr", "p", "", "PR URL or owner/repo#number format (required)")
	cmd.Flags().String("provider", "ollama", "LLM provider (ollama, openai)")
	cmd.Flags().String("model", "qwen2.5-coder:32b", "Model name")
	cmd.Flags().Int("threshold", 70, "Anti-slop threshold (0-100)")
	cmd.Flags().Bool("no-comment", false, "Don't post GitHub comment")
	cmd.Flags().Bool("json", false, "Output JSON format")

	// Mark PR as required
	cmd.MarkFlagRequired("pr")

	return cmd
}

func runReview(cmd *cobra.Command) error {
	prURL, _ := cmd.Flags().GetString("pr")
	provider, _ := cmd.Flags().GetString("provider")
	model, _ := cmd.Flags().GetString("model")
	threshold, _ := cmd.Flags().GetInt("threshold")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Parse PR URL
	owner, repo, prNum, err := parsePRURL(prURL)
	if err != nil {
		return fmt.Errorf("invalid PR URL: %w", err)
	}

	ctx := context.Background()

	// Build pipeline
	pipeline := NewReviewPipeline(provider, model, threshold)

	// Run review (in real impl, would fetch PR from GitHub)
	// For now, create a placeholder PR from the URL info
	pr := &models.PR{
		Number:     prNum,
		Title:      fmt.Sprintf("PR #%d from %s/%s", prNum, owner, repo),
		BaseBranch: "main",
		HeadBranch: "feature",
	}

	result, err := pipeline.Run(ctx, pr)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Format output
	if jsonOutput {
		return output.PrintJSON(result)
	}

	return pipeline.outputFormatter.PrintReview(&output.ReviewResult{
		Action:      result.Action,
		ShouldClose: result.ShouldClose,
		CloseReason: result.CloseReason,
		AntiSlop:    result.AntiSlop,
		Review:      result.Review,
	})
}

// parsePRURL extracts owner, repo, and PR number from various URL formats
func parsePRURL(input string) (owner, repo string, num int, err error) {
	input = strings.TrimSpace(input)

	// Handle owner/repo#number format
	if strings.Contains(input, "#") && !strings.Contains(input, "github.com") {
		parts := strings.Split(input, "#")
		if len(parts) != 2 {
			return "", "", 0, fmt.Errorf("invalid format: expected owner/repo#number")
		}
		ownerRepo := strings.Split(parts[0], "/")
		if len(ownerRepo) != 2 {
			return "", "", 0, fmt.Errorf("invalid format: expected owner/repo#number")
		}
		num, err = strconv.Atoi(parts[1])
		if err != nil {
			return "", "", 0, fmt.Errorf("invalid PR number: %w", err)
		}
		return ownerRepo[0], ownerRepo[1], num, nil
	}

	// Handle full URL
	parsed, err := url.Parse(input)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Remove .git suffix if present
	path := strings.TrimSuffix(parsed.Path, ".git")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Expected format: owner/repo/pull/123
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("URL does not appear to be a PR: %s", input)
	}

	owner = parts[0]
	repo = parts[1]
	num, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number in URL: %w", err)
	}

	return owner, repo, num, nil
}

// NewReviewPipeline creates a configured review pipeline
func NewReviewPipeline(provider, model string, threshold int) *ReviewPipeline {
	router := llm.NewRouter()

	// Register providers based on config
	switch provider {
	case "ollama":
		ollama := llm.NewOllamaProvider("http://localhost:11434")
		router.Register(ollama)
	case "openai":
		openai := llm.NewOpenAIProvider("")
		router.Register(openai)
	}

	return &ReviewPipeline{
		antiSlopChecker: anti_slop.NewChecker(threshold),
		llmRouter:       router,
		reviewPipeline:  review.NewReviewPipeline(),
		outputFormatter: output.NewCLIFormatter(),
	}
}

// Run executes the full review pipeline
func (p *ReviewPipeline) Run(ctx context.Context, pr *models.PR) (*ReviewResult, error) {
	result := &ReviewResult{}

	// Step 1: Anti-Slop check
	antiSlopResult := p.antiSlopChecker.Check(pr)
	result.AntiSlop = antiSlopResult

	// Step 2: If anti-slop failed, return early with close action
	if !antiSlopResult.Passed {
		result.Action = models.ActionClose
		result.ShouldClose = true
		result.CloseReason = "Anti-slop check failed"
		return result, nil
	}

	// Step 3: Run AI review
	reviewResult, err := p.reviewPipeline.Review(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}
	result.Review = reviewResult

	// Step 4: Determine final action
	result.Action = reviewResult.Action
	result.ShouldClose = reviewResult.Action == models.ActionClose

	return result, nil
}

// ReviewResult is the combined output of review pipeline
type ReviewResult struct {
	Action      models.ReviewAction    `json:"action"`
	ShouldClose bool                   `json:"should_close"`
	CloseReason string                 `json:"close_reason,omitempty"`
	AntiSlop    *models.AntiSlopResult `json:"anti_slop,omitempty"`
	Review      *models.ReviewResult   `json:"review,omitempty"`
}
