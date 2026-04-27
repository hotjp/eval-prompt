# RFC: Folder-Based Asset Structure

## Status

Proposed

## Motivation

当前每个 asset 以单文件 `.md` 存储，frontmatter 存元数据。这种模式在以下场景下不够用：

- **Skill 资产**：需要附带脚本代码、依赖文件（`handler.py`, `requirements.txt`）
- **MCP 资产**：需要附带服务配置、schema 定义
- **复合 Prompt**：一个 prompt 由 overview + 多个 `.md` 片段组成
- **外部资产**：希望零侵入地引用外部已有的 skill/agent 目录，避免重复拷贝

## Proposal

改用 **文件夹结构** 管理每个 asset，核心文件 `asset.yaml` 作为注册表，`main:` 字段指向入口文件。

## Folder Structure

```
repo/                                   # Git repository root（用户配置的独立 repo）
  assets/                               # 注册表（eval-prompt 管理，不侵入实际内容）
    prompts/
      my-prompt.yaml                   # ID = "my-prompt"
      system-design.yaml               # ID = "system-design"
    skills/
      calculator.yaml                  # ID = "calculator"
      github-agent.yaml                 # ID = "github-agent"

  prompts/                              # 实际 Prompt 内容
    my-prompt/
      overview.md
      part1.md
    system-design/
      overview.md
      part1-背景分析.md
      part2-技术选型.md

  skills/                               # 实际 Skill 内容
    calculator/
      handler.py                       # main 入口
      requirements.txt
    github-agent/
      agent.md
      config.json

  shared-utils/                          # 被 external 引用的共享文件
    common.py
```

**关键设计：**
- `assets/` 是注册表，与实际内容目录分离，不侵入
- 所有 `main:` 和 `external[].path` 均使用 **repo 相对路径**（从 repo root 算）
- 外部导入的 skill/agent 统一放到 repo 内对应 type 目录

## asset.yaml Schema

```yaml
# === 必需字段 ===
asset_type: string          # asset 类型：prompt | skill | agent | mcp | workflow | knowledge
name: string                 # 显示名称
main: string                 # 入口文件（repo 相对路径）

# === 基本信息 ===
description: string           # import 时留空，后续补充
tags: [string]               # import 时留空，后续补充
category: string              # import 时默认 "content"
state: string                # draft | published | archived | deleted | unavailable

# === 入口 ===
main_function: string        # 入口函数名（skill/agent），不写则按约定查找

# === 文件关联 ===
files:                       # asset 内所有文件，不写则自动扫描
  - path: string
    role: string            # main | script | config | doc | data
external:                    # 主文件外的关联文件（跨目录）
  - path: string
    role: string            # lib | template | data | doc

# === 版本与上游 ===
version: string              # 语义化版本
upstream:
  url: string               # 上游 Git 仓库 URL
  branch: string            # 上游分支
  last_sync: string         # 上次同步时间

# === 元数据（系统自动管理，用户不编辑）===
metadata:
  created_at: string
  created_by: string
  updated_at: string
  updated_by: string

# === 扩展字段（按需启用，结构严格保障）===
# 未来可按需扩展，示例：
# models: [string]
# biz_line: string
# timeout_seconds: number
# ...
```

**设计原则：**
- 必需字段只有 3 个（type, name, main），创建时必填
- 可选字段按需使用，不强制
- 扩展字段按需启用，SQLite 不存储空字段
- **向前兼容**：未来加新字段时，旧的 YAML 仍可正常读取，新字段默认为空
- metadata 由系统自动管理，不允许用户编辑

### Example: Skill with local script

```yaml
# ID = "calculator"
type: skill
name: "计算器 Skill"
main: skills/calculator/handler.py
description: "提供四则运算能力"
tags: [math, tool]
state: published
files:
  - path: skills/calculator/handler.py
    role: main
  - path: skills/calculator/requirements.txt
    role: config
```

### Example: External Skill (zero-invasion)

```yaml
# ID = "external-calculator"
type: skill
name: "外部计算器"
main: /Users/king/.local/skills/calculator/handler.py
external:
  - path: shared-utils/common.py
    role: lib
```

### Example: Composite Prompt

