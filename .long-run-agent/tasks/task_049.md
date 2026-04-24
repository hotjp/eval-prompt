# task_049

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_049.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

O01 slog 日志集成 — JSON Handler、trace_id/span_id 链路

## 需求 (requirements)

- internal/telemetry/logger.go
- slog JSON Handler
- trace_id/span_id 链路字段
- 敏感字段自动脱敏

## 验收标准 (acceptance)

- [ ] slog 集成正常
- [ ] JSON 输出格式正确
- [ ] trace_id/span_id 字段正常

## 交付物 (deliverables)

- `internal/telemetry/logger.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: slog 配置
- [ ] **测试验证**: 日志输出测试
- [ ] **影响范围**: 全链路日志

### 测试步骤
1. 日志 JSON 格式验证
2. trace_id 字段验证
