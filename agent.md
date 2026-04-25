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
eval_stats:
  gpt-4o:
    count: 5
    mean: 88.4
    min: 82
    max: 95
    last_run: 2026-04-25
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

# Run eval (requires evals/{asset-id}.md to exist)
ep eval run <found-id>

# IMPORTANT: Sync to update index with eval results
ep sync reconcile
```

### Complete Eval Workflow

**Prerequisite**: An eval prompt file must exist at `evals/{asset-id}.md`

```bash
# 1. Ensure eval prompt exists (contains rubric and test cases)
cat evals/<asset-id>.md

# 2. Run eval
ep eval run <asset-id>

# 3. Sync results into index (required before viewing in UI)
ep sync reconcile

# 4. View version history in Web UI
#    Navigate to: http://127.0.0.1:18080 → Assets → <asset> → Version History

# 5. Compare versions in Web UI
#    Navigate to: http://127.0.0.1:18080 → Compare
#    Select asset and two versions to see score delta
```

**Viewing eval results directly:**

```bash
# Check front matter of the prompt file
cat prompts/<asset-id>.md | head -30

# Key fields in front matter:
#   eval_history[]  - list of past eval runs
#   eval_stats{}    - Welford-aggregated stats per model
#   labels[]        - version labels (e.g., "prod", "stable")
```

### Batch Import
```bash
ep import ./my-prompts/*.txt
```
