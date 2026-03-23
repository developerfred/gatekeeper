# GateKeeper — AI-Powered PR & Issue Validation Platform

## 1. Concept & Vision

**GateKeeper** is an open-source, self-hosted platform that validates PRs and issues before human review, eliminating the "review tax" developers pay when wading through AI-generated slop and low-quality contributions.

The core promise: *Maintainers stop drowning in noise. Developers get actionable, intelligent feedback before their PR ever needs a human eye.*

This isn't another AI comment bot. It's a **gatekeeper** — it filters the garbage automatically and routes genuine contributions to human reviewers faster.

**Personality**: No-nonsense, direct, technical. It speaks like a senior engineer reviewing your code: precise, actionable, never fluff. It respects developers' time.

---

## 2. Design Language

### Aesthetic Direction
**Industrial precision** — Think a senior engineer's terminal setup. Clean, information-dense, monospace where it matters. No soft gradients, no friendly cartoons. Dark by default (developers live in dark mode), with a sharp accent color.

### Color Palette
```
Background Primary:    #0D1117 (GitHub dark)
Background Secondary:  #161B22
Background Tertiary:   #21262D
Border:               #30363D
Text Primary:         #E6EDF3
Text Secondary:        #8B949E
Text Muted:           #484F58

Accent Success:       #3FB950 (merge ready, approved)
Accent Warning:       #D29922 (needs changes, medium risk)
Accent Danger:        #F85149 (blocking issues, slop detected)
Accent Info:          #58A6FF (informational)
Accent Purple:        #A371F7 (AI/automation signals)
```

### Typography
```
Headings:    JetBrains Mono (monospace, technical feel)
Body:        Inter (readable, professional)
Code/Diffs:  JetBrains Mono
Fallback:    system-ui, -apple-system, sans-serif
```

### Spatial System
- Base unit: 4px
- Spacing scale: 4, 8, 12, 16, 24, 32, 48, 64px
- Border radius: 6px (cards), 4px (inputs), 2px (badges)
- Dense information layout — no wasted whitespace

### Motion Philosophy
Minimal, functional:
- State transitions: 150ms ease-out
- Loading states: subtle pulse, not spinners
- No decorative animations — every motion communicates state

---

## 3. Layout & Structure

### 3.1 CLI Output

The CLI is the primary interface for developers. Output is designed for terminal readability:

```
┌─────────────────────────────────────────────────────────────┐
│ 🔍 GATEKEEPER REVIEW                                       │
├─────────────────────────────────────────────────────────────┤
│ PR: owner/repo#123 — feat: add user authentication        │
│ Author: @johndoe · 2 files changed · +127/-34             │
├─────────────────────────────────────────────────────────────┤
│                                                            │
│ ✅ MERGE READINESS: 78/100                                 │
│    Likely to pass CI · Has tests · Breaking: No           │
│                                                            │
│ ⚠️ ISSUES FOUND: 3                                        │
│                                                            │
│ [MEDIUM] auth/middleware.go:42                            │
│    Missing error handling on token expiration              │
│    → Add check for ErrTokenExpired and return 401          │
│                                                            │
│ [LOW] auth/handlers.go:89                                 │
│    Unused import: 'fmt'                                   │
│    → Remove or use fmt.Sprintf                           │
│                                                            │
│ [INFO] db/migrations/001.sql:15                           │
│    Consider adding index on user_id                        │
│    → ALTER TABLE sessions ADD INDEX idx_user_id (user_id)  │
│                                                            │
├─────────────────────────────────────────────────────────────┤
│ 🤖 Anti-Slop: PASSED (score: 92/100)                      │
│    Account age: 2y 3m · Past PRs: 47 (86% merged)        │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 GitHub Comment Format

Comments are posted as grouped blocks, not line-by-line noise:

```
## 🤖 GateKeeper Review — PR #123

### Merge Readiness: 78/100 ✅
Likely to pass CI · Has tests · Breaking: No

### Issues Found: 3

#### [MEDIUM] auth/middleware.go:42 — Missing error handling
The token expiration case is unhandled. This will cause silent auth failures.

```go
// Current
token, err := ValidateToken(rawToken)

// Add
if errors.Is(err, ErrTokenExpired) {
    return ErrUnauthorized
}
```

#### [LOW] auth/handlers.go:89 — Unused import
`fmt` is imported but never used.

#### [INFO] db/migrations/001.sql:15 — Missing index
Consider indexing `user_id` on sessions table for query performance.

---