```yaml
# ID = "system-design-guide"
type: prompt
name: "系统设计指南"
main: prompts/system-design/overview.md
description: "完整的系统设计文档集"
external:
  - path: prompts/system-design/part1-背景分析.md
    role: doc
  - path: prompts/system-design/part2-技术选型.md
    role: doc
files:
  - path: prompts/system-design/overview.md
    role: main
  - path: prompts/system-design/part1-背景分析.md
    role: doc
  - path: prompts/system-design/part2-技术选型.md
    role: doc
```

## Main File Resolution

### 路径格式

所有路径均为 **repo 相对路径**（从 repo root 算）：

```yaml
main: skills/calculator/handler.py      # repo 相对路径
main: prompts/my-prompt/overview.md
main: ~/my-skills/handler.py            # ~ 展开为用户主目录（外部引用）
```

**解析规则：**
- 相对于 repo root 计算
- `~` 展开为用户主目录（外部引用场景）
- 绝对路径（以 `/` 或 `~` 开头）表示 repo 外外部引用

### 函数入口（Skill/Agent 类型）

当 `main:` 指向一个脚本文件时，系统按约定查找入口函数：

```yaml
# 方式一：使用 main_function 显式指定
main: skills/calculator/handler.py
main_function: process

# 方式二：不指定，按约定查找（默认）
main: skills/calculator/handler.py
```

**函数查找约定（按优先级）：**
1. `main()`
2. `handler()`
3. `process()`
4. `run()`
5. 文件级代码（无函数，如 shell script）

**按 type 的文件约定：**
| Type | main 指向的文件类型 |
|------|-------------------|
| skill | `*.py`, `*.ts`, `*.js`, `*.sh` |
| agent | `*.md`, `*.yaml` |
| prompt | `*.md` |
| mcp | `*.py`, `*.json`, `*.yaml` |
| workflow | `*.yaml`, `*.md` |
| knowledge | `*.md`, `*.json` |

## Indexing Strategy

### Scan Flow

```
scan assets/{type}/*.yaml for each type in {prompts, skills, agents, mcp, workflows, knowledge}:
    asset_yaml = "assets/{type}/{id}.yaml"     # ID = 文件名（去掉 .yaml）
    parse asset_yaml
    main_resolved = resolve(main: relative to repo root)
    is_external = is_path_outside_repo(main_resolved)

    if not is_external:
        index content + metadata
        watch file for changes
    else:
        index metadata only
        validate existence

    # 扫描 external 和 files 中的所有 path，建立完整 file_tree
```

### Source of Truth 层级

**核心原则：文件事实优先，YAML 和 SQLite 都是索引（缓存），置信度从高到低。**

```
Source of Truth（置信度从高到低）：
1. skills/agent 等实际内容文件夹    ← 最真实的 truth
2. assets/*.yaml                    ← 文件索引，置信度中等
3. SQLite                           ← 缓存，置信度最低
```

**修复时以更高置信度为准。**

### 一致性场景与处理

| 场景 | 实际文件夹 | YAML | SQLite | 处理方式 |
|------|-----------|------|--------|---------|
| A | ✓ | ✓ | ✓ | 正常 |
| B | ✓ | ✓ | ✗ | 补 SQLite（从 YAML 解析） |
| C | ✓ | ✗ | ✓ | 补 YAML（从文件夹重建） |
| D | ✓ | ✗ | ✗ | 提示用户：发现孤儿文件夹，需手动创建 YAML |
| E | ✗ | ✓ | ✓ | main 指向的文件没了 → 标记 `unavailable` |
| F | ✗ | ✗ | ✓ | 清理 SQLite 记录 |
| G | ✗ | ✓ | ✗ | 清理 YAML（文件都没了，YAML 也应删除） |

**详细说明：**

| 场景 | 处理逻辑 |
|------|---------|
| A | 三者一致，无需处理 |
| B | YAML 存在，SQLite 缺记录 → 解析 YAML，补入 SQLite |
| C | 文件夹存在但无 YAML → 从文件夹重建 YAML（自动推断 type、main 等） |
| D | 孤儿文件夹 → 提示用户手动创建 YAML 或忽略 |
| E | main 指向的文件不存在 → 标记 `state: unavailable`，提示用户修复 |
| F | SQLite 有记录但文件没了 → 清理 SQLite（文件是 truth） |
| G | YAML 有但文件没了 → 清理 YAML（文件是 truth） |

