# eval-prompt

**[中文](README_zh.md)** | English

Version-controlled AI asset management. Git for versioning, SQLite for indexing, accessible via Web UI / CLI / MCP — all data stays local and private.

## The Problem We Solve

| Pain Point | What eval-prompt Does |
|---|---|
| Prompts scattered everywhere (ChatGPT, Notion, Slack) | Centralized Git repo, searchable |
| Version chaos (which is latest? which is in prod?) | Git history, version tags |
| No way to measure prompt quality | Eval with deterministic + rubric scoring |
| Agents can't reuse prompt knowledge | MCP protocol for agent consumption |

## Architecture

```
Filesystem (prompts/*.md) ← Source of Truth
         ↓
SQLite Index ← Fast search (id, name, tags, content_hash)
         ↓
Web UI / CLI / MCP ← Access layer
```

**Core principle**: Edit `.md` files directly. The database is an index, not the source.

## Performance

Built with Go — no JVM, no Node.js, no Python runtime.

| Metric | Value |
|--------|-------|
| Cold start | <10ms |
| Memory usage | ~16MB |
| CLI command | <10ms |
| SQLite query | <5ms |
| Binary size | ~28MB |

Compared to Python/TypeScript tools: **10x faster**, **5x less memory**.

## Quick Start

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/hotjp/eval-prompt/main/install.sh | sudo sh

# 2. Initialize project
ep init ./my-prompts
cd ./my-prompts

# 3. Start server
ep serve
# Open http://127.0.0.1:18880

# 4. Create a prompt
ep asset create my-prompt --content "# My Prompt\n\nYou are an expert..."

# 5. Run eval (requires evals/my-prompt.md to exist)
ep eval run my-prompt

# 6. Sync results — REQUIRED after every eval
ep sync reconcile

# 7. View in Web UI: Assets → my-prompt → Version History
```

## Prompt File Format

```yaml
---
id: my-prompt
name: My Prompt
version: v1.0.0
state: active
tags: [review, go]
eval_history:
  - run_id: run-001
    score: 92
    model: gpt-4o
    date: 2026-04-25
eval_stats:
  gpt-4o:
    count: 5
    mean: 88.4
    min: 82
    max: 95
    last_run: 2026-04-25
labels:
  - name: prod
    snapshot: v1.0.0
---
# Prompt Content

You are an expert at...
```

**eval_stats** uses Welford's algorithm for incremental mean/variance — updated automatically after each `ep eval run`.

## For AI Agents

> **Important**: Read [agent.md](./agent.md) for complete command reference.

**MCP protocol** — connect programmatically:

```bash
ep trigger match "SQL injection detection" --top 3
ep asset get common/code-review
ep eval run common/code-review
```

**CLI commands:**

```bash
ep asset list              # List all prompts
ep asset create <id>       # Create new prompt
ep snapshot create <id>    # Create new version
ep eval run <id>           # Run eval (requires evals/{id}.md)
ep sync reconcile          # Sync filesystem with index (required after eval)
```

**Complete eval workflow:**

```bash
# 1. Run eval (requires evals/{asset-id}.md to exist)
ep eval run <asset-id>

# 2. Sync results into index — REQUIRED before viewing in UI
ep sync reconcile

# 3. View in Web UI:
#    - Version History: Assets → <asset> → History tab
#    - Compare: Compare tab → select asset and two versions
```

## Features

| Feature | Description |
|---|---|
| **Git Version Control** | All prompts in Git, history + diff + team collaboration |
| **Eval Verification** | Deterministic checker + LLM rubric grader,量化质量 |
| **MCP Protocol** | Agent can query, fetch, and eval prompts programmatically |
| **Cross-Model Adaptation** | Auto-adjust format for Claude ↔ GPT ↔ local models |
| **Sandbox Security** | File isolation, command whitelist, execution timeout |

## Configuration

```yaml
# config.yaml
server:
  port: 18080

plugins:
  llm:
    enabled: true
    provider: openai
    api_key: sk-xxx
    endpoint:           # for third-party APIs (e.g., Groq)
    api_path: /v1/chat/completions
    default_model: gpt-4o
```

Or environment variables:

```bash
export APP_PLUGINS_LLM_API_KEY=sk-xxx
export APP_PLUGINS_LLM_DEFAULT_MODEL=gpt-4o
export APP_PLUGINS_LLM_ENDPOINT=https://api.groq.com
```

## Installation

```bash
# macOS / Linux / Git Bash
curl -fsSL https://raw.githubusercontent.com/hotjp/eval-prompt/main/install.sh | sudo sh

# Windows
# Coming soon ⏳ — star the repo to get notified: https://github.com/hotjp/eval-prompt
```

**Requirements:**
- macOS: Xcode Command Line Tools
- Linux: gcc + SQLite dev libraries

## License

MIT
