package main

import (
	"fmt"
	"os"

	"github.com/gatekeeper/gatekeeper/internal/anti_slop"
	"github.com/gatekeeper/gatekeeper/internal/core/models"
	"github.com/gatekeeper/gatekeeper/internal/llm"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("GateKeeper v0.1.0")
	case "check":
		runCheck()
	case "review":
		runReview()
	case "init":
		runInit()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`GateKeeper - AI-Powered PR & Issue Validation

Usage:
  gatekeeper <command> [options]

Commands:
  version     Show version information
  check       Run anti-slop check on a PR
  review      Run full AI review on a PR
  init        Initialize gatekeeper configuration

Examples:
  gatekeeper check --pr https://github.com/owner/repo/pull/123
  gatekeeper review --pr https://github.com/owner/repo/pull/123 --provider ollama
  gatekeeper init`)
}

func runCheck() {
	pr := &models.PR{
		Number:     1,
		Title:      os.Getenv("PR_TITLE"),
		Body:       os.Getenv("PR_BODY"),
		HeadBranch: os.Getenv("PR_BRANCH"),
		Author: models.Author{
			Login:       os.Getenv("PR_AUTHOR"),
			HasBio:      true,
			HasLocation: true,
		},
		Files: []models.File{
			{Path: "example.go", Additions: 100, Deletions: 10},
		},
	}

	checker := anti_slop.NewChecker(70)
	result := checker.Check(pr)

	if result.Passed {
		fmt.Printf("✅ PASSED (score: %d/100)\n", result.Score)
	} else {
		fmt.Printf("❌ FAILED (score: %d/100)\n", result.Score)
		fmt.Println("\nFailures:")
		for _, f := range result.Failures {
			fmt.Printf("  - %s: %s\n", f, result.Reasons[f])
		}
	}

	fmt.Println("\nWarnings:")
	for _, w := range result.Warnings {
		fmt.Printf("  - %s: %s\n", w, result.Reasons[w])
	}
}

func runReview() {
	router := llm.NewRouter()

	ollama := llm.NewOllamaProvider("http://localhost:11434")
	router.Register(ollama)

	fmt.Println("🤖 GateKeeper AI Review")
	fmt.Println("========================")

	if !ollama.IsAvailable(nil) {
		fmt.Println("⚠️  Ollama not available at http://localhost:11434")
		fmt.Println("   Make sure Ollama is running: brew install ollama && ollama serve")
		fmt.Println("   Or use: gatekeeper review --provider openai")
		return
	}

	messages := []llm.Message{
		{Role: "system", Content: "You are a senior code reviewer. Provide concise, actionable feedback."},
		{Role: "user", Content: "Review this PR: Add user authentication with JWT tokens.\n\nFiles changed: auth/jwt.go, auth/jwt_test.go\n\nFocus on: security, correctness, and best practices."},
	}

	resp, err := router.Chat(nil, messages, llm.ChatOptions{
		Model:       "qwen2.5-coder:32b",
		MaxTokens:   1000,
		Temperature: 0.3,
	})

	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	fmt.Printf("✅ Review complete (%.2fs, $%.4f)\n\n", resp.Duration.Seconds(), resp.CostUSD)
	fmt.Println(resp.Content)
}

func runInit() {
	fmt.Println("Initializing GateKeeper configuration...")

	config := `# GateKeeper Configuration
version: "1"

anti_slop:
  enabled: true
  threshold: 70

review:
  enabled: true
  min_merge_score: 60
  llm_provider: ollama
  ollama_model: qwen2.5-coder:32b

issue_triage:
  enabled: true
`

	fmt.Println("\nCreated .gatekeeper.yml:")
	fmt.Println(config)
	fmt.Println("Customize this file to match your project's needs.")
}
