# task_029

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_029.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

G06 中间件注册 — Recover/RequestID/Metrics/Logging/CORS

## 需求 (requirements)

- gateway/middleware/recover.go
- gateway/middleware/request_id.go
- gateway/middleware/metrics.go
- gateway/middleware/logging.go
- gateway/middleware/cors.go
- gateway/router.go: 中间件注册

## 验收标准 (acceptance)

- [ ] 所有中间件实现完整
- [ ] 中间件注册顺序正确
- [ ] CORS 仅允许 localhost

## 交付物 (deliverables)

- `internal/gateway/middleware/`
- `internal/gateway/router.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 中间件链
- [ ] **测试验证**: 中间件测试
- [ ] **影响范围**: 所有请求

### 测试步骤
1. 中间件执行顺序测试
2. CORS 头验证
