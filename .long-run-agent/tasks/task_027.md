# task_027

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_027.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

G04 MCPHandler + SSE — MCP 协议端点: prompts/list、prompts/get、prompts/eval

## 需求 (requirements)

- gateway/handlers/mcp_handler.go: MCPHandler
- GET /mcp/v1/sse: SSE 端点
- POST /mcp/v1: JSON-RPC 请求
- prompts/list: 返回可用 Prompt
- prompts/get: 获取 Prompt 内容
- prompts/eval: 触发 Eval

## 验收标准 (acceptance)

- [ ] MCPHandler 实现完整
- [ ] SSE 连接正常
- [ ] JSON-RPC 处理正常

## 交付物 (deliverables)

- `internal/gateway/handlers/mcp_handler.go`
- `internal/gateway/mcp/sse.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: MCP 协议实现
- [ ] **测试验证**: SSE 连接测试
- [ ] **影响范围**: Agent 集成

### 测试步骤
1. SSE 端点连接测试
2. JSON-RPC 调用测试
