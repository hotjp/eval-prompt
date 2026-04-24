# task_026

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_026.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

G03 TriggerHandler — Prompt 匹配查询 API

## 需求 (requirements)

- gateway/handlers/trigger_handler.go: TriggerHandler
- POST /api/v1/trigger/match: 触发匹配
- GET /api/v1/trigger/prompts: 可用 Prompt 列表

## 验收标准 (acceptance)

- [ ] Trigger Handler 实现完整
- [ ] REST API 工作正常

## 交付物 (deliverables)

- `internal/gateway/handlers/trigger_handler.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Trigger API 逻辑
- [ ] **测试验证**: HTTP 测试
- [ ] **影响范围**: API 网关

### 测试步骤
1. curl 测试 Trigger API