*Reviewed by GateKeeper · ~45s · $0.00 (Ollama/qwen2.5-coder:32b)*
```

### 3.3 Web Dashboard (SaaS)

```
┌────────────────────────────────────────────────────────────────────┐
│  🏠 Dashboard   📊 Repos   ⚙️ Settings   👤 Account               │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  OVERVIEW                        Last 30 days                     │
│  ──────────────────────────────────────────────────────────────── │
│  PRs Reviewed:        1,247     ██████████████████████           │
│  Slop Blocked:          89     ██                                 │
│  Avg Review Time:      12s      ▓▓▓▓▓▓░░░░░░░░░░░░░░░░         │
│  Time Saved:        14.2 hrs    (estimated 43s per PR × 1,247)    │
│                                                                    │
│  MERGE READINESS DISTRIBUTION                                      │
│  ──────────────────────────────────────────────────────────────── │
│  90-100 (✅ Auto-approved):  ████████████ 623 PRs (50%)          │
│  70-89  (⚠️  Minor issues): ██████ 412 PRs (33%)               │
│  50-69  (🔴 Major issues):  ██ 156 PRs (13%)                   │
│  0-49   (🚫 Blocked):       █ 56 PRs (4%)                      │
│                                                                    │
│  RECENT ACTIVITY                                                    │
│  ──────────────────────────────────────────────────────────────── │
│  12:34  PR #1247 owner/repo · merge:82 · blocked: no             │
│  12:31  PR #1246 owner/repo · merge:45 · issues:3 · auto-close  │
│  12:28  Issue #892 owner/repo · duplicate · auto-closed          │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 4. Features & Interactions

### 4.1 Anti-Slop Filter

**Purpose**: Auto-close obvious garbage PRs before wasting any LLM cycles.

**Rules (31 total, inspired by Anti-Slop project)**:

| Category | Rule | Threshold | Action |
|----------|------|-----------|--------|
| Branch | Blocked terms | any match | fail |
| Branch | AI-generated patterns | confidence > 70% | warn |
| Title | Empty or template-only | length < 10 | fail |
| Title | Filler words density | > 40% filler | warn |
| Title | Emoji only | only emojis | fail |
| Description | Empty | length < 20 | fail |
| Description | Template not followed | missing sections | warn |
| Description | Bullet point density | > 80% bullets | warn |
| Description | Generic words | > 50% generic | warn |
| Description | No "why" explanation | missing context | warn |
| Commit | Generic messages | matches generic pattern | warn |
| Diff | Only whitespace | lines changed > 10% whitespace | fail |
| Diff | Only imports added | no logic changes | fail |
| Diff | Cosmetic only | semantic changes < 5% | fail |
| Author | Account age | < 7 days | warn |
| Author | Past merge ratio | < 30% merged | warn |
| Author | Profile completeness | no bio, no location | warn |
| Author | Previous PR quality | avg anti-slop score < 40 | fail |
| Files | Binary files only | all binary | fail |
| Files | No code files | no .go, .ts, .py, etc | warn |
| Honeypot | Trapped term | matches hidden pattern | fail |
| Honeypot | Missing required word | not in description | fail |
| Similarity | Exact duplicate | diff hash matches closed PR | fail |
| Similarity | Near duplicate | > 90% similarity to recent PR | warn |
| Rate | Burst submit | > 5 PRs/hour from same account | warn |
| Label | Required labels missing | project requires label | warn |
| Size | Diff too large | > 5000 lines | warn |
| Size | Too many files | > 100 files | warn |
| CI | No CI runs | project has CI, PR has none | warn |
| Test | Missing tests | changed code, no test files | warn |
| Security | Secrets in diff | detected api key, token | fail |

**Interaction**:
- Score 0-100. Default threshold: 70.
- Below threshold → auto-close with educational comment explaining why
- Above threshold → pass through to AI review

### 4.2 AI Code Review

**Purpose**: Provide intelligent, actionable feedback on PR quality.

**Sub-pipelines**:

#### A. Context Retrieval (RAG)
- Chunk diff into semantic hunks (max 500 lines each)
- Embed chunks using `text-embedding-3-small` or Ollama equivalent
- Retrieve top-10 most relevant code snippets from repo history
- Retrieve relevant commit messages from similar changes

#### B. Merge Readiness Score (Fast Path)
- Route to fast/cheap model (Haiku or qwen2.5:3b)
- Output: JSON schema `{ score: 0-100, reasoning: str, likely_breaking: bool, has_tests: bool }`
- Time budget: < 5 seconds
- If score > 90: post approval, minimal comments

