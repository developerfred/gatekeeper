package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

// CLIFormatter handles terminal output formatting
type CLIFormatter struct{}

// NewCLIFormatter creates a new CLI formatter
func NewCLIFormatter() *CLIFormatter {
	return &CLIFormatter{}
}

// PrintReview prints a review result to the terminal
func (f *CLIFormatter) PrintReview(result *ReviewResult) error {
	if result == nil {
		return fmt.Errorf("nil result")
	}

	// Box drawing characters for terminal UI
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 🔍 GATEKEEPER REVIEW                                       │")
	fmt.Println("├─────────────────────────────────────────────────────────────┤")

	// Anti-slop status
	if result.AntiSlop != nil {
		fmt.Printf("│ 🤖 Anti-Slop: ")
		if result.AntiSlop.Passed {
			fmt.Printf("PASSED (score: %d/100)", result.AntiSlop.Score)
		} else {
			fmt.Printf("FAILED (score: %d/100)", result.AntiSlop.Score)
		}
		fmt.Println(strings.Repeat(" ", max(0, 52-len(fmt.Sprintf("PASSED (score: %d/100)", result.AntiSlop.Score)))) + "│")
	}

	// Review score
	if result.Review != nil {
		scoreEmoji := "⚠️"
		if result.Review.MergeScore >= 85 {
			scoreEmoji = "✅"
		} else if result.Review.MergeScore < 60 {
			scoreEmoji = "🔴"
		}

		fmt.Printf("│ %s MERGE READINESS: %d/100", scoreEmoji, result.Review.MergeScore)
		fmt.Println(strings.Repeat(" ", max(0, 53-len(fmt.Sprintf("%s MERGE READINESS: %d/100", scoreEmoji, result.Review.MergeScore)))) + "│")

		// Issues count
		issueCount := len(result.Review.Issues)
		if issueCount > 0 {
			fmt.Printf("│ ⚠️  ISSUES FOUND: %d", issueCount)
			fmt.Println(strings.Repeat(" ", max(0, 53-len(fmt.Sprintf("⚠️  ISSUES FOUND: %d", issueCount)))) + "│")

			// Print top issues (max 5)
			for i, issue := range result.Review.Issues {
				if i >= 5 {
					fmt.Println("│   ... and more issues                                     │")
					break
				}
				severityStr := fmt.Sprintf("[%s]", strings.ToUpper(string(issue.Severity)))
				fileStr := ""
				if issue.File != "" {
					fileStr = fmt.Sprintf(" %s:%d", issue.File, issue.Line)
				}
				titleStr := fmt.Sprintf(" %s%s", severityStr, fileStr)

				if len(titleStr) > 48 {
					titleStr = titleStr[:48]
				}
				fmt.Printf("│%s", titleStr)
				fmt.Println(strings.Repeat(" ", max(0, 56-len(titleStr))) + "│")

				if issue.Title != "" {
					bodyStr := "   " + issue.Title
					if len(bodyStr) > 52 {
						bodyStr = bodyStr[:52] + "..."
					}
					fmt.Printf("│%s", bodyStr)
					fmt.Println(strings.Repeat(" ", max(0, 56-len(bodyStr))) + "│")
				}
			}
		}
	}

	// Action recommendation
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ 📋 RECOMMENDATION: ")
	switch result.Action {
	case models.ActionApprove:
		fmt.Print("APPROVE")
	case models.ActionRequestChanges:
		fmt.Print("REQUEST CHANGES")
	case models.ActionClose:
		fmt.Print("CLOSE")
	default:
		fmt.Print("COMMENT")
	}
	actionStr := string(result.Action)
	if result.ShouldClose {
		actionStr += " (auto-close)"
	}
	fmt.Println(strings.Repeat(" ", max(0, 47-len(actionStr))) + "│")

	// Duration and cost
	if result.Review != nil {
		costStr := fmt.Sprintf("⏱️  Time: %dms · 💰 Cost: $%.4f", result.Review.DurationMs, result.Review.CostUSD)
		fmt.Printf("│ %s", costStr)
		fmt.Println(strings.Repeat(" ", max(0, 56-len(costStr))) + "│")
	}

	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	return nil
}

// PrintJSON prints the result as JSON
func PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal failed: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// ReviewResult mirrors the review pipeline result for output
type ReviewResult struct {
	Action      models.ReviewAction    `json:"action"`
	ShouldClose bool                   `json:"should_close"`
	CloseReason string                 `json:"close_reason,omitempty"`
	AntiSlop    *models.AntiSlopResult `json:"anti_slop,omitempty"`
	Review      *models.ReviewResult   `json:"review,omitempty"`
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
