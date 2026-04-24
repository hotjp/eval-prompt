# task_006

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_006.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D02 Snapshot 领域实体 + Storage — version/content_hash/commit_hash、Ent schema

## 需求 (requirements)

- domain/snapshot.go: Snapshot 实体
- storage/ent/schema/snapshot.go: Ent schema
- storage/snapshot_repository.go: Snapshot Repository

## 验收标准 (acceptance)

- [ ] Snapshot 实体完整
- [ ] Ent schema 生成成功
- [ ] Repository 可用

## 交付物 (deliverables)

- `internal/domain/snapshot.go`
- `internal/storage/ent/schema/snapshot.go`
- `internal/storage/snapshot_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Snapshot 版本管理
- [ ] **测试验证**: CRUD 测试
- [ ] **影响范围**: 版本历史、Git 集成

### 测试步骤
1. Snapshot.Create/Get/List 测试
