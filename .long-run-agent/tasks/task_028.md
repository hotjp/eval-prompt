# task_028

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_028.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

G05 StaticHandler + Go embed — React 静态资源打包

## 需求 (requirements)

- gateway/static_handler.go: 静态资源服务
- Go embed 打包 React 构建产物
- SPA fallback (index.html)

## 验收标准 (acceptance)

- [ ] StaticHandler 实现完整
- [ ] Go embed 打包成功
- [ ] SPA fallback 正常

## 交付物 (deliverables)

- `internal/gateway/static_handler.go`
- `internal/gateway/static/` (embed 目录)

## 验证证据（完成前必填）

- [ ] **实现证明**: embed 配置
- [ ] **测试验证**: 静态资源访问测试
- [ ] **影响范围**: Web UI

### 测试步骤
1. Web UI 访问测试
2. SPA 路由测试