#### C. Deep Review (Full Path)
- Route to capable model (Sonnet 4 or qwen2.5-coder:32b)
- Categories checked:
  1. **Correctness**: Logic errors, edge cases, boundary conditions
  2. **Security**: Injection, auth bypass, secrets, OWASP Top 10
  3. **Performance**: N+1 queries, unbounded loops, memory leaks
  4. **Maintainability**: Code duplication, complexity, naming
  5. **AI Failure Modes** (from research):
     - Optimistic catch blocks (empty or silent catches)
     - Unbounded retry loops
     - Global state mutation
     - Missing error propagation
     - Hardcoded credentials
     - Race conditions in async code
     - Missing timeouts/deadlines

#### D. Smart Comment Strategy
- Score > 85: Approve with minimal summary comment
- Score 60-85: Request changes, group issues by severity
- Score < 60: Request changes, blocking issues highlighted, mention anti-slop score correlation

### 4.3 Issue Triage

**Purpose**: Automatically classify, deduplicate, and prioritize GitHub issues.

**Capabilities**:
1. **Duplicate Detection**: Embed issue description, search for similar open issues (> 70% similarity → duplicate candidate)
2. **Classification**: Bug vs Feature vs Question vs Discussion (LLM-based)
3. **Priority Inference**: Critical/High/Medium/Low based on keywords, severity descriptions, affected components
4. **Actionability Check**: Is the issue specific enough to act on? (missing repro steps, vague requests)
5. **Auto-label**: Propose labels based on content
6. **Auto-close**: Duplicates, non-actionable, spam

### 4.4 Configuration

Each repository can have a `.gatekeeper.yml`:

```yaml
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
    enabled: false
    webhook_url: ""
```

---

## 5. Component Inventory

### 5.1 Merge Readiness Badge

| State | Visual | When |
|-------|--------|------|
| Excellent | 🟢 90-100 | Auto-approved, minimal comments |
| Good | 🟡 70-89 | Minor issues, request changes |
| Warning | 🟠 50-69 | Major issues, blocking |
| Poor | 🔴 0-49 | Auto-close recommended |
| Slop | ⛔ SLOP | Anti-slop triggered, auto-closed |

### 5.2 Issue Card

```
┌─────────────────────────────────────────────────────────────┐
│ 🐛 Bug: Login fails with special characters in password    │
│ #4521 · opened 2h ago by @contributor                     │
│                                                             │
│ Labels: [bug] [priority:high] [auth]                      │
│ Status: Will close as duplicate · Original: #3892          │
└─────────────────────────────────────────────────────────────┘
```

### 5.3 CLI Status Bar

```
gatekeeper review --pr https://github.com/owner/repo/pull/123

🔍 Analyzing...       [███████████████████████] 100%
✅ Merge Ready: 82/100
⚠️  Issues Found: 2
🤖 Anti-Slop: PASSED
⏱️  Time: 34s · 💰 Cost: $0.00
```

### 5.4 Error States

| Error | Message | Resolution |
|-------|---------|------------|
| LLM unavailable | "⚠️ LLM provider down. PR queued for manual review." | Fall back to anti-slop only, notify |
| Rate limited | "⚠️ Rate limit hit. Retrying in 30s..." | Exponential backoff, cache if possible |
| Config invalid | "❌ Invalid .gatekeeper.yml: field 'threshold' must be 0-100" | Show validation errors, use defaults |
| Auth failed | "❌ GitHub authentication failed. Check GITHUB_TOKEN." | Prompt re-auth, don't process PR |

---

## 6. Technical Architecture

### 6.1 Project Structure