**注意：**
- 用户手动改名会导致场景 D（孤儿文件夹）或 E（文件丢失）
- 系统无法自动判断是改名还是真的丢了，需要用户判断

### Indexed Fields

- `id`: 文件夹名（相对于 type 目录），如 `calculator`、`github-mcp`
- `name`: from asset.yaml
- `type`: from asset.yaml
- `category`: from asset.yaml
- `tags`: from asset.yaml
- `state`: from asset.yaml
- `folder_path`: 完整相对路径，如 `skills/calculator`（用于 API 路由）
- `main_raw`: asset.yaml 中原始的 main: 值，如 `../../external/foo/bar.py`
- `main_resolved`: 相对于 repo root 的解析路径，如 `skills/calculator/handler.py`
- `main_function`: 可选，入口函数名（默认按约定查找）
- `content_hash`: SHA256 of resolved main file content
- `is_external`: bool（main_resolved 是否在 repo 外）
- `file_tree`: JSON，asset 所有关联文件（main + external）

### Index Storage

SQLite schema 扩展（待定义新表）：

```sql
CREATE TABLE assets (
    id TEXT PRIMARY KEY,           -- 文件名（去掉 .yaml），type 内唯一，如 "calculator"
    type TEXT NOT NULL,            -- prompt | skill | agent | mcp | workflow | knowledge
    name TEXT NOT NULL,
    asset_path TEXT NOT NULL,      -- asset.yaml 路径，如 "assets/skills/calculator.yaml"
    main TEXT NOT NULL,            -- repo 相对路径，如 "skills/calculator/handler.py"
    main_function TEXT,            -- 可选，入口函数名
    main_resolved TEXT NOT NULL,   -- 解析后的路径（同 main，因为都是 repo 相对路径）
    is_external BOOLEAN DEFAULT FALSE,
    content_hash TEXT,
    file_tree TEXT,                -- JSON: 所有关联文件（main + external）
    category TEXT,
    tags TEXT,                     -- JSON array
    state TEXT,
    updated_at DATETIME,
    INDEX idx_type (type),
    INDEX idx_asset_path (asset_path),
    INDEX idx_state (state)
);
```

### ID 语义化设计

使用文件名（去掉 .yaml）作为 ID 有以下优势：

| 方面 | 说明 |
|------|------|
| 天然唯一 | 同 type 目录下不允许同名 .yaml 文件 |
| 语义化 | `calculator` 比 `01AR5ZWHKQ...` 对 Agent 更友好 |
| 无特殊字符 | 不需要处理 ULID 的特殊字符，文件名即人类可读 |
| 多用户友好 | 不同用户 clone 到不同路径，ID 始终一致（相对于 repo） |
| 注册表分离 | asset.yaml 在 `assets/` 下，不侵入实际内容目录 |

## API Mapping

### Web UI Editing Scope

**Web UI 编辑功能保持简单：只编辑 `main:` 指向的主文件。**
- `PUT /assets/{id}/content` → 写入 `main:` 指向的文件
- external 和 files 中的其他文件不通过 Web UI 编辑（后续有需要再扩展）
- 多文件查看（files API）为只读浏览，不做编辑

### Existing APIs (modified behavior)

| API | Current | New Behavior |
|-----|---------|--------------|
| `GET /assets` | 扫描所有 `.md` frontmatter | 扫描 `assets/{type}/*.yaml` |
| `GET /assets/{id}` | 读文件 frontmatter | 读 `assets/{type}/{id}.yaml` |
| `GET /assets/{id}/content` | 返回 file body | 返回 `main:` 指向的文件内容 |
| `PUT /assets/{id}/content` | 写文件 body | 写入 `main:` 指向的文件 |
| `POST /assets` | 创建 `.md` | 创建 `assets/{type}/{id}.yaml` + 初始化实际内容目录 |
| `DELETE /assets/{id}` | 删除文件 | 删除 `assets/{type}/{id}.yaml` + 实际内容目录 |

### New APIs

| API | Scope | Behavior |
|-----|-------|----------|
| `GET /assets/{id}/files` | 只读 | 列出 asset 所有关联文件（main + external） |
| `GET /assets/{id}/files/{path}` | 只读 | 读取指定文件（穿透文件夹浏览） |
| `GET /assets/{id}/external` | 只读 | 返回 external 列表 |
| `PUT /assets/{id}/files/{path}` | **暂不支持** | 写入指定文件（后续扩展） |

