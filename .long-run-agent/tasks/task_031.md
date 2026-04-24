# task_031

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_031.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

C02 ep serve — HTTP 服务启动、Web UI 入口

## 需求 (requirements)

- cmd/ep/commands/serve.go: ep serve 命令
- 启动 HTTP 服务 (默认 127.0.0.1:8080)
- 自动打开浏览器 (可选 --no-browser)
- 依赖注入组装
- 优雅关闭

## 验收标准 (acceptance)

- [ ] ep serve 命令可用
- [ ] HTTP 服务启动成功
- [ ] Web UI 可访问
- [ ] 优雅关闭正常

## 交付物 (deliverables)

- `cmd/ep/commands/serve.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: 服务启动逻辑
- [ ] **测试验证**: ep serve 测试
- [ ] **影响范围**: 主要入口

### 测试步骤
1. `ep serve --no-browser`
2. curl http://127.0.0.1:8080
