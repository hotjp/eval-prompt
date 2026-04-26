# eval-prompt

中文 | **[English](README.md)**

团队级 Prompt 资产管理工具。单个 Go 二进制文件，自带 Web UI，支持 CLI 和 MCP 协议 — 专为 AI Agent 读写、版本化管理、评测 Prompt 设计。

## 给 AI Agent 用

你可以通过以下方式与 eval-prompt 交互：
- **MCP 协议** — 作为 MCP client 连接，程序化读写 Prompt
- **CLI** — `ep asset create`、`ep eval run` 等命令
- **Web UI** — 人类友好界面 http://127.0.0.1:18080

### 你与 eval-prompt 的协作流程

```
1. 创建 Prompt:      ep asset create my-prompt
2. 编写内容:         ep asset update my-prompt
3. 创建版本:         ep snapshot create my-prompt
4. 运行评测:         ep eval run my-prompt
5. 分数达标:         ep snapshot create my-prompt  # 新版本
6. 重复 4-5 直到满意
```

### Prompt 文件格式

Prompt 是 Markdown 文件，带 YAML front matter：

```yaml
---
id: code-review
name: Code Review Prompt
version: v1.0.0
state: active
tags: [go, review]
eval_history:
  - run_id: run-001
    score: 85
    model: gpt-4o
    date: 2026-04-25
labels:
  - name: prod
    snapshot: v1.0.0
---
# Prompt 内容

你是 Go 代码评审专家...
```

### Agent 视角的架构

```
prompts/*.md  ←  你的 Prompt 文件（Git 版本控制）
     ↓
SQLite 索引   ←  快速搜索：id, name, tags, content_hash
     ↓
  MCP / CLI  ←  你访问 Prompt 的方式
```

**核心原则**：`.md` 文件是事实来源，不是数据库。始终编辑文件，不要直接操作数据库。

## 快速开始

```bash
# 1. 安装
curl -fsSL https://raw.githubusercontent.com/hotjp/eval-prompt/main/install.sh | sudo sh

# 2. 初始化项目
ep init ./my-prompts
cd ./my-prompts

# 3. 启动服务
ep serve
# 浏览器打开 http://127.0.0.1:18080

# 4. 创建第一个 Prompt
ep asset create my-first-prompt
ep asset update my-first-prompt  # 编辑内容

# 5. 创建版本并评测
ep snapshot create my-first-prompt
ep eval run my-first-prompt
```

## 命令参考

### 资产管理

```bash
ep asset list              # 列出所有 Prompt（仅 ACTIVE 状态）
ep asset create <id>       # 创建新 Prompt
ep asset get <id>          # 查看 Prompt 详情和内容
ep asset update <id>       # 更新 Prompt 内容（打开编辑器）
ep asset archive <id>      # 归档（软删除，可恢复）
ep asset restore <id>       # 从归档恢复
```

### 版本管理

```bash
ep snapshot list <id>           # 列出 Prompt 的所有版本
ep snapshot create <id>         # 创建新版本（自动递增 version）
ep snapshot diff <id> v1 v2     # 对比两个版本
```

### 评测

```bash
ep eval run <id> [--case-ids xxx]   # 对 Prompt 运行评测
ep eval report <run-id>             # 查看详细评测报告
ep eval compare <id> v1 v2          # 对比两个版本的评测分数
```

### Git 同步

```bash
ep sync reconcile   # 同步文件系统与索引（启动时自动运行）
ep sync status      # 查看本地与远程状态
ep sync push        # 推送到远程
ep sync pull        # 从远程拉取
```

## 配置

```yaml
# config.yaml
server:
  port: 18080

plugins:
  llm:
    enabled: true
    provider: openai        # 或: claude, ollama
    api_key: sk-xxx
    endpoint:              # 第三方 OpenAI 兼容接口
    api_path: /v1/chat/completions
    default_model: gpt-4o
```

或使用环境变量：

```bash
export APP_PLUGINS_LLM_API_KEY=sk-xxx
export APP_PLUGINS_LLM_DEFAULT_MODEL=gpt-4o
export APP_PLUGINS_LLM_ENDPOINT=https://api.groq.com
```

## 安装

```bash
# macOS / Linux / Git Bash
curl -fsSL https://raw.githubusercontent.com/hotjp/eval-prompt/main/install.sh | sudo sh

# Windows
# 开发中，预计一个月内发布 ⏳
# 请先 star 本仓库关注更新：https://github.com/hotjp/eval-prompt
```

**二进制命名：**

| 文件名 | 操作系统 | 架构 |
|--------|----------|------|
| `ep-darwin-arm64` | macOS | Apple Silicon |
| `ep-darwin-amd64` | macOS | Intel |
| `ep-linux-arm64` | Linux | ARM64 |
| `ep-linux-amd64` | Linux | x86_64 |
| `ep-windows-amd64.exe` | Windows | x86_64 |

**依赖：**
- macOS：需要 Xcode Command Line Tools
- Linux：需要 gcc 和 SQLite 开发库
- Windows：推荐使用 Git Bash 或 WSL

## License

MIT
