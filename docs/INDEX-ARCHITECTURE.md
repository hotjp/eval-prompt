# 索引与存储架构设计

**版本**: V1.1
**状态**: 定稿
**日期**: 2026-04-25
**目标读者**: 开发团队、架构评审

## 重要更新 (V1.1)

架构核心转变：**数据库是索引，文件系统是唯一事实来源**

详见 [Issue #11](https://github.com/hotjp/eval-prompt/issues/11)

---

## 核心目标

| 目标 | 含义 |
|------|------|
| 1. 轻量 | 单二进制，无额外服务 |
| 2. 性能好 | 本地索引快，无网络延迟 |
| 3. 不干扰 | 远程 Git 改动不自动同步，本地改动自主可控 |
| 4. 快速同步 | eval 分数验证后的确定性优化可一键推送 |

---

## 一、核心定义：文件系统是唯一事实来源，数据库是索引

### 1.1 架构转变

| 之前（错误） | 现在（正确） |
|--------------|--------------|
| 数据库是元数据缓存 | 数据库只是索引 |
| 文件系统存内容 | 文件系统是唯一事实来源 |
| eval 历史存在数据库 | eval 历史存在 .md 文件 |

### 1.2 存储 vs 索引

```
┌─────────────────────────────────────────────────────────┐
│  文件系统（prompts/*.md）                               │
│  ─────────────────────────────────────────────────────  │
│  唯一的 prompt 内容                                     │
│  唯一的内容哈希（content_hash）计算来源                  │
│  Git 版本控制的载体                                     │
│  eval 历史（YAML front matter）                          │
│  labels（YAML front matter）                            │
└─────────────────────────────────────────────────────────┘
                          │
                          │ 用于搜索的字段
                          ▼
┌─────────────────────────────────────────────────────────┐
│  SQLite 数据库（索引）                                  │
│  ─────────────────────────────────────────────────────  │
│  用于快速搜索的字段：                                    │
│  - id, name, description, tags, state, content_hash   │
│                                                         │
│  不存储：                                               │
│  - eval 历史（存在文件里）                             │
│  - labels（存在文件里）                                │
│  - 快照详情（存在文件里）                              │
└─────────────────────────────────────────────────────────┘
```

### 1.3 content_hash 的作用

```go
// 创建 asset 时
content_hash = SHA256(文件内容)

// 检查是否需要更新（reconcile 时）
if dbAsset.content_hash != SHA256(文件内容) {
    // 文件已变更
}
```

`content_hash` 是连接索引和存储的桥梁。

### 1.4 .md 文件格式

```yaml
---
id: code-review
name: Code Review Prompt
description: 对 Go 代码进行结构化评审
version: v1.2.3
content_hash: sha256:abc123...
state: active
tags: [go, review, quality]
eval_history:
  - run_id: run-001
    score: 92
    model: gpt-4o
    date: 2026-04-25
    by: alice
labels:
  - name: prod
    snapshot: v1.2.3
    date: 2026-04-25
---
# Prompt Content

你是一位 Go 开发专家...
```

**结论**：
- **文件系统是唯一事实来源**（Content of Truth）
- **数据库是索引**（Index），不是存储

---

## 二、索引策略

### 2.1 数据库只索引必要字段

数据库作为索引，存储用于快速搜索的字段：
- id, name, description, tags, state, content_hash

**不存储**（这些在文件里）：
- eval_history
- labels
- Snapshot 详情

### 2.2 设计原则

| 原则 | 说明 |
|------|------|
| **只索引 HEAD** | 搜索的是"当前可用的 prompt"，不是历史版本 |
| **历史在文件里** | eval_history、labels 存在 .md 文件的 YAML front matter |
| **Git 历史用于审计** | `git log` 查看变更记录 |

---

## 三、同步机制：Reconcile 对账模式

### 3.1 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| **Reconcile 频率** | 启动时一次 + 可选手动 | 目标3：远程改动不自动同步 |
| **索引引擎** | 内存索引 | 目标1/2：轻量、性能好 |
| **Git Hook** | 不集成 | 目标3：远程 commit 不干扰本地 |

### 3.2 Reconcile 的语义

Reconcile（对账）的定义：**比对文件系统状态和索引（数据库）状态，修复不一致**。

**核心变化**：
- 之前：数据库是主源，索引文件系统
- 现在：文件系统是主源，索引数据库

**不是**：
- ❌ 事件驱动（每改一个文件就同步）
- ❌ 遍历所有 Git 历史

**而是**：
- ✅ 启动时全量扫描文件系统
- ✅ 比对 content_hash
- ✅ 差异部分更新索引（数据库）

### 3.3 触发方式

| 方式 | 触发时机 | 说明 |
|------|----------|------|
| **启动时** | `ep serve` 启动时自动执行一次 | 确保启动后状态正确 |
| **手动触发** | `ep sync reconcile` | 首次初始化、修复不一致 |

### 3.4 Reconcile 算法

```go
func Reconcile(ctx context.Context) ReconcileReport {
    report := ReconcileReport{}

    // 1. 扫描文件系统，获取当前所有 .md 文件
    files := scanPromptsDir("prompts/")

    for _, file := range files {
        // 2. 解析 YAML front matter 获取元数据
        metadata := parseFrontMatter(file.content)

        // 3. 计算 content_hash
        hash := SHA256(file.content)

        // 4. 查询索引（数据库）
        indexed, err := index.GetByFilePath(file.path)

        if !indexed {
            // 5. 文件存在，索引不存在 → 新增
            index.Create(metadata, hash)
            report.Added++
        } else if indexed.content_hash != hash {
            // 6. hash 不同 → 文件已变更，更新索引
            index.Update(metadata, hash)
            report.Updated++
        }
        // 7. hash 相同 → 无需操作
    }

    // 8. 清理孤儿索引（索引有，文件系统没有）
    for _, indexed := range index.GetAll() {
        if !fileExists(indexed.file_path) {
            index.Delete(indexed.id)
            report.Deleted++
        }
    }

    return report
}
```

### 3.5 好处

| 之前 | 现在 |
|------|------|
| 数据库是主源 | 文件系统是主源 |
| eval 历史不共享 | eval 历史在文件里，Git 自动共享 |
| 换设备数据丢失 | 完整恢复（Git clone 即可） |
| 两份数据要同步 | 单一数据源，天然一致 |

---

## 四、索引与存储的交互关系

### 4.1 各操作的同步行为

| 操作 | 文件系统（事实来源） | 数据库（索引） | 说明 |
|------|---------------------|---------------|------|
| `AssetService.Create()` | 创建 .md 文件（含 YAML front matter） | Reconcile 后自动创建索引 | 内容写入文件，索引自动同步 |
| `AssetService.Update()` | 更新 .md 文件 | Reconcile 后自动更新索引 | |
| `AssetService.Delete()` | 删除 .md 文件 | Reconcile 后自动删除索引 | |
| `ep serve` 启动 | - | Reconcile 同步 | 启动时全量同步 |
| `ep sync reconcile` | 读取 | 对账 | 手动触发 |

### 4.2 架构图（修正后）

```
┌─────────────────────────────────────────────────────────┐
│  L4-Service (AssetService)                             │
│  ─────────────────────────────────────────────────────  │
│  职责：业务逻辑、输入校验、事务边界                       │
│                                                         │
│  Create() → 写入文件系统（.md 含 YAML front matter）     │
│  Update() → 更新文件系统                                │
│  Delete() → 删除文件系统                                │
│                                                         │
│  注意：不直接操作索引，保持分层                          │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  Reconcile（启动时 + 手动触发）                         │
│  ─────────────────────────────────────────────────────  │
│  扫描文件系统 → 解析 YAML front matter → 更新索引       │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│  L1-Storage (SQLite) = 索引                           │
│  ─────────────────────────────────────────────────────  │
│  只存储用于搜索的字段：id, name, description, tags...   │
│  不存储 eval_history、labels（在文件里）                │
└─────────────────────────────────────────────────────────┘
```

**关键点**：
1. 文件系统是唯一事实来源
2. 数据库只是索引，不是存储
3. Service 层不直接操作索引（保持分层）
4. Reconcile 是同步机制

---

## 五、CLI 直接调用 Service 的可行性

### 5.1 当前问题

```go
// cmd/ep/commands/serve.go
indexer := search.NewIndexer()  // 内存索引，每次启动为空
triggerService := service.NewTriggerService(indexer, gitBridge)
```

- `ep serve` 启动时没有 Reconcile
- 新创建的 asset 搜索不到

### 5.2 解决方案

**Reconcile 作为启动时的初始化步骤**

```go
func serve(cmd *cobra.Command, args []string) error {
    // 1. 初始化存储
    client, err := storage.NewClient(cfg.Database)
    if err != nil {
        return err
    }

    // 2. Reconcile（启动时全量同步）
    indexer := search.NewIndexer()
    syncService := service.NewSyncService(client, gitBridge, indexer)
    report, err := syncService.Reconcile(ctx)
    if err != nil {
        log.Warn("reconcile failed", "error", err)
    } else {
        log.Info("reconcile completed", "added", report.Added, "updated", report.Updated)
    }

    // 3. 启动 HTTP 服务...
}
```

### 5.3 CLI 离线模式

```go
// ep asset list --local（离线模式，直接调用 Service）
func assetList(cmd *cobra.Command, args []string) error {
    client, err := storage.NewClient(cfg.Database)
    if err != nil {
        return err
    }
    defer client.Close()

    // 离线模式下，也需要 Reconcile 保证索引一致
    indexer := search.NewIndexer()
    syncService := service.NewSyncService(client, gitBridge, indexer)
    syncService.Reconcile(ctx)

    // 直接调用 Service（无 HTTP）
    assetService := service.NewAssetService(client)
    assets, err := assetService.ListAssets(ctx, nil)

    // ...
}
```

---

## 六、Git 同步操作（薄封装）

### 6.1 设计原则

- 不替代 Git，只是状态展示和便利封装
- Agent 用户可直接使用原生 Git
- Human 用户可用 `ep sync` 命令

### 6.2 命令设计

```bash
ep sync status       # 查看本地/远程状态：领先/落后/冲突
ep sync reconcile   # 手动触发 Reconcile
ep sync push        # git push（带资产状态审计）
ep sync pull        # git pull --ff-only（安全拉取）
ep sync diff <id>   # 查看某个 prompt 的远程 vs 本地差异
```

---

## 七、协作机制（Deferred）

### 7.1 当前状态

**协作机制暂不设计，让使用发展。**

### 7.2 待观察的问题

| 问题 | 说明 |
|------|------|
| 按人分支 vs 按设备分支 | 跨设备同步 + 团队协作如何共存？ |
| merge/rebase 帮助 | 提示词作为资产是否需要版本合并工具？ |
| 共识性提示词 | 如何区分"个人实验"和"团队共识"？ |

### 7.3 推荐的最佳实践（文档层面）

```markdown
## 分支策略推荐

### 个人使用
- 使用 main 分支即可
- 本地改动直接 commit

### 团队协作
1. 每个人 fork 自己的分支
2. 改动在个人分支开发
3. eval 分数验证后发起 PR 到 main
4. review 后合并

### 跨设备同步
1. 使用 Git remote（如 GitHub）
2. 设备 A: `git push`
3. 设备 B: `git pull`
```

**原则**：不强制流程，用文档推荐最佳实践，让用户自己找到合适的方式。

---

## 八、架构优势

### 8.1 对比

| | 之前（错误） | 现在（正确） |
|---|---|---|
| **事实来源** | 数据库 + 文件系统 | 文件系统单一来源 |
| **eval 历史** | SQLite（不共享） | .md 文件（Git 共享）|
| **换设备** | 数据丢失 | 完整恢复 |
| **团队协作** | 数据隔离 | 自动同步 |
| **冲突解决** | Git + SQLite 双系统 | 纯 Git（YAML 可 merge）|

### 8.2 为什么之前的设计是错的

之前的设计把数据库当作"元数据缓存"，但：
- **缓存 = 副本**
- **副本就涉及同步**
- **同步就带来复杂度**

改为"数据库是索引"后：
- **索引 ≠ 副本**
- **索引可以从数据重建**
- **不需要同步，只需要对账（Reconcile）**

---

## 九、Asset 生命周期与状态

### 9.1 状态设计（二分法）

删除/归档使用二分法，不做 DEPRECATED 等中间状态。

```
状态：ACTIVE | ARCHIVED
```

| 操作 | 效果 | 约束 |
|------|------|------|
| `ep asset create` | 默认 ACTIVE | - |
| `ep asset archive` | ACTIVE → ARCHIVED | 软删除，可恢复 |
| `ep asset restore` | ARCHIVED → ACTIVE | 从归档恢复 |
| `ep asset rm` | 只能删 ARCHIVED | ACTIVE 需先 archive |

**Agent 行为**：
- 触发/搜索 → 只返回 ACTIVE
- ARCHIVED 对 Agent 不可见

**简化理由**：
- DEPRECATED 中间态场景不明确
- "效果差但不敢删" → 用 ARCHIVED 代替
- 避免过度设计

---

## 十、版本与推荐标记

### 10.1 Snapshot vs 推荐标记语义分离

**Snapshot**：历史记录，"那一刻的内容"，**不关心优劣**
**推荐标记**：演进方向，"更优"，是**业务判断**

两者语义不同，不应混淆。

### 10.2 设计决策

```go
// Snapshot = 历史记录，不关心优劣
Snapshot {
    id, version, content_hash, content, ...
    // 没有 status 字段
}

// 推荐标记 = Asset 的业务决策
Asset {
    id, name, type, ...
    recommended_snapshot_id    // 当前推荐版本
    candidate_snapshot_ids     // 候选版本列表（可选）
}
```

### 10.3 版本语义

| 概念 | 语义 | 工具关心 |
|------|------|----------|
| Snapshot.version | 时间线记录（如 v1.0.0） | 不关心优劣 |
| recommended_snapshot_id | 当前推荐 | 工具使用 |
| candidate_snapshot_ids | 候选/实验 | 可选追踪 |

### 10.4 多版本并存支持（冷启动场景）

支持探索期允许多版本并存：

```go
Asset (code-review)
  ├── recommended_snapshot_id = "snap-v2"  // 推荐
  ├── candidate_snapshot_ids = ["snap-v3"] // 实验中
  │
  └── Snapshots
      ├── snap-v1 (历史)
      ├── snap-v2 (推荐)
      └── snap-v3 (候选)
```

### 10.5 Agent 触发行为

- 优先使用 `recommended_snapshot_id` 版本
- 如果没有推荐标记，使用最新的 `candidate_snapshot_ids`
- 不会返回 `archived` 状态的 Snapshot

### 10.6 版本演进流程

```
创建 v1 (自动成为 candidate)
    ↓
eval 通过，标记 recommended
    ↓
创建 v2 (成为新的 candidate)
    ↓
eval 通过，标记 v2 为 recommended，v1 保留历史
    ↓
... 循环
```

### 10.7 为什么不混用

| 混用方案 | 问题 |
|----------|------|
| Snapshot.version 带推荐语义 | v2 比 v1 好？还是时间在 v1 之后？语义不清 |
| Snapshot 加 status 字段 | 更新推荐时需要改多个 Snapshot，不简洁 |

---

## 十一、评测 Prompt 也是 Asset

### 11.1 问题

评测集（EvalCase）中的 prompt 也是 prompt，与被测 prompt 本质相同，只是用途/类型不同。

当前设计不一致：
- 被测 prompt → Asset → 文件系统
- 评测 prompt → EvalCase.prompt → 数据库

### 11.2 解决方案

评测 prompt 也应该是 Asset，只是 type 不同：

```go
Asset {
    id, name, type, ...
    type: "content" | "eval"  // 内容 prompt 或 评测 prompt
}
```

### 11.3 待讨论

1. 评测 prompt 改动了，之前的 eval 结果还算数吗？
2. 不同模型的评测 prompt 是不同的 Asset 还是同一个 Asset 的不同版本？
3. Reconcile 时，评测 prompt 也要索引到搜索吗？

---

## 十二、Eval 可靠性与多次验证

### 12.1 问题

在没有 ground truth 的情况下，如何建立对 Prompt 优化效果的信任？

### 12.2 挑战复杂度

| 挑战 | 说明 |
|------|------|
| **模型不确定性** | 单次 eval 分数不可信，模型有随机性 |
| **评测集覆盖度** | 评测集是真实场景的子集 |
| **跨模型泛化** | 在一个模型上优化，不一定对其他模型有效 |
| **评测集本身在变** | 业务场景变化，评测集需要维护 |

### 12.3 每次 Eval Run 需要记录

```json
{
    "eval_run_id": "run-xxx",
    "asset_id": "code-review",
    "snapshot_version": "v1.2.3",
    "model": "gpt-4o-2024-05-13",
    "eval_case_version": "v2.3",
    "score": 90,
    "deterministic_score": 0.95,
    "rubric_score": 90,
    "tested_by": "alice",
    "tested_at": "2026-04-25T10:00:00Z"
}
```

### 12.4 多次 Eval 统计

- 同一 Snapshot + 同一模型，跑 N 次
- 聚合：平均分、标准差、置信区间
- 区分：单次分数 vs 统计分数

### 12.5 待设计

1. 多次 eval 如何聚合？平均分？中位数？分布图？
2. "通过"的标准是什么？单次 >= 80？连续3次 >= 80？
3. 跨模型泛化能力如何呈现？
4. 分数差异多大算"显著"？

---

## 十三、待实现清单（架构转变后）

- [ ] `SyncService.Reconcile()` 实现（需更新：解析 YAML front matter）
- [ ] `ep sync reconcile` 命令
- [ ] `ep serve` 启动时 Reconcile
- [ ] Asset 状态：ACTIVE / ARCHIVED
- [ ] 推荐标记：recommended_snapshot_id
- [ ] .md 文件格式：YAML front matter 支持
- [ ] 文档：分支策略最佳实践

### 架构转变相关

- [ ] 更新 Asset Schema：从 SQLite 存储 eval_history 改为文件存储
- [ ] 更新 Reconcile 算法：解析 YAML front matter
- [ ] 移除 Snapshot 表（eval 历史存在文件里）
- [ ] 更新 EvalService：写入 .md 文件而非数据库

---

## 十四、相关设计文档

| 文档 | 内容 |
|------|------|
| [INDEX-ARCHITECTURE.md](./INDEX-ARCHITECTURE.md) | 本文档：核心架构决策 |
| [reconcile-design.md](./reconcile-design.md) | Reconcile 算法设计 |
| [asset-create-design.md](./asset-create-design.md) | Asset 创建流程设计 |
| [asset-lifecycle-design.md](./asset-lifecycle-design.md) | Asset 生命周期设计 |
| [recommended-mark-design.md](./recommended-mark-design.md) | 推荐标记设计 |
| [batch-import-design.md](./batch-import-design.md) | 批量导入设计 |

## 十五、相关 Issue

| Issue | 标题 |
|-------|-------|
| [#1](https://github.com/hotjp/eval-prompt/issues/1) | ep sync reconcile 命令未实现 |
| [#3](https://github.com/hotjp/eval-prompt/issues/3) | 批量导入提示词支持 |
| [#4](https://github.com/hotjp/eval-prompt/issues/4) | Asset 创建流程优化：支持直接粘贴内容 |
| [#6](https://github.com/hotjp/eval-prompt/issues/6) | README 缺少 eval-prompt 使用说明 |
| [#7](https://github.com/hotjp/eval-prompt/issues/7) | 评测 prompt 应视为 Asset，类型不同 |
| [#8](https://github.com/hotjp/eval-prompt/issues/8) | 简化删除/归档状态：只用 ACTIVE/ARCHIVED 二分法 |
| [#9](https://github.com/hotjp/eval-prompt/issues/9) | Prompt 优化效果如何证明：Eval 可靠性与多次验证机制 |
| [#10](https://github.com/hotjp/eval-prompt/issues/10) | 版本与推荐标记：Snapshot vs Version 语义分离 |
| [#11](https://github.com/hotjp/eval-prompt/issues/11) | 架构转变：数据库是索引，文件是唯一存储 |

---

**文档状态**：定稿（含 V1.1 所有设计决策）
