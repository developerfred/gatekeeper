package models

import (
	"time"
)

// Severity levels for review issues
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ReviewAction indicates what action to take on a PR
type ReviewAction string

const (
	ActionApprove        ReviewAction = "approve"
	ActionRequestChanges ReviewAction = "request_changes"
	ActionComment        ReviewAction = "comment"
	ActionClose          ReviewAction = "close"
)

// IssueState represents the state of an issue
type IssueState string

const (
	IssueStateOpen   IssueState = "open"
	IssueStateClosed IssueState = "closed"
	IssueStateAll    IssueState = "all"
)

// EventType represents the type of forge event
type EventType string

const (
	EventTypePROpened     EventType = "pr_opened"
	EventTypePRUpdated    EventType = "pr_updated"
	EventTypePRClosed     EventType = "pr_closed"
	EventTypePRMerged     EventType = "pr_merged"
	EventTypeIssueOpened  EventType = "issue_opened"
	EventTypeIssueUpdated EventType = "issue_updated"
)

// IssueType represents the classification of an issue
type IssueType string

const (
	IssueTypeBug        IssueType = "bug"
	IssueTypeFeature    IssueType = "feature"
	IssueTypeQuestion   IssueType = "question"
	IssueTypeDiscussion IssueType = "discussion"
	IssueTypeOther      IssueType = "other"
)

// Priority represents issue priority
type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
	PriorityNone     Priority = "none"
)

// Author represents a contributor
type Author struct {
	Login       string        `json:"login"`
	AccountAge  time.Duration `json:"account_age"`
	PRCount     int           `json:"pr_count"`
	MergeRatio  float64       `json:"merge_ratio"` // 0.0 - 1.0
	HasBio      bool          `json:"has_bio"`
	HasLocation bool          `json:"has_location"`
}

// File represents a file changed in a PR
type File struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // added, modified, deleted
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// Commit represents a commit in a PR
type Commit struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
}

// PR represents a pull request under review
type PR struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Author     Author    `json:"author"`
	BaseBranch string    `json:"base_branch"`
	HeadBranch string    `json:"head_branch"`
	Diff       string    `json:"diff"`
	Files      []File    `json:"files"`
	Commits    []Commit  `json:"commits"`
	Labels     []string  `json:"labels"`
	State      string    `json:"state"` // open, closed, merged
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RepoRef represents a repository reference
type RepoRef struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Host  string `json:"host"` // github.com, gitlab.com, self-hosted URL
}

// Issue represents a GitHub issue under triage
type Issue struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Author    Author     `json:"author"`
	Labels    []string   `json:"labels"`
	State     IssueState `json:"state"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ReviewIssue is a single issue found in review
type ReviewIssue struct {
	Severity   Severity `json:"severity"`
	Category   string   `json:"category"` // security, correctness, performance, etc
	File       string   `json:"file"`
	Line       int      `json:"line"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Suggestion string   `json:"suggestion,omitempty"`
	Rule       string   `json:"rule,omitempty"` // which rule detected it
}

// ReviewResult is the output of the review pipeline
type ReviewResult struct {
	MergeScore     int           `json:"merge_score"` // 0-100
	Reasoning      string        `json:"reasoning"`
	LikelyBreaking bool          `json:"likely_breaking"`
	HasTests       bool          `json:"has_tests"`
	Issues         []ReviewIssue `json:"issues"`
	Labels         []string      `json:"labels"`
	Action         ReviewAction  `json:"action"`
	LLMUsed        string        `json:"llm_used,omitempty"`
	DurationMs     int64         `json:"duration_ms"`
	CostUSD        float64       `json:"cost_usd,omitempty"`
}

// AntiSlopResult is the output of anti-slop filter
type AntiSlopResult struct {
	Passed   bool              `json:"passed"`
	Score    int               `json:"score"` // 0-100
	Failures []string          `json:"failures,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Reasons  map[string]string `json:"reasons,omitempty"`
}

// TriageResult is the output of issue triage
type TriageResult struct {
	Action        string    `json:"action"` // close, label, confirm
	IssueType     IssueType `json:"issue_type"`
	Priority      Priority  `json:"priority"`
	Labels        []string  `json:"labels"`
	DuplicateOf   *int      `json:"duplicate_of,omitempty"` // issue number if duplicate
	Reasoning     string    `json:"reasoning"`
	IsActionable  bool      `json:"is_actionable"`
	SimilarIssues []int     `json:"similar_issues,omitempty"` // potential duplicates
}

// Event represents a normalized forge event
type Event struct {
	Type   EventType `json:"type"`
	Source string    `json:"source"` // github, gitlab, bitbucket
	Repo   RepoRef   `json:"repo"`
	Actor  Author    `json:"actor"`
	PR     *PR       `json:"pr,omitempty"`
	Issue  *Issue    `json:"issue,omitempty"`
	SentAt time.Time `json:"sent_at"`
}

// PipelineResult is the combined output of all pipeline stages
type PipelineResult struct {
	AntiSlop    *AntiSlopResult `json:"anti_slop,omitempty"`
	Review      *ReviewResult   `json:"review,omitempty"`
	Triage      *TriageResult   `json:"triage,omitempty"`
	ShouldClose bool            `json:"should_close"`
	CloseReason string          `json:"close_reason,omitempty"`
	Errors      []string        `json:"errors,omitempty"`
}
