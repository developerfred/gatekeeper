package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version       string              `yaml:"version"`
	AntiSlop      AntiSlopConfig      `yaml:"anti_slop"`
	Review        ReviewConfig        `yaml:"review"`
	IssueTriage   IssueTriageConfig   `yaml:"issue_triage"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

type AntiSlopConfig struct {
	Enabled   bool           `yaml:"enabled"`
	Threshold int            `yaml:"threshold"`
	Rules     AntiSlopRules  `yaml:"rules"`
	Honeypot  HoneypotConfig `yaml:"honeypot"`
}

type AntiSlopRules struct {
	MinAccountAgeDays     int      `yaml:"min_account_age_days"`
	MinPastPRRatio        float64  `yaml:"min_past_pr_ratio"`
	BlockedBranchPatterns []string `yaml:"blocked_branch_patterns"`
}

type HoneypotConfig struct {
	Enabled      bool   `yaml:"enabled"`
	RequiredWord string `yaml:"required_word"`
}

type ReviewConfig struct {
	Enabled         bool              `yaml:"enabled"`
	MinMergeScore   int               `yaml:"min_merge_score"`
	RequireTests    bool              `yaml:"require_tests"`
	BlockOnSecurity bool              `yaml:"block_on_security"`
	LLMOverrides    map[string]string `yaml:"llm_overrides"`
	CommentStyle    string            `yaml:"comment_style"`
	MaxComments     int               `yaml:"max_comments"`
}

type IssueTriageConfig struct {
	Enabled                bool   `yaml:"enabled"`
	AutoCloseDuplicates    bool   `yaml:"auto_close_duplicates"`
	AutoCloseNonActionable bool   `yaml:"auto_close_non_actionable"`
	LabelScheme            string `yaml:"label_scheme"`
}

type NotificationsConfig struct {
	Slack   SlackConfig   `yaml:"slack"`
	Discord DiscordConfig `yaml:"discord"`
}

type SlackConfig struct {
	Enabled bool     `yaml:"enabled"`
	Channel string   `yaml:"channel"`
	Events  []string `yaml:"events"`
}

type DiscordConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

var validCommentStyles = map[string]bool{
	"inline":              true,
	"grouped_by_file":     true,
	"grouped_by_severity": true,
}

var validLabelSchemes = map[string]bool{
	"github_default": true,
	"linear":         true,
	"asana":          true,
	"jira":           true,
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config file is empty")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Version: "1",
		AntiSlop: AntiSlopConfig{
			Enabled:   true,
			Threshold: 70,
			Rules: AntiSlopRules{
				MinAccountAgeDays: 7,
				MinPastPRRatio:    0.3,
			},
			Honeypot: HoneypotConfig{
				Enabled: true,
			},
		},
		Review: ReviewConfig{
			Enabled:       true,
			MinMergeScore: 60,
			CommentStyle:  "inline",
			MaxComments:   5,
		},
		IssueTriage: IssueTriageConfig{
			Enabled:             true,
			AutoCloseDuplicates: true,
		},
	}
}

func (c *Config) Validate() error {
	if c.AntiSlop.Enabled || c.AntiSlop.Threshold != 0 {
		if c.AntiSlop.Threshold < 0 || c.AntiSlop.Threshold > 100 {
			return fmt.Errorf("anti_slop.threshold must be 0-100, got %d", c.AntiSlop.Threshold)
		}
	}

	if c.AntiSlop.Rules.MinPastPRRatio < 0 || c.AntiSlop.Rules.MinPastPRRatio > 1 {
		return fmt.Errorf("anti_slop.rules.min_past_pr_ratio must be 0.0-1.0, got %f", c.AntiSlop.Rules.MinPastPRRatio)
	}

	if c.AntiSlop.Rules.MinAccountAgeDays < 0 {
		return fmt.Errorf("anti_slop.rules.min_account_age_days must be >= 0, got %d", c.AntiSlop.Rules.MinAccountAgeDays)
	}

	for _, pattern := range c.AntiSlop.Rules.BlockedBranchPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid branch pattern regex %q: %w", pattern, err)
		}
	}

	if c.Review.Enabled {
		if c.Review.MinMergeScore < 0 || c.Review.MinMergeScore > 100 {
			return fmt.Errorf("review.min_merge_score must be 0-100, got %d", c.Review.MinMergeScore)
		}

		if c.Review.MaxComments < 0 {
			return fmt.Errorf("review.max_comments must be >= 0, got %d", c.Review.MaxComments)
		}

		if c.Review.CommentStyle != "" && !validCommentStyles[c.Review.CommentStyle] {
			return fmt.Errorf("review.comment_style must be one of: inline, grouped_by_file, grouped_by_severity, got %q", c.Review.CommentStyle)
		}
	}

	if c.IssueTriage.Enabled {
		if c.IssueTriage.LabelScheme != "" && !validLabelSchemes[c.IssueTriage.LabelScheme] {
			return fmt.Errorf("issue_triage.label_scheme must be one of: github_default, linear, asana, jira, got %q", c.IssueTriage.LabelScheme)
		}
	}

	if c.Notifications.Slack.Enabled {
		if c.Notifications.Slack.Channel != "" && !strings.HasPrefix(c.Notifications.Slack.Channel, "#") && !strings.HasPrefix(c.Notifications.Slack.Channel, "@") {
			return fmt.Errorf("notifications.slack.channel must start with # or @, got %q", c.Notifications.Slack.Channel)
		}
	}

	if c.Notifications.Discord.Enabled {
		if c.Notifications.Discord.WebhookURL != "" && !isValidDiscordWebhook(c.Notifications.Discord.WebhookURL) {
			return fmt.Errorf("notifications.discord.webhook_url must be a valid Discord webhook URL")
		}
	}

	return nil
}

func isValidDiscordWebhook(url string) bool {
	return strings.Contains(url, "discord.com/api/webhooks/")
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("GATEKEEPER_ANTI_SLOP_THRESHOLD"); v != "" {
		if threshold, err := strconv.Atoi(v); err == nil {
			cfg.AntiSlop.Threshold = threshold
		}
	}

	if v := os.Getenv("GATEKEEPER_REVIEW_MIN_MERGE_SCORE"); v != "" {
		if score, err := strconv.Atoi(v); err == nil {
			cfg.Review.MinMergeScore = score
		}
	}

	if v := os.Getenv("GATEKEEPER_ANTI_SLOP_ENABLED"); v != "" {
		cfg.AntiSlop.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("GATEKEEPER_REVIEW_ENABLED"); v != "" {
		cfg.Review.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("GATEKEEPER_ISSUE_TRIAGE_ENABLED"); v != "" {
		cfg.IssueTriage.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("GATEKEEPER_SLACK_ENABLED"); v != "" {
		cfg.Notifications.Slack.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("GATEKEEPER_SLACK_CHANNEL"); v != "" {
		cfg.Notifications.Slack.Channel = v
	}

	if v := os.Getenv("GATEKEEPER_DISCORD_ENABLED"); v != "" {
		cfg.Notifications.Discord.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("GATEKEEPER_DISCORD_WEBHOOK_URL"); v != "" {
		cfg.Notifications.Discord.WebhookURL = v
	}

	if v := os.Getenv("GATEKEEPER_MAX_COMMENTS"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			cfg.Review.MaxComments = max
		}
	}
}

func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	result := *c

	if other.AntiSlop.Threshold != 0 {
		result.AntiSlop.Threshold = other.AntiSlop.Threshold
	}
	if other.AntiSlop.Enabled {
		result.AntiSlop.Enabled = other.AntiSlop.Enabled
	}
	if other.AntiSlop.Rules.MinAccountAgeDays != 0 {
		result.AntiSlop.Rules.MinAccountAgeDays = other.AntiSlop.Rules.MinAccountAgeDays
	}
	if other.AntiSlop.Rules.MinPastPRRatio != 0 {
		result.AntiSlop.Rules.MinPastPRRatio = other.AntiSlop.Rules.MinPastPRRatio
	}
	if len(other.AntiSlop.Rules.BlockedBranchPatterns) > 0 {
		result.AntiSlop.Rules.BlockedBranchPatterns = other.AntiSlop.Rules.BlockedBranchPatterns
	}

	if other.Review.MinMergeScore != 0 {
		result.Review.MinMergeScore = other.Review.MinMergeScore
	}
	if other.Review.MaxComments != 0 {
		result.Review.MaxComments = other.Review.MaxComments
	}
	if other.Review.CommentStyle != "" {
		result.Review.CommentStyle = other.Review.CommentStyle
	}

	return &result
}
