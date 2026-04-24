# task_015

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_015.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

P05 TraceCollector 接口 — Span/Event 收集、JSONL trace 文件

## 需求 (requirements)

- service/interfaces.go: TraceCollector 接口定义
- StartSpan: 开始追踪
- RecordEvent: 记录事件
- Finalize: 输出 trace 文件路径

## 验收标准 (acceptance)

- [ ] TraceCollector 接口定义完整
- [ ] JSONL trace 文件生成

## 交付物 (deliverables)

- `internal/service/interfaces.go` (补充 TraceCollector)
- `plugins/eval/trace_collector.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: OpenTelemetry 集成
- [ ] **测试验证**: Trace 文件生成测试
- [ ] **影响范围**: EvalService

### 测试步骤
1. Trace 收集测试
2. JSONL 文件验证
