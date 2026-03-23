package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	yaml := `
version: "1"
anti_slop:
  enabled: true
  threshold: 70
  rules:
    min_account_age_days: 7
    min_past_pr_ratio: 0.3
    blocked_branch_patterns:
      - "^fix-ai-.*"
      - "^update-readme.*"
  honeypot:
    enabled: true
    required_word: "clank"
review:
  enabled: true
  min_merge_score: 60
  require_tests: true
  block_on_security: true
  llm_overrides:
    security: anthropic/claude-sonnet-4-5
    general: openai/gpt-4o
  comment_style: grouped_by_file
  max_comments: 10
issue_triage:
  enabled: true
  auto_close_duplicates: true
  auto_close_non_actionable: true
  label_scheme: github_default
notifications:
  slack:
    enabled: false
    channel: "#pr-reviews"
    events: [blocked, high_risk]
  discord:
    enabled: true
    webhook_url: ""
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gatekeeper.yml")
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1")
	}
	if !cfg.AntiSlop.Enabled {
		t.Error("AntiSlop.Enabled = false, want true")
	}
	if cfg.AntiSlop.Threshold != 70 {
		t.Errorf("AntiSlop.Threshold = %d, want %d", cfg.AntiSlop.Threshold, 70)
	}
	if cfg.AntiSlop.Rules.MinAccountAgeDays != 7 {
		t.Errorf("AntiSlop.Rules.MinAccountAgeDays = %d, want %d", cfg.AntiSlop.Rules.MinAccountAgeDays, 7)
	}
	if cfg.AntiSlop.Rules.MinPastPRRatio != 0.3 {
		t.Errorf("AntiSlop.Rules.MinPastPRRatio = %f, want %f", cfg.AntiSlop.Rules.MinPastPRRatio, 0.3)
	}
	if len(cfg.AntiSlop.Rules.BlockedBranchPatterns) != 2 {
		t.Errorf("len(AntiSlop.Rules.BlockedBranchPatterns) = %d, want %d", len(cfg.AntiSlop.Rules.BlockedBranchPatterns), 2)
	}
	if !cfg.AntiSlop.Honeypot.Enabled {
		t.Error("AntiSlop.Honeypot.Enabled = false, want true")
	}
	if cfg.AntiSlop.Honeypot.RequiredWord != "clank" {
		t.Errorf("AntiSlop.Honeypot.RequiredWord = %q, want %q", cfg.AntiSlop.Honeypot.RequiredWord, "clank")
	}
	if !cfg.Review.Enabled {
		t.Error("Review.Enabled = false, want true")
	}
	if cfg.Review.MinMergeScore != 60 {
		t.Errorf("Review.MinMergeScore = %d, want %d", cfg.Review.MinMergeScore, 60)
	}
	if !cfg.Review.RequireTests {
		t.Error("Review.RequireTests = false, want true")
	}
	if !cfg.Review.BlockOnSecurity {
		t.Error("Review.BlockOnSecurity = false, want true")
	}
	if cfg.Review.LLMOverrides["security"] != "anthropic/claude-sonnet-4-5" {
		t.Errorf("Review.LLMOverrides[security] = %q, want %q", cfg.Review.LLMOverrides["security"], "anthropic/claude-sonnet-4-5")
	}
	if cfg.Review.CommentStyle != "grouped_by_file" {
		t.Errorf("Review.CommentStyle = %q, want %q", cfg.Review.CommentStyle, "grouped_by_file")
	}
	if cfg.Review.MaxComments != 10 {
		t.Errorf("Review.MaxComments = %d, want %d", cfg.Review.MaxComments, 10)
	}
	if !cfg.IssueTriage.Enabled {
		t.Error("IssueTriage.Enabled = false, want true")
	}
	if !cfg.IssueTriage.AutoCloseDuplicates {
		t.Error("IssueTriage.AutoCloseDuplicates = false, want true")
	}
	if !cfg.IssueTriage.AutoCloseNonActionable {
		t.Error("IssueTriage.AutoCloseNonActionable = false, want true")
	}
	if cfg.IssueTriage.LabelScheme != "github_default" {
		t.Errorf("IssueTriage.LabelScheme = %q, want %q", cfg.IssueTriage.LabelScheme, "github_default")
	}
	if cfg.Notifications.Slack.Enabled {
		t.Error("Notifications.Slack.Enabled = true, want false")
	}
	if cfg.Notifications.Slack.Channel != "#pr-reviews" {
		t.Errorf("Notifications.Slack.Channel = %q, want %q", cfg.Notifications.Slack.Channel, "#pr-reviews")
	}
	if !cfg.Notifications.Discord.Enabled {
		t.Error("Notifications.Discord.Enabled = false, want true")
	}
	if cfg.Notifications.Discord.WebhookURL != "" {
		t.Errorf("Notifications.Discord.WebhookURL = %q, want %q", cfg.Notifications.Discord.WebhookURL, "")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.gatekeeper.yml")
	if err == nil {
		t.Error("Load() error = nil, want error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	invalidYAML := `
