# task_007

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_007.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

D03 Label 领域实体 + Storage — prod/dev/staging labels、Ent schema

## 需求 (requirements)

- domain/label.go: Label 实体
- storage/ent/schema/label.go: Ent schema
- storage/label_repository.go: Label Repository

## 验收标准 (acceptance)

- [ ] Label 实体完整
- [ ] Ent schema 生成成功
- [ ] Repository 可用

## 交付物 (deliverables)

- `internal/domain/label.go`
- `internal/storage/ent/schema/label.go`
- `internal/storage/label_repository.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Label 指针管理
- [ ] **测试验证**: CRUD 测试
- [ ] **影响范围**: prod 晋升、版本标记

### 测试步骤
1. Label.SetProd/Label.SetDev 测试