```
gatekeeper/
├── cmd/
│   ├── gatekeeper/           # Main CLI entry
│   │   ├── main.go
│   │   └── commands/
│   │       ├── review.go
│   │       ├── triage.go
│   │       ├── diff.go
│   │       ├── init.go
│   │       ├── config.go
│   │       ├── daemon.go
│   │       └── root.go
│   ├── gatekeeperd/         # Background daemon (webhooks)
│   │   └── main.go
│   └── gatekeeper-web/      # Web dashboard (Next.js)
│
├── internal/
│   ├── core/
│   │   ├── engine.go        # Pipeline orchestrator
│   │   ├── config/          # Config loading/validation
│   │   ├── models/          # Domain models
│   │   └── metrics/         # Prometheus metrics
│   │
│   ├── anti_slop/
│   │   ├── checker.go       # Main checker
│   │   ├── rules/           # Individual rule implementations
│   │   ├── honeypot.go      # Honeypot trap logic
│   │   └── scorer.go        # Score calculation
│   │
│   ├── review/
│   │   ├── pipeline.go      # Review orchestration
│   │   ├── context/         # RAG retrieval
│   │   ├── merge_score.go   # Merge readiness scorer
│   │   ├── deep_review.go   # Full review
│   │   └── comment.go       # Comment formatting
│   │
│   ├── triage/
│   │   ├── pipeline.go      # Triage orchestration
│   │   ├── dedup.go         # Duplicate detection
│   │   ├── classifier.go    # Issue classification
│   │   └── prioritizer.go   # Priority inference
│   │
│   ├── llm/
│   │   ├── provider.go      # Provider interface
│   │   ├── openai.go        # OpenAI implementation
│   │   ├── anthropic.go     # Anthropic implementation
│   │   ├── ollama.go        # Ollama implementation
│   │   ├── router.go        # Smart routing (cost/speed)
│   │   └── cache.go         # LLM response cache
│   │
│   ├── forge/
│   │   ├── github.go        # GitHub integration
│   │   ├── gitlab.go        # GitLab integration
│   │   ├── bitbucket.go     # Bitbucket integration
│   │   └── event.go         # Normalized event format
│   │
│   ├── output/
│   │   ├── github_comment.go # GitHub PR comments
│   │   ├── slack.go         # Slack notifications
│   │   ├── discord.go       # Discord notifications
│   │   └── cli.go           # CLI output formatting
│   │
│   ├── storage/
│   │   ├── sqlite/          # SQLite adapter (self-hosted)
│   │   ├── redis/           # Redis adapter (SaaS)
│   │   └── postgres/        # PostgreSQL adapter (SaaS)
│   │
│   └── bot/
│       ├── slack_bot.go
│       ├── discord_bot.go
│       └── handlers.go      # Bot command handlers
│
├── pkg/
│   ├── diff/                # Diff parsing utilities
│   ├── embed/               # Embedding utilities
│   └── yaml/                # YAML utilities
│
├── web/                     # Next.js dashboard
│   ├── app/
│   │   ├── (dashboard)/
│   │   │   ├── page.tsx
│   │   │   ├── repos/
│   │   │   └── settings/
│   │   ├── (auth)/
│   │   │   ├── login/
│   │   │   └── callback/
│   │   └── api/
│   ├── components/
│   └── lib/
│
├── .github/
│   └── workflows/
│       ├── ci.yml           # Tests
│       ├── release.yml      # Goreleaser
│       └── action.yml       # GitHub Action
│
├── test/
│   ├── fixtures/            # Test data
│   │   ├── pr_slop.json
│   │   ├── pr_legit.json
│   │   └── issue_duplicate.json
│   └── integration/         # Integration tests
│
├── SPEC.md
├── README.md
├── LICENSE (MIT)
├── go.mod
├── go.sum
├── docker-compose.yml
└── Dockerfile
```

### 6.2 Core Domain Models

```go
// PR represents a pull request under review
type PR struct {
    Number     int
    Title      string
    Body       string
    Author     Author
    BaseBranch string
    HeadBranch string
    Diff       string
    Files      []File
    Commits    []Commit
    Labels     []string
    CreatedAt  time.Time
}

// Issue represents a GitHub issue under triage
type Issue struct {
    Number    int
    Title     string
    Body      string
    Author    Author
    Labels    []string
    State     IssueState
    CreatedAt time.Time
}

// Author represents a contributor
type Author struct {
    Login        string
    AccountAge   time.Duration
    PRCount      int
    MergeRatio   float64
    HasBio       bool
    HasLocation  bool
}

// ReviewResult is the output of the review pipeline
type ReviewResult struct {
    MergeScore      int
    Reasoning       string
    LikelyBreaking  bool
    HasTests        bool
    Issues          []ReviewIssue
    Labels          []string
    Action          ReviewAction // approve, request_changes, comment
    LLMUsed         string
    DurationMs      int64
    CostUSD         float64
}

// ReviewIssue is a single issue found in review
type ReviewIssue struct {
    Severity   Severity   // critical, high, medium, low, info
    Category   string     // security, correctness, performance, etc
    File       string
    Line       int
    Title      string
    Body       string
    Suggestion string
    Rule       string     // which rule detected it
}

// AntiSlopResult is the output of anti-slop filter
type AntiSlopResult struct {
    Passed   bool
    Score    int         // 0-100
    Failures []string    // list of failed rule names
    Warnings []string    // list of warning rule names
    Reasons  map[string]string // rule -> explanation
}
```

### 6.3 API Design

#### Internal Events (Normalized)

