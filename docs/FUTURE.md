# Future Ideas (Long-shots)

记录尚未确定落地的想法，不承诺实现，仅供参考。

---

## Per-Project Repo Configuration

### 背景

当前 `ep repo switch` 是全局的，每个用户机器只有一个"当前仓库"。但实际可能有以下场景需要 per-project 配置：

- **外包团队**：每个客户项目对接不同资产仓库
- **Monorepo**：多个子项目各自关联不同资产
- **多租户 SaaS**：每个 workspace 独立资产库

### 设想

在项目目录下放一个 `.epconfig` 文件：

```yaml
# /project-root/.epconfig
repo: /Users/kingj/client-a-assets
assets_dir: prompts
evals_dir: .evals
```

### Auto-discovery 流程

```
cd /project-x
ep assets list
  → 查找 .epconfig (当前目录)
    → 查找 .epconfig (父目录)
      → ... → 找到或用全局 ~/.ep/lock.json
```

### 实现路径

1. `lock.json` 加一个字段 `project_config: true`
2. 启动时从当前目录向上查找 `.epconfig`
3. 找到则用 `.epconfig` 的值覆盖 lock.json 的 current
4. 作为 pip/npm 包分发 `ep` CLI

### 前提

需要先有真实用户需求驱动，当前只是 long-shot 想法。

---

## 其他 Long-shot

（待补充）

---

## JSONL 文件分片

### 背景

`.evals/calls/{execution_id}/calls.jsonl` 可能增长过大（10000+ 条记录），导致：
- Git diff 变慢
- 读取/写入性能下降

### 决策

**先不加限制，出问题再分片。**

### 可能的分片方案

```
.evals/calls/{execution_id}/
  calls_001.jsonl   # 1-1000 条
  calls_002.jsonl   # 1001-2000 条
  calls_003.jsonl   # ...
```

详见 [EVAL-STORAGE-DESIGN.md](./EVAL-STORAGE-DESIGN.md)

---

## Eval 历史数据归档

### 背景

`.evals/` 目录可能快速增长（GB 级别），进 Git 会导致：
- clone 慢
- 磁盘占用大
- 归档操作本身会产生大量 Git 变更

### 接口预留（本期实现）

```go
func (s *ExecutionFileStore) Archive(ctx context.Context, id string) error
func (s *ExecutionFileStore) IsArchived(ctx context.Context, id string) bool
```

### 长期方案：独立仓库存储

```
main-repo/
  .evals/           # 最近 30 天，进 Git

eval-history-repo/  # 独立仓库，按需 clone
  2026-03/
  2026-04/
```

- main-repo 保持轻量
- 历史数据在独立仓库，不污染主仓库
- 需要两套引用管理

### 当前决策

**预留 Archive 接口，具体实现放 long-shot。**

---

## Dataset 概念

### 背景

当前通过 Asset 数组做批量 eval，但未来可能需要：
- Dataset 复用（同一个测试集反复用）
- Dataset 版本管理
- Dataset 级别的权限控制
- Dataset 元数据（描述、标签）

### 当前方案

直接用 Asset 数组，不需要独立实体。

### 长期方案

```
Dataset {
  id, name, version
  asset_ids: [asset_id, ...]
  description
  tags
}
```

- Dataset 是独立实体，有自己的 lifecycle
- 可以对 Dataset 做 eval
- 支持版本演进

### 当前决策

**先不加，长期再评估。**

---

## MCP prompts/list Cursor Pagination Bug

### 问题

`handlePromptsList` 中 cursor 分页逻辑有 bug：

```go
// 第 180 行
if offset+limit < len(results) {
    nextCursor = results[offset+limit-1].ID
}
```

**Bug**: 当 `offset + limit >= len(results)` 时，nextCursor 不会被设置，但实际上可能还有更多数据（因为 slice 已经从 offset 开始截取了）。

### 正确逻辑

nextCursor 应该基于**原始结果集**判断，而不是截取后的 slice：

```go
if offset+limit < len(allResults) {
    nextCursor = allResults[offset+limit-1].ID
}
```

### 影响

目前 `prompts/list` 实际没有真正被调用分页，暂不影响使用。

### 修复方案

1. Search 返回时保留原始长度
2. 或者用 `hasMore := offset+limit < totalCount` 判断