### Content Resolution Flow

```
GET /assets/{id}/content
  1. Lookup asset by id in DB (id = filename without .yaml)
  2. Get asset_path (e.g., "assets/skills/calculator.yaml") and main (e.g., "skills/calculator/handler.py")
  3. Resolve:
       - If main starts with / or ~:
            → is_external = true
            → 展开 ~ 后直接读写外部文件系统
       - If main is repo relative path:
            → is_external = false
            → 读写 repo_root/main
  4. If main_function is set:
       → 从文件中提取指定函数
  5. Return content with metadata headers:
       - X-Content-Hash: sha256前8字节
       - X-Main-Path: main
       - X-Main-Function: main_function (if set)
       - X-Is-External: bool
```

### PUT /assets/{id}/content Flow

```
PUT /assets/{id}/content
  1. Lookup asset by id in DB
  2. Get main and is_external
  3. If is_external:
       - Return 403 Forbidden: "external asset is read-only"
  4. Write content to repo_root/main
  5. Update content_hash in DB
  6. Git add + commit asset.yaml + main 指向的文件
```

## Search 实现

### 搜索流程

```
用户搜索: "代码审查"
  ↓
【基础层：关键词匹配】
  命中字段: name + description + tags + keywords
  ↓
返回: 匹配结果
```

### LLM 增强层（可选，有 default LLM 时启用）

```
用户搜索: "代码审查"
  ↓
【LLM 意图识别】
  - 提取关键词: [代码, 审查, review, pr]
  - 理解意图: 找代码审查类 skill
  ↓
返回: 更精准的匹配结果
```

### 字段说明

| 字段 | 来源 | 说明 |
|------|------|------|
| name | asset.yaml | 资产名称 |
| description | asset.yaml + LLM | 详细描述 |
| tags | asset.yaml + LLM | 标签 |
| keywords | LLM 生成 | 关键词，增强搜索命中 |

**注意：**
- 无 default LLM 时，纯关键词匹配
- keywords 在 import 时由 LLM 生成，或用户手动补充
- intent 识别只在配置了 default LLM 时启用

## Git Semantics

### Local Asset (main: 指向 repo 内文件)

- Git 管理两部分：
  - `assets/{type}/{id}.yaml`（注册表）
  - `main:` 指向的文件（实际内容）
- `git add` 提交这两部分
- 删除 asset = 删除 `assets/{type}/{id}.yaml` + 实际内容目录

### External Asset (main: 指向 repo 外)

- Git 只管理 `assets/{type}/{id}.yaml`（注册表）
- `assets/{type}/{id}.yaml` 变更 = `git add` + commit
- 外部文件变更不触发 Git 事件
- 外部文件不存在时，系统标记 asset 为 `unavailable`

## State 变更语义

### State 值

| State | 含义 | API 行为 |
|-------|------|---------|
| `draft` | 草稿中，未发布 | 默认不显示，需显式查询 |
| `published` | 已发布，正常使用 | 默认出现在列表中 |
| `archived` | 已废弃，不再使用 | 默认不显示，需显式查询 |
| `deleted` | 已删除（软删除） | 不显示在任何列表中 | YAML 和内容保留 |
| `unavailable` | 索引存在但文件丢失 | 不显示，需修复 | YAML 保留 |

### State 变更操作

**核心原则：团队/企业资产管理使用逻辑删除（软删除），不物理删除文件。**

| 操作 | 文件变更 | SQLite 变更 | Git 变更 |
|------|---------|------------|---------|
| 发布 draft | `state: published` | `state` 更新 | commit yaml |
| 废弃 published | `state: archived` | `state` 更新 | commit yaml |
| 恢复 archived | `state: published` | `state` 更新 | commit yaml |
| **删除（软删除）** | `state: deleted` | `state` 更新 | commit yaml |
| **彻底删除** | 删除 yaml + 内容目录 | 删除记录 | commit 删除 |
| **文件丢失** | — | `state: unavailable` | — |

### 软删除 vs 彻底删除

