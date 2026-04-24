# task_005

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_005.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D01 Asset 领域实体 + Storage — Asset state machine、Validate、Ent schema、Repository

## 需求 (requirements)

- domain/asset.go: Asset 实体、AssetState 枚举、Validate()、CanPromote()
- storage/ent/schema/asset.go: Ent schema 定义
- storage/asset.go: Asset Repository
- 状态机: CREATED → EVALUATING → EVALUATED → PROMOTED → ARCHIVED

## 验收标准 (acceptance)

- [ ] Asset 实体完整
- [ ] 状态机转换规则正确
- [ ] Ent schema 生成成功
- [ ] Repository CRUD 可用

## 交付物 (deliverables)

- `internal/domain/asset.go`
- `internal/storage/ent/schema/asset.go`
- `internal/storage/asset_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 状态转换逻辑
- [ ] **测试验证**: CRUD 测试
- [ ] **影响范围**: 核心实体，其他模块依赖

### 测试步骤
1. Asset.Create/Update/Delete 测试
2. 状态转换测试
3. Ent query 测试