version: "1"
anti_slop:
  enabled: true
  threshold: [invalid
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gatekeeper.yml")
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() error = nil, want error for invalid YAML")
	}
}

func TestValidate_ThresholdOutOfRange(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		wantErr   bool
	}{
		{"threshold 0", 0, false},
		{"threshold 50", 50, false},
		{"threshold 100", 100, false},
		{"threshold -1", -1, true},
		{"threshold 101", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				AntiSlop: AntiSlopConfig{
					Enabled:   true,
					Threshold: tt.threshold,
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_InvalidCommentStyle(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Review: ReviewConfig{
			Enabled:      true,
			CommentStyle: "invalid_style",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid CommentStyle")
	}
}

func TestValidate_InvalidLabelScheme(t *testing.T) {
	cfg := &Config{
		Version: "1",
		IssueTriage: IssueTriageConfig{
			Enabled:     true,
			LabelScheme: "invalid_scheme",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid LabelScheme")
	}
}

func TestValidate_InvalidMinMergeScore(t *testing.T) {
	tests := []struct {
		name          string
		minMergeScore int
		wantErr       bool
	}{
		{"score 0", 0, false},
		{"score 50", 50, false},
		{"score 100", 100, false},
		{"score -1", -1, true},
		{"score 101", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				Review: ReviewConfig{
					Enabled:       true,
					MinMergeScore: tt.minMergeScore,
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_InvalidMaxComments(t *testing.T) {
	tests := []struct {
		name        string
		maxComments int
		wantErr     bool
	}{
		{"max 0", 0, false},
		{"max 50", 50, false},
		{"max 1000", 1000, false},
		{"max -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				Review: ReviewConfig{
					Enabled:      true,
					MaxComments:  tt.maxComments,
					CommentStyle: "inline",
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != "1" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1")
	}
	if !cfg.AntiSlop.Enabled {
		t.Error("AntiSlop.Enabled = false, want true")
	}
	if cfg.AntiSlop.Threshold != 70 {
		t.Errorf("AntiSlop.Threshold = %d, want %d", cfg.AntiSlop.Threshold, 70)
	}
	if cfg.AntiSlop.Rules.MinAccountAgeDays != 7 {
		t.Errorf("AntiSlop.Rules.MinAccountAgeDays = %d, want %d", cfg.AntiSlop.Rules.MinAccountAgeDays, 7)
	}
	if cfg.AntiSlop.Rules.MinPastPRRatio != 0.3 {
		t.Errorf("AntiSlop.Rules.MinPastPRRatio = %f, want %f", cfg.AntiSlop.Rules.MinPastPRRatio, 0.3)
	}
	if !cfg.AntiSlop.Honeypot.Enabled {
		t.Error("AntiSlop.Honeypot.Enabled = false, want true")
	}
	if !cfg.Review.Enabled {
		t.Error("Review.Enabled = false, want true")
	}
	if cfg.Review.MinMergeScore != 60 {
		t.Errorf("Review.MinMergeScore = %d, want %d", cfg.Review.MinMergeScore, 60)
	}
	if cfg.Review.CommentStyle != "inline" {
		t.Errorf("Review.CommentStyle = %q, want %q", cfg.Review.CommentStyle, "inline")
	}
	if cfg.Review.MaxComments != 5 {
		t.Errorf("Review.MaxComments = %d, want %d", cfg.Review.MaxComments, 5)
	}
	if !cfg.IssueTriage.Enabled {
		t.Error("IssueTriage.Enabled = false, want true")
	}
	if !cfg.IssueTriage.AutoCloseDuplicates {
		t.Error("IssueTriage.AutoCloseDuplicates = false, want true")
	}
	if cfg.Notifications.Slack.Enabled {
		t.Error("Notifications.Slack.Enabled = true, want false")
	}
	if cfg.Notifications.Discord.Enabled {
		t.Error("Notifications.Discord.Enabled = true, want false")
	}
}

func TestLoad_WithEnvOverrides(t *testing.T) {
	yaml := `
version: "1"
anti_slop:
  enabled: true
  threshold: 70
review:
  enabled: true
  min_merge_score: 60
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gatekeeper.yml")
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	os.Setenv("GATEKEEPER_ANTI_SLOP_THRESHOLD", "80")
	os.Setenv("GATEKEEPER_REVIEW_MIN_MERGE_SCORE", "75")
	defer func() {
		os.Unsetenv("GATEKEEPER_ANTI_SLOP_THRESHOLD")
		os.Unsetenv("GATEKEEPER_REVIEW_MIN_MERGE_SCORE")
	}()

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AntiSlop.Threshold != 80 {
		t.Errorf("AntiSlop.Threshold = %d, want %d (from env)", cfg.AntiSlop.Threshold, 80)
	}
	if cfg.Review.MinMergeScore != 75 {
		t.Errorf("Review.MinMergeScore = %d, want %d (from env)", cfg.Review.MinMergeScore, 75)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gatekeeper.yml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Error("Load() error = nil, want error for empty file")
	}
}

func TestValidate_InvalidMinPastPRRatio(t *testing.T) {
	tests := []struct {
		name           string
		minPastPRRatio float64
		wantErr        bool
	}{
		{"ratio 0", 0.0, false},
		{"ratio 0.5", 0.5, false},
		{"ratio 1.0", 1.0, false},
		{"ratio -0.1", -0.1, true},
		{"ratio 1.1", 1.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				AntiSlop: AntiSlopConfig{
					Enabled:   true,
					Threshold: 70,
					Rules: AntiSlopRules{
						MinPastPRRatio: tt.minPastPRRatio,
					},
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_InvalidMinAccountAgeDays(t *testing.T) {
	tests := []struct {
		name              string
		minAccountAgeDays int
		wantErr           bool
	}{
		{"days 0", 0, false},
		{"days 30", 30, false},
		{"days 365", 365, false},
		{"days -1", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				AntiSlop: AntiSlopConfig{
					Enabled:   true,
					Threshold: 70,
					Rules: AntiSlopRules{
						MinAccountAgeDays: tt.minAccountAgeDays,
					},
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	yaml := `
version: "1"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gatekeeper.yml")
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("Version = %q, want %q", cfg.Version, "1")
	}
	if cfg.AntiSlop.Enabled {
		t.Error("AntiSlop.Enabled = true, want false (default)")
	}
}

func TestValidate_DiscordWebhookURL(t *testing.T) {
	tests := []struct {
		name       string
		webhookURL string
		wantErr    bool
	}{
		{"empty", "", false},
		{"valid https", "https://discord.com/api/webhooks/123456/abcdef", false},
		{"valid http", "http://discord.com/api/webhooks/123456/abcdef", false},
		{"invalid no webhook", "https://discord.com/channels/123456/789012", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				Notifications: NotificationsConfig{
					Discord: DiscordConfig{
						Enabled:    true,
						WebhookURL: tt.webhookURL,
					},
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SlackChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		wantErr bool
	}{
		{"valid #channel", "#pr-reviews", false},
		{"valid @user", "@username", false},
		{"empty", "", false},
		{"no prefix", "pr-reviews", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Version: "1",
				Notifications: NotificationsConfig{
					Slack: SlackConfig{
						Enabled: true,
						Channel: tt.channel,
						Events:  []string{"blocked"},
					},
				},
			}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_BlockedBranchPatterns(t *testing.T) {
	cfg := &Config{
		Version: "1",
		AntiSlop: AntiSlopConfig{
			Enabled:   true,
			Threshold: 70,
			Rules: AntiSlopRules{
				BlockedBranchPatterns: []string{"^fix-ai-.*", "[invalid regex("},
			},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() error = nil, want error for invalid regex pattern")
	}
}

func TestConfig_Merge(t *testing.T) {
	base := &Config{
		Version: "1",
		AntiSlop: AntiSlopConfig{
			Enabled:   true,
			Threshold: 70,
			Rules: AntiSlopRules{
				MinAccountAgeDays: 7,
			},
		},
	}
	override := &Config{
		Version: "1",
		AntiSlop: AntiSlopConfig{
			Threshold: 80,
			Rules: AntiSlopRules{
				MinAccountAgeDays: 14,
			},
		},
	}

	merged := base.Merge(override)

	if merged.AntiSlop.Threshold != 80 {
		t.Errorf("AntiSlop.Threshold = %d, want %d", merged.AntiSlop.Threshold, 80)
	}
	if merged.AntiSlop.Rules.MinAccountAgeDays != 14 {
		t.Errorf("AntiSlop.Rules.MinAccountAgeDays = %d, want %d", merged.AntiSlop.Rules.MinAccountAgeDays, 14)
	}
	if !merged.AntiSlop.Enabled {
		t.Error("AntiSlop.Enabled = false, want true (from base)")
	}
}