| 类型 | 行为 | 触发时机 |
|------|------|---------|
| **软删除（推荐）** | 只改 `state: deleted`，YAML 和内容保留 | 用户点击"删除" |
| **彻底删除** | 物理删除 YAML + 内容目录 | 用户点击"彻底删除"（二次确认） |

**设计理由：**
- 软删除可恢复，团队协作更安全
- 彻底删除需要二次确认，防止误操作
- 差异检测时，`deleted` 状态的 asset 不影响 N=M 判断

### Orphan Index（索引存在，文件丢失）

当 `assets/*.yaml` 存在，但 `main:` 指向的文件/文件夹不存在时：

```
assets/skills/calculator.yaml  # 索引存在
skills/calculator/            # 但这个文件夹不存在了
```

**处理方式：**
1. 差异检测发现 main 指向路径不存在
2. 自动标记 `state: unavailable`
3. 提示用户：
   - "资产文件丢失，可从 Git 恢复"
   - "或彻底删除该资产"

**注意：**
- 用户手动改名会导致此情况（如把 `skills/calculator/` 改名为 `skills/advanced-calculator/`）
- 系统无法自动检测是改名还是真的丢了，需要用户判断

## Migration

**No backward compatibility.** 已有单文件 `.md` 资产不需要迁移。当前为零用户阶段，可直接使用新结构。

### Optional Migration Path (future)

如果未来需要迁移：

```bash
# 伪代码：单文件 → 文件夹转换
for each *.md with frontmatter:
    create folder: {type}/{asset-id}/
    move *.md to folder/main: {type}/{asset-id}/prompt.md
    generate asset.yaml from frontmatter
    git mv *.md folder/
```

## Open Questions

1. **软链接处理**：如果 `main:` 指向一个软链接，是否解引用读取？
2. **外部资产变更检测**：外部文件没有 Git tracking，如何通知用户内容已过期？
3. **循环引用检测**：`../../external/foo/` 下又有个 `asset.yaml` 引用回 `prompts/` 是否允许？
4. **文件 watcher**：外部文件变更时是否需要重新 index？

## 快速 Import

**目标：用户丢文件进 `.import/`，系统自动完成，无需点击按钮。**

### 核心流程

```
用户: 把 skill 文件夹丢进 .import/
  ↓
FileWatcher 检测到新文件夹
  ↓
AssetFileManager.Scan() 生成 asset.yaml
  ↓
Git add + commit
  ↓
前端实时显示导入成功（WebSocket / SSE）
```

### 实现要点

| 组件 | 说明 |
|------|------|
| FileWatcher | 监听 `.import/` 目录变更，检测到新文件夹自动触发 |
| AssetFileManager | 生成 asset.yaml + 移动文件到 repo |
| WebSocket/SSE | 实时通知前端显示导入结果 |
| Git Hook | 可选：post-commit hook 自动同步 |

### 预期时间

- 文件检测到导入完成：< 3 秒
- 用户体验：丢文件 → 看到"导入成功"提示

### CLI 命令

```bash
# 启动 FileWatcher 监听 .import/
ep asset watch

# 或一次性处理
ep asset import --from-inbox
```

### 与 Web UI 的区别

| 方式 | 触发 | 适用场景 |
|------|------|---------|
| **快速 Import** | 文件丢入自动触发 | 追求极致体验 |
| **手动 Import** | 用户点击按钮 | 需要预览确认 |

## CLI Commands

### ep asset import

导入外部资产到 repo 内。

```bash
ep asset import <外部路径> --type <type> [--name <name>]
```

**示例：**
```bash
# 导入外部 skill
ep asset import /Users/king/.local/skills/calculator --type skill

# 导入外部 agent
ep asset import /Users/king/agents/github-agent --type agent
```

**执行步骤：**
1. 复制内容到 repo 内对应 type 目录（如 `skills/calculator/`）
2. 生成 `assets/skills/calculator.yaml`（ID = 目录名）
3. 自动推断 `main:` 入口文件
4. Git add + commit

**注意：**
- ID 使用目录名，同 type 下不能重名
- 外部路径内容会被复制到 repo 内，不是软链接

## 异步流水线导入模式（Human-friendly）

针对人类用户设计：简单、直接、自动化。

### 概念

用户把文件丢进 `.import/` 导入箱，系统异步自动处理、分类、整理、组织，流水线化运行，用户无需关心细节。

### Folder Structure