```go
// All forge events normalize to this format
type Event struct {
    Type    EventType // pr_opened, pr_updated, issue_opened, etc
    Source  string    // github, gitlab, bitbucket
    Repo    RepoRef   // owner/name
    Actor   Author
    Payload interface{} // raw payload from forge
    SentAt  time.Time
}

type RepoRef struct {
    Owner string
    Name  string
    Host  string // github.com, gitlab.com, self-hosted URL
}
```

#### CLI Commands

```bash
# Review a PR
gatekeeper review --pr https://github.com/owner/repo/pull/123

# Review local diff (pre-commit)
gatekeeper diff --from HEAD~1 --to HEAD

# Triage an issue
gatekeeper triage --issue https://github.com/owner/repo/issues/456

# Initialize repo with config
gatekeeper init

# Edit config
gatekeeper config edit

# Run daemon (webhook listener)
gatekeeper daemon --port 8080

# Show metrics
gatekeeper metrics

# Check status
gatekeeper status
```

### 6.4 Data Flow

```
PR Received
    │
    ▼
┌─────────────────┐
│  Event Router   │ ← Normalize from GitHub/GitLab/Bitbucket
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Anti-Slop      │ ← ~100ms, $0
│  Filter         │
└────────┬────────┘
         │
    ┌────┴────┐
    │ PASS?   │
    └────┬────┘
    NO   │   YES
    │    │
    ▼    ▼
┌──────┐ ┌─────────────────┐
│Auto  │ │  Context        │
│Close │ │  Retrieval      │
│      │ │  (RAG)         │
└──────┘ └────────┬────────┘
                  │
                  ▼
         ┌─────────────────┐
         │  Merge Ready    │ ← Fast model (~5s)
         │  Score          │
         └────────┬────────┘
                  │
         ┌────────┴────────┐
         │ Score > 90?      │
         └────────┬────────┘
           YES    │   NO
           │      │
           ▼      ▼
    ┌──────────┐ ┌─────────────────┐
    │ Approve  │ │  Deep Review   │
    │ + comment│ │  (~60s)        │
    └──────────┘ └────────┬────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │  Smart Comment │
                  │  + Labels      │
                  │  + Metrics     │
                  └─────────────────┘
```

### 6.5 Caching Strategy

```go
// Cache key structure
type CacheKey struct {
    Hash       string // SHA256(diff + context_hash)
    Model      string // "gpt-4o", "qwen2.5-coder:32b"
    ConfigHash string // SHA256(.gatekeeper.yml)
    TTL        time.Duration
}

// LLM response cache
type CacheEntry struct {
    Key       CacheKey
    Response  string
    CreatedAt time.Time
    HitCount  int
}

// Expected hit rates
// - PR updated (same diff): 30-50%
// - Fresh PR: 0%
// - RAG context retrieval: 60-70% (same files touched repeatedly)
```

---

## 7. Out of Scope (v1)

- [ ] GitLab / Bitbucket integration (v2)
- [ ] Web dashboard (v2)
- [ ] Slack/Discord bots (v2)
- [ ] RAG enhancement (v2)
- [ ] Metrics dashboard (v2)
- [ ] Multi-tenant SaaS (v2)
- [ ] Linear/Jira integration (v3)
- [ ] Team learning / adaptative patterns (v3)

---

## 8. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| PRs reviewed | 10,000+ | GitHub App events |
| Slop PRs blocked | 500+ | Auto-close count |
| Median review time | < 30s | End-to-end pipeline |
| LLM cost per PR | < $0.05 | With Ollama |
| False positive rate | < 10% | Legit PRs auto-closed |
| CLI adoption | 1,000+ installs | Homebrew + binary downloads |
| GitHub stars | 1,000+ | 90 days post-launch |

---

## 9. Competitive Positioning

| Feature | GateKeeper | PR-Agent | Anti-Slop | CodeRabbit |
|---------|------------|----------|-----------|------------|
| MIT License | ✅ | ❌ AGPL | ✅ AGPL | ❌ SaaS |
| Self-hosted | ✅ | ✅ | ✅ | ❌ |
| Anti-slop filter | ✅ | ❌ | ✅ | ❌ |
| AI review | ✅ | ✅ | ❌ | ✅ |
| Issue triage | ✅ | ❌ | ❌ | ❌ |
| Ollama/local | ✅ | ✅ | ❌ | ❌ |
| CLI-first | ✅ | ✅ | ❌ | ❌ |
| Web dashboard | v2 | ❌ | ❌ | ✅ |
| Linear/Jira | v3 | ❌ | ❌ | ❌ |
