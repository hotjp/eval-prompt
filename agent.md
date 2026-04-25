# eval-prompt Agent Reference

You are a Claude Agent. This file teaches you how to use eval-prompt programmatically.

## Core Concepts

- **Prompts are files**: `prompts/*.md` — your prompt content + metadata
- **Database is an index**: SQLite stores only id, name, tags, content_hash for fast search
- **Filesystem is source of truth**: Edit `.md` files directly, database syncs via Reconcile
- **Works offline**: No network required after installation

## Quick Reference

```bash
# Search prompts
ep trigger match "code review" --top 3
ep asset list
ep asset list --tags agent

# Get prompt content
ep asset cat <asset-id>

# Create prompt
echo "# My Prompt" | ep asset create --name "my-prompt" --content "# My Prompt"

# Run eval
ep eval run <asset-id>

# Sync filesystem with index
ep sync reconcile
```

## File Format

```yaml
---
id: my-prompt
name: My Prompt
version: v1.0.0
content_hash: sha256:abc123...
state: active
tags: [agent, code-review, gpt-4o]
eval_history:
  - run_id: run-001
    score: 92
    model: gpt-4o
    date: 2026-04-25
---
# Prompt Content

Your prompt text here...
```

## Tag Classification

| Type | Tag | Example |
|------|-----|---------|
| Agent prompt | `agent` | `[agent, code-review]` |
| Skill | `skill` | `[skill, translation]` |
| Workflow | `workflow` | `[workflow, multi-step]` |
| System | `system` | `[system, jailbreak-detection]` |

## Directory Structure

```
./
├── prompts/          # Content prompts (*.md)
├── evals/           # Eval prompts (*.md)
├── eval-prompt.db   # SQLite index
└── ep               # The binary
```

## MCP Protocol

If running server (`ep serve`), connect via MCP at `/mcp/v1` endpoint for programmatic access.

## Examples

### Create a Code Review Prompt
```bash
ep asset create --name "code-review" --content "# Code Review Prompt

You are a code reviewer. Analyze the following Go code for bugs, performance issues, and style problems." --tags agent,go,review
```

### Search and Use
```bash
# Find relevant prompt
ep trigger match "review go code" --top 1

# Get content
ep asset cat <found-id>

# Run eval
ep eval run <found-id>
```

### Batch Import
```bash
ep import ./my-prompts/*.txt
```