```
repo/
  .import/                    # 导入箱（用户只管丢文件）
    README.md                # 说明扁平结构要求
    calculator/
      handler.py
    github_agent/
      agent.md

  assets/                     # 系统管理
    skills/
      calculator.yaml
      github_agent.yaml

  skills/                     # 实际内容
    calculator/
      handler.py
    github_agent/
      agent.md
```

### .import/README.md

```
Import Directory
===============
Please place each skill/agent in its own flat folder.
Nested structure is not supported.
Example:
  .import/my-skill/handler.py  ✓
  .import/nested/skill/        ✗
```

### AssetFileManager 统一流程

**`.import/` 导入和 `sync` 检查共用同一套 AssetFileManager 工具。**

```
AssetFileManager.Scan(source)     # source = .import/ 或 repo 任意目录
  ↓
解析文件夹结构，生成/更新 asset.yaml
  ↓
【LLM 渐进增强】（可选）
  → 补充 description
  → 提取 keywords/tags
  → 建议更准确的 category
  ↓
Git commit（如有变更）
```

**核心原则：**
- 函数/工具生成基础结构（type, name, main, files, state, category 等）
- LLM 补充语义信息（description, tags），非必需
- import 和 sync 走同一套逻辑，代码复用

### 处理流程

```
用户: 把文件复制到 .import/
用户: 点击"处理导入"（Web UI / API）
系统: AssetFileManager.Scan(.import/)
  ↓
  1. 扫描 .import/ 下的所有文件夹（仅支持扁平结构）
  2. 自动检测 type:
       - 含 __init__.py / handler.py → skill
       - 含 .md → agent 或 prompt（按内容分析）
       - 含 .yaml → workflow 或 mcp
  3. 生成 ID（使用文件夹名）
  4. 生成 asset.yaml（自动推断 main 入口，基础字段）
  5. 【LLM 渐进增强】补充 description + tags（如启用）
  6. 移动文件到对应 type 目录（如 skills/{id}/）
  7. Git add + commit
系统: 完成通知（Web UI 提示 / API 返回）
```

### 差异检测逻辑

**目的：** 发现未被索引的实际内容文件夹。

```
扫描 assets/{type}/*.yaml 数量 = N
扫描 {type}/*/ 文件夹数量 = M
→ N != M → 提示用户有未索引或孤儿文件夹
```

| 情况 | 含义 |
|------|------|
| N < M | 有文件夹未创建 asset.yaml |
| N > M | 有 asset.yaml 但实际内容被删除 |

**处理方式：** Web UI 提示用户检查缺失项，用户可选择修复或忽略。

### 两种处理模式

| 模式 | 行为 | 适用场景 |
|------|------|---------|
| **手动确认** | 扫描后显示预览，用户确认再执行 | 需要人工审核分类结果 |
| **全自动** | 扫描后直接执行，无需确认 | 信任自动分类，快速导入 |

### UI 设计

- **导入箱视图**：显示 `.import/` 文件列表 + 处理按钮
- **处理进度**：显示分类结果预览
- **完成后**：显示导入结果（成功/失败/需人工处理）

### 异步非阻塞模式

import 处理为异步后台执行，不阻塞用户操作。

```
用户: 点击"处理导入" → 立即返回
系统: 后台流水线处理
用户: 去做其他事情
系统: 完成后通知（Web UI toast / CLI 输出）
```

**预估时间：**
| 操作 | 单个 | 批量 10 个 |
|------|------|-----------|
| 生成 YAML | < 100ms | < 1s |
| 移动文件 | < 500ms | < 5s |
| SQLite INSERT | < 50ms | < 500ms |
| Git add + commit | < 500ms | < 5s |
| **总计** | **~1s** | **~10-15s** |

### Git Author 配置

import 时需要 Git author 信息。优先使用全局配置：

```bash
git config --global user.name "Your Name"
git config --global user.email "your@email.com"
```

如果未配置，使用默认值 `eval-prompt <agent@eval-prompt.local>`。

### 异常处理

| 异常情况 | 处理方式 |
|---------|---------|
| 文件名冲突 | 自动 rename（如 `calculator_1/`） |
| YAML 生成失败 | 跳过该文件，记录错误 |
| Git commit 失败 | 重试 1 次，仍失败则提示用户手动处理 |

