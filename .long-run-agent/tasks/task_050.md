# task_050

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_050.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

O02 OpenTelemetry — Trace + Metrics、Prometheus 端点

## 需求 (requirements)

- internal/telemetry/otel.go
- Trace: 全链路追踪
- Metrics: prompt_assets_total, eval_runs_total, eval_duration_seconds
- Prometheus 端点: :9090/metrics

## 验收标准 (acceptance)

- [ ] OpenTelemetry 集成正常
- [ ] Trace 链路正常
- [ ] Metrics 端点正常

## 交付物 (deliverables)

- `internal/telemetry/otel.go`
- `internal/telemetry/metrics.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: OTel 配置
- [ ] **测试验证**: Trace/Metrics 测试
- [ ] **影响范围**: 可观测性

### 测试步骤
1. Trace 链路测试
2. curl :9090/metrics 测试
