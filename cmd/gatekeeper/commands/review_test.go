package commands

import (
	"context"
	"testing"

	"github.com/gatekeeper/gatekeeper/internal/core/models"
)

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantNum   int
		wantErr   bool
	}{
		{
			name:      "GitHub HTTPS",
			input:     "https://github.com/owner/repo/pull/123",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantNum:   123,
			wantErr:   false,
		},
		{
			name:      "GitHub with .git suffix",
			input:     "https://github.com/owner/repo/pull/456.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantNum:   456,
			wantErr:   false,
		},
		{
			name:      "Short form PR number",
			input:     "owner/repo#789",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantNum:   789,
			wantErr:   false,
		},
		{
			name:    "Invalid URL",
			input:   "not-a-url",
			wantErr: true,
		},
		{
			name:    "Missing PR number",
			input:   "https://github.com/owner/repo/pull/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, num, err := parsePRURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePRURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("repo = %v, want %v", repo, tt.wantRepo)
				}
				if num != tt.wantNum {
					t.Errorf("num = %v, want %v", num, tt.wantNum)
				}
			}
		})
	}
}

func TestReviewCmd_FlagDefaults(t *testing.T) {
	cmd := NewReviewCmd()

	// Check provider default
	provider, _ := cmd.Flags().GetString("provider")
	if provider != "ollama" {
		t.Errorf("default provider = %q, want %q", provider, "ollama")
	}

	// Check model default
	model, _ := cmd.Flags().GetString("model")
	if model != "qwen2.5-coder:32b" {
		t.Errorf("default model = %q, want %q", model, "qwen2.5-coder:32b")
	}

	// Check threshold default
	threshold, _ := cmd.Flags().GetInt("threshold")
	if threshold != 70 {
		t.Errorf("default threshold = %d, want %d", threshold, 70)
	}
}

func TestReviewCmd_URLFlag(t *testing.T) {
	cmd := NewReviewCmd()

	// Test --pr flag exists and accepts values
	if err := cmd.Flags().Set("pr", "https://github.com/owner/repo/pull/123"); err != nil {
		t.Errorf("failed to set --pr flag: %v", err)
	}
}

func TestReviewPipeline_Integration(t *testing.T) {
	// Test that review pipeline works end-to-end with mocks
	pipeline := NewReviewPipeline("ollama", "qwen2.5-coder:32b", 70)

	pr := &models.PR{
		Number:     123,
		Title:      "feat: add user authentication",
		Body:       "Implements JWT-based authentication",
		BaseBranch: "main",
		HeadBranch: "feature/auth",
		Files: []models.File{
			{Path: "auth/jwt.go", Additions: 150, Deletions: 0},
			{Path: "auth/jwt_test.go", Additions: 80, Deletions: 0},
		},
		Author: models.Author{
			Login:       "developer",
			AccountAge:  365 * 24,
			PRCount:     10,
			MergeRatio:  0.85,
			HasBio:      true,
			HasLocation: true,
		},
	}

	ctx := context.Background()
	result, err := pipeline.Run(ctx, pr)

	if err != nil {
		t.Fatalf("pipeline.Run() error = %v", err)
	}

	if result == nil {
		t.Fatal("pipeline.Run() returned nil result")
	}

	if result.Action == "" {
		t.Error("Action is empty")
	}
}
