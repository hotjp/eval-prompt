# task_024

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_024.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

G01 PromptAssetHandler — Asset CRUD、版本管理 API

## 需求 (requirements)

- gateway/handlers/asset_handler.go: PromptAssetHandler
- GET /api/v1/assets: 列表
- GET /api/v1/assets/:id: 详情
- POST /api/v1/assets: 创建
- PUT /api/v1/assets/:id: 更新
- DELETE /api/v1/assets/:id: 删除

## 验收标准 (acceptance)

- [ ] Asset Handler 实现完整
- [ ] REST API 工作正常
- [ ] JSON 响应正确

## 交付物 (deliverables)

- `internal/gateway/handlers/asset_handler.go`

## 验证证据（完成前必填）

- [ ] **实现证明**: Handler 逻辑
- [ ] **测试验证**: HTTP 测试
- [ ] **影响范围**: API 网关

### 测试步骤
1. curl 测试 CRUD API
