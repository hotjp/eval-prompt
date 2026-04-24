# task_051

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_051.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

O03 Health Endpoints — /healthz、/readyz

## 需求 (requirements)

- internal/gateway/health.go
- GET /healthz: 存活检查
- GET /readyz: 就绪检查 (DB + Redis)

## 验收标准 (acceptance)

- [ ] /healthz 正常响应
- [ ] /readyz 检查 DB + Redis
- [ ] 状态码正确

## 交付物 (deliverables)

- `internal/gateway/health.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Health 端点
- [ ] **测试验证**: curl 测试
- [ ] **影响范围**: 运维监控

### 测试步骤
1. curl /healthz 测试
2. curl /readyz 测试
