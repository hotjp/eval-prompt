# task_004

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_004.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

F04 L1-Storage 骨架 — SQLite client、ent generate、Outbox 轮询器

## 需求 (requirements)

- internal/storage/client.go: SQLite client 封装
- ent generate: 使用 ent 生成代码
- Outbox 轮询器: 5秒轮询，事务内处理
- 参考 docs/DESIGN.md Section 3.2 L1-Storage

## 验收标准 (acceptance)

- [ ] SQLite 连接正常
- [ ] ent generate 成功
- [ ] Outbox 轮询器工作

## 交付物 (deliverables)

- `internal/storage/client.go`
- `ent/schema/` (基础 schema)
- `internal/storage/outbox_poller.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: SQLite WAL 模式、连接池配置
- [ ] **测试验证**: 自动迁移测试
- [ ] **影响范围**: 所有数据持久化依赖

### 测试步骤
1. SQLite 数据库创建成功
2. ent schema 自动迁移
3. Outbox 轮询日志验证