**完成后返回：**
- 成功列表
- 失败列表（+ 原因，可点击查看详情）

### CLI 支持

```bash
# 处理 .import/ 下的所有文件
ep asset import --from-inbox

# 预览模式（不执行，只显示处理结果）
ep asset import --from-inbox --dry-run

# 检查索引差异（不修复）
ep asset check

# 检查并修复差异
ep asset check --fix
```

## 版本 Diff

**目标：查看 asset 在 git 历史中两个版本之间的内容变更。**

### 核心流程

```
用户: 选择 asset → 查看历史 → 选择两个版本对比
  ↓
GitDiff(diff between commit1 and commit2)
  ↓
展示: 文件内容差异（side-by-side 或 unified）
```

### Git 定位

| 资源 | Git 命令 |
|------|---------|
| asset.yaml 历史 | `git log assets/skills/calculator.yaml` |
| 内容文件历史 | `git log skills/calculator/handler.py` |
| 版本对比 | `git diff <commit1> <commit2> -- <file>` |

### API 设计

```bash
# 获取 asset 的 git 提交历史
GET /assets/{id}/history?limit=10

# 返回
{
  "commits": [
    { "hash": "abc123", "date": "2026-04-27", "message": "update description" },
    { "hash": "def456", "date": "2026-04-26", "message": "initial import" }
  ]
}

# 获取两个版本之间的 diff
GET /assets/{id}/diff?from=abc123&to=def456

# 返回
{
  "from": "abc123",
  "to": "def456",
  "changes": [
    { "file": "assets/skills/calculator.yaml", "type": "modified" },
    { "file": "skills/calculator/handler.py", "type": "modified" }
  ],
  "diff": "..."  # unified diff format
}
```

### Web UI

- 历史列表：显示 commit hash、message、date
- Diff 视图：side-by-side 或 unified diff 格式
- 支持按文件过滤（只看 yaml 变更 / 只看内容变更）

## 批量 Tag

**目标：选中多个 asset，批量添加/移除 tag。**

### 操作流程

```
Web UI: 勾选多个 asset → 选择"批量打 tag"
  ↓
选择 tag 或输入新 tag
  ↓
确认
  ↓
批量更新 asset.yaml（每个 asset 的 tags 字段）
  ↓
Git add + commit（批量 commit 或各自 commit）
```

### API 设计

```bash
# 批量添加 tag
POST /assets/batch/tag
Body: { "ids": ["skill-a", "skill-b"], "action": "add", "tag": "prod" }

# 批量移除 tag
POST /assets/batch/tag
Body: { "ids": ["skill-a", "skill-b"], "action": "remove", "tag": "draft" }
```

### Git Commit 策略

| 策略 | 说明 |
|------|------|
| 批量 commit | 多个 asset 的 tag 变更合成一个 commit |
| 各自 commit | 每个 asset 单独 commit，保持追踪粒度 |

### 检查入口统一

**import 场景和手动丢文件场景共用同一套检查和修复逻辑。** 区别只是触发入口：

| 场景 | 触发入口 |
|------|---------|
| 用户从 .import/ 处理 | 自动流水线 |
| 用户手动丢文件进 skills/ | 用户点击"检查" / CLI `ep asset check` |
| Server 启动 | 自动 reconcile |

**统一的修复流程：**
```
发现差异 → 显示问题列表 → 用户确认修复 → 批量处理（生成YAML + SQLite + commit）
```

### CLI 与 Server 解耦

**核心原则：CLI 直接操作 Git/SQLite，不调用 localhost API。Server 的 Indexer 独立运行。**

| 组件 | 职责 |
|------|------|
| CLI | 直接操作 SQLite + Git，独立完成 import/check/fix |
| Server | 启动时 reconcile，定期检查 |
| Web UI | 调用 API 展示问题，用户触发修复 |

**Server 启动时：**
```
Indexe.Reconcile() → 扫描 assets/*.yaml → 修正 SQLite
```

**Web UI 检查时机：**
| 时机 | 触发方 | 行为 |
|------|--------|------|
| Web UI 加载时 | 前端 | 调用 `GET /admin/repo-status` |
| 用户点击"检查"按钮 | 用户 | 调用 `POST /admin/reconcile` |
| Server 启动 | Server | 自动 reconcile |
