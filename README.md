# GateKeeper

**AI-Powered PR & Issue Validation Platform**

GateKeeper filters AI-generated spam and low-quality PRs before they waste your time, then provides intelligent code review using LLMs — all self-hosted and open source.

## Features

- **Anti-Slop Filter**: Automatically detect and close AI-generated spam PRs using heuristic rules (inspired by Anti-Slop)
- **AI Code Review**: LLM-powered review with support for OpenAI, Anthropic Claude, and local Ollama models
- **Issue Triage**: Automatic classification, deduplication, and prioritization of GitHub issues
- **Self-Hosted**: 100% local operation with Ollama — your code never leaves your infrastructure
- **Multi-Provider**: OpenAI, Anthropic, Ollama — bring your own API key or use local models

## Quick Start

### Install

```bash
# macOS/Linux
curl -fsSL https://gate.keeper.dev/install.sh | sh

# Or via Homebrew
brew install gatekeeper

# Build from source
git clone https://github.com/gatekeeper/gatekeeper
cd gatekeeper
go build -o gatekeeper ./cmd/gatekeeper
```

### Prerequisites

For local LLM inference (recommended):
```bash
# Install Ollama
brew install ollama

# Pull a code-capable model
ollama pull qwen2.5-coder:32b
```

### Usage

```bash
# Check if a PR passes anti-slop filters
PR_TITLE="Add feature" PR_BODY="Implements new feature" PR_BRANCH="feat/test" PR_AUTHOR="user" gatekeeper check

# Run AI review on a PR (requires Ollama running)
gatekeeper review

# Initialize configuration
gatekeeper init
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        INPUT LAYER                          │
│  GitHub App  │  GitHub Actions  │  CLI  │  Webhooks       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    PIPELINE ORCHESTRATOR                     │
│                                                              │
│  Anti-Slop Filter (Heuristic, ~100ms, $0)                    │
│       │                                                      │
│       ▼                                                      │
│  AI Review (LLM, ~30-90s)                                  │
│       │                                                      │
│       ▼                                                      │
│  Issue Triage (LLM, ~10s)                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      OUTPUT LAYER                           │
│  GitHub Comments  │  Slack/Discord  │  Metrics Dashboard    │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

Create `.gatekeeper.yml` in your repository:

```yaml
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
  auto_close_duplicates: true
```

## Anti-Slop Rules

| Rule | Description |
|------|-------------|
| `branch_name` | Blocks AI-generated branch patterns |
| `title_length` | Rejects titles < 10 chars |
| `description_length` | Rejects empty descriptions |
| `diff_whitespace_only` | Catches cosmetic-only diffs |
| `diff_imports_only` | Catches PRs that only add imports |
| `account_age` | Warns on new accounts |
| `merge_ratio` | Warns on low PR merge rate |
| `honeypot` | Catches hidden AI markers |

## License

MIT License - see [LICENSE](LICENSE)
