# GateKeeper Roadmap

> AI-Powered PR & Issue Validation Platform

## Overview

GateKeeper is an open-source, self-hosted platform that validates PRs and issues before human review, eliminating the "review tax" developers pay when wading through AI-generated slop and low-quality contributions.

**License**: MIT
**Repository**: https://github.com/developerfred/gatekeeper

---

## Version History

### v0.1.0 - MVP Core (Current)

**Status**: ✅ Implemented

| Component | Status | Tests | Description |
|-----------|--------|-------|-------------|
| `anti_slop` | ✅ | 21 | 13 rules for detecting low-quality PRs |
| `config` | ✅ | 21 | YAML config loading and validation |
| `llm` | ✅ | 21 | OpenAI + Ollama provider abstraction |
| `github` | ✅ | 42 | GitHub REST + GraphQL API clients |
| `review` | ✅ | 19 | Merge readiness + deep review pipeline |
| `triage` | ✅ | 19 | Issue classification, deduplication, prioritization |
| `cli/review` | ✅ | 4 | `gatekeeper review` command |

---

## Roadmap

### 🔴 v0.2.0 - CLI Completion

**Target**: Complete the CLI command suite

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| `triage` command | P1 | ⏳ | `gatekeeper triage --issue <url>` |
| `diff` command | P1 | ⏳ | `gatekeeper diff --from HEAD~1 --to HEAD` for pre-commit |
| `init` command | P2 | ⏳ | `gatekeeper init` - create `.gatekeeper.yml` |
| `config` command | P2 | ⏳ | `gatekeeper config edit` |
| `root` command | P2 | ⏳ | Cobra root with `--version`, `--help` |
| `daemon` command | P3 | ⏳ | `gatekeeper daemon --port 8080` for webhooks |

### 🟡 v0.3.0 - GitHub Integration

**Target**: Full GitHub App experience

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| GitHub App manifest | P1 | ⏳ | One-click GitHub App installation |
| Webhook handler | P1 | ⏳ | Handle PR/issue events from GitHub |
| GitHub comment posting | P1 | ⏳ | Post review comments on PRs |
| GitHub Actions | P1 | ⏳ | `gatekeeper-action` for workflow integration |
| Repository configuration | P2 | ⏳ | Auto-detect `.gatekeeper.yml` from repo |

### 🟢 v0.4.0 - Multi-Provider LLM

**Target**: Support all major LLM providers

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| Anthropic Claude | P1 | ⏳ | `llm/anthropic.go` provider |
| LLM Router | P2 | ⏳ | Route to fast/cheap vs capable model |
| LLM Cache | P2 | ⏳ | Cache LLM responses by diff hash |
| Azure OpenAI | P3 | ⏳ | Azure OpenAI Service support |
| Vertex AI | P3 | ⏳ | Google Vertex AI support |

### 🔵 v0.5.0 - Output Channels

**Target**: Deliver results everywhere

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| GitHub Comment Formatter | P1 | ⏳ | Post grouped, actionable comments |
| Slack Integration | P2 | ⏳ | `gatekeeper/bot/slack.go` |
| Discord Integration | P2 | ⏳ | `gatekeeper/bot/discord.go` |
| Webhook Notifications | P3 | ⏳ | Generic webhook for custom integrations |
| Email Digest | P3 | ⏳ | Daily/weekly review summary |

### 🟣 v0.6.0 - Advanced Features

**Target**: Production-hardening

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| RAG Context Retrieval | P2 | ⏳ | Embed code chunks, retrieve relevant context |
| Prometheus Metrics | P2 | ⏳ | `gatekeeper metrics` + `/metrics` endpoint |
| Redis Cache | P2 | ⏳ | Distributed cache for multi-instance deploy |
| PostgreSQL Storage | P3 | ⏳ | Persistent storage for review history |
| SQLite Adapter | P3 | ⏳ | Self-hosted single-instance storage |

### ⚫ v1.0.0 - Stable Release

**Target**: Production-ready

| Task | Priority | Status | Description |
|------|----------|--------|-------------|
| Stable API | P1 | ⏳ | v1 API with breaking change policy |
| Comprehensive Docs | P1 | ⏳ | docs.gatekeeper.dev with guides |
| Docker Compose | P1 | ⏳ | `docker-compose up` self-hosted |
| GitHub Action v1 | P1 | ⏳ | Stable GitHub Action release |
| Binary Releases | P1 | ⏳ | Homebrew, apt, yum, Docker Hub |

---

## Out of Scope (v1)

These are planned for v2+:

- [ ] GitLab / Bitbucket integration
- [ ] Web Dashboard (SaaS)
- [ ] Linear / Jira integration
- [ ] Team learning / adaptive patterns
- [ ] Multi-tenant SaaS

---

## Architecture

```
gatekeeper/
├── cmd/
│   ├── gatekeeper/           # CLI
│   │   └── commands/         # review, triage, diff, init, config, daemon
│   └── gatekeeperd/         # Background daemon (future)
│
├── internal/
│   ├── anti_slop/           # ✅ 21 tests
│   ├── config/              # ✅ 21 tests
│   ├── core/                # engine, models
│   ├── forge/               # GitHub, GitLab, Bitbucket adapters
│   ├── github/              # ✅ 42 tests
│   ├── llm/                # ✅ 21 tests
│   ├── output/              # CLI, GitHub comment, Slack, Discord
│   ├── review/              # ✅ 19 tests
│   ├── triage/              # ✅ 19 tests
│   └── storage/             # SQLite, Redis, PostgreSQL
│
└── web/                     # Future: Next.js dashboard
```

---

## Success Metrics

| Metric | v0.1 Target | v1 Target |
|--------|-------------|------------|
| CLI installs | 100 | 1,000+ |
| GitHub stars | 50 | 1,000+ |
| PRs reviewed | 1,000 | 10,000+ |
| Slop blocked | 50 | 500+ |
| Contributors | 2 | 10+ |

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests first (TDD)
4. Ensure all tests pass (`go test ./...`)
5. Commit your changes
6. Push to the branch
7. Open a Pull Request

---

## License

MIT License - see [LICENSE](./LICENSE)
