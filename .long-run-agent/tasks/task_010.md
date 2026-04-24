# task_010

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_010.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D06 ModelAdaptation Storage — 跨模型适配记录

## 需求 (requirements)

- domain/model_adaptation.go: ModelAdaptation 实体
- storage/ent/schema/model_adaptation.go: Ent schema
- storage/model_adaptation_repository.go

## 验收标准 (acceptance)

- [ ] ModelAdaptation 实体完整
- [ ] Ent schema 生成成功
- [ ] Repository 可用

## 交付物 (deliverables)

- `internal/domain/model_adaptation.go`
- `internal/storage/ent/schema/model_adaptation.go`
- `internal/storage/model_adaptation_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 适配记录管理
- [ ] **测试验证**: CRUD 测试
- [ ] **影响范围**: ModelAdapter 插件

### 测试步骤
1. ModelAdaptation.Create/List 测试
