# i18n 国际化设计方案

## 1. 概述

为 eval-prompt 项目实现完整的国际化支持，同时覆盖 CLI 命令行工具和 Web UI 界面。

### 1.1 目标
- CLI 和 Web UI 共用同一套语言包
- 支持中文（zh-CN）和英文（en-US）
- 最小化侵入性，不改变现有代码结构
- 易于维护和扩展

### 1.2 现状

**CLI 硬编码消息（部分）：**
```
资产已创建、资产已归档、资产已删除
Eval 运行已启动、Eval 已取消
启动服务、服务已启动、API 端点、SSE 端点
初始化 prompt assets 仓库、Git 仓库初始化完成
对账完成、新增、更新、删除
```

**Web UI**
- 使用 Ant Design 组件
- 无自定义文本国际化

---

## 2. 设计原则

### 2.1 共用语言包
- 语言包统一存放在 `i18n/locales/` 目录
- Go 和 Web 各自用自己的加载器读取同一套语言定义文件（YAML 格式）

### 2.2 消息 Key 命名规范
```
{模块}_{动作}_{描述}

示例：
asset_create_success     # 资产创建成功
asset_archive_success    # 资产归档成功
eval_run_started         # Eval 运行已开始
serve_started           # 服务已启动
common_cancel           # 取消
common_confirm          # 确认
common_error            # 错误
common_loading          # 加载中
```

### 2.3 消息参数
```yaml
# 使用 {{.Name}} Go 模板语法
asset_delete_success: "资产已删除: {{.ID}}"
eval_run_progress: "正在运行 Eval ({{.Current}}/{{.Total}})"
```

---

## 3. 架构设计

### 3.1 文件结构

```
eval-prompt/
├── i18n/
│   ├── locales/
│   │   ├── zh-CN.yaml      # 中文语言包
│   │   └── en-US.yaml       # 英文语言包
│   └── README.md            # 语言包编辑指南
│
├── internal/
│   └── i18n/
│       ├── i18n.go          # 核心: T() 函数、语言切换
│       ├── loader.go         # YAML 文件加载
│       └── messages.go      # 消息 key 常量定义
│
└── web/
    └── src/
        ├── i18n/
        │   ├── index.ts     # i18next 配置
        │   ├── zh-CN.json   # Web 中文翻译
        │   └── en-US.json   # Web 英文翻译
        └── hooks/
            └── useTranslation.ts
```

### 3.2 Go CLI i18n 实现

```go
// internal/i18n/i18n.go

var currentLang = "zh-CN"  // 默认中文

// T returns the localized string for the given key
func T(key string, args ...any) string {
    return getMessage(key, args...)
}

// SetLang sets the current language
func SetLang(lang string) {
    currentLang = lang
}

// getMessage retrieves message from loaded locales
func getMessage(key string, args ...any) string {
    // 从已加载的语言包中查找
    // 支持参数替换
}
```

**使用方式：**
```go
// Before
fmt.Printf("资产已创建: %s\n", id)

// After
fmt.Printf("%s: %s\n", i18n.T("asset_create_success"), id)
fmt.Println(i18n.T("asset_create_success", "id", id))  // 另一种方式
```

### 3.3 Web i18n 实现

使用 `i18next` + `react-i18next`：

```typescript
// web/src/i18n/index.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

i18n
  .use(initReactI18next)
  .init({
    lng: localStorage.getItem('lang') || 'zh-CN',
    resources: {
      'zh-CN': { translation: require('./zh-CN.json') },
      'en-US': { translation: require('./en-US.json') },
    },
  });

export default i18n;
```

**使用方式：**
```tsx
// Before
<span>加载中...</span>

// After
<span>{t('common_loading')}</span>
```

### 3.4 语言检测顺序

1. CLI: 环境变量 `EP_LANG`
2. CLI: `LANG` 环境变量（如 `en_US.UTF-8`）
3. Web: `localStorage.lang`
4. Web: 浏览器 `navigator.language`
5. 默认: `zh-CN`

---

## 4. 语言包设计

### 4.1 zh-CN.yaml 结构

```yaml
# 资产操作
asset_create_success: "资产已创建: {{.ID}}"
asset_archive_success: "资产已归档: {{.ID}}"
asset_restore_success: "资产已恢复: {{.ID}}"
asset_delete_success: "资产已删除: {{.ID}}"
asset_not_found: "资产不存在: {{.ID}}"
asset_state_conflict: "请先 archive"

# Eval 操作
eval_run_started: "Eval 运行已启动: {{.ID}}"
eval_run_status: "状态: {{.Status}}"
eval_cancel_success: "Eval 已取消: {{.ID}}"
eval_compare_title: "{{.AssetID}}: {{.V1}} vs {{.V2}}"
eval_score_delta: "得分差: {{.Delta}}"

# 服务启动
serve_starting: "启动服务: http://{{.Addr}}"
serve_started: "服务已启动"
serve_api_endpoint: "API 端点: http://{{.Addr}}/mcp/v1"
serve_sse_endpoint: "SSE 端点: http://{{.Addr}}/mcp/v1/sse"
serve_opening_browser: "正在打开浏览器..."

# 初始化
init_title: "初始化 prompt assets 仓库: {{.Path}}"
init_git_complete: "Git 仓库初始化完成"
init_lock_added: "仓库已添加到锁文件"
init_complete: "初始化完成"
init_serve_hint: "运行 'ep serve' 启动服务"

# 对账
sync_reconcile_done: "对账完成"
sync_added: "新增: {{.Count}}"
sync_updated: "更新: {{.Count}}"
sync_deleted: "删除: {{.Count}}"
sync_error: "错误: {{.Error}}"

# 通用
common_cancel: "取消"
common_confirm: "确认"
common_error: "错误"
common_loading: "加载中"
common_success: "成功"
common_warning: "警告"
common_retry: "重试"

# 错误消息
err_asset_not_found: "资产不存在"
err_invalid_id: "无效的 ID"
err_git_not_initialized: "尚未初始化任何仓库，请先运行 'ep init <path>'"
err_storage_not_configured: "存储未配置"
```

### 4.2 en-US.yaml 结构

```yaml
# Asset Operations
asset_create_success: "Asset created: {{.ID}}"
asset_archive_success: "Asset archived: {{.ID}}"
asset_restore_success: "Asset restored: {{.ID}}"
asset_delete_success: "Asset deleted: {{.ID}}"
asset_not_found: "Asset not found: {{.ID}}"
asset_state_conflict: "Please archive first"

# Eval Operations
eval_run_started: "Eval run started: {{.ID}}"
eval_run_status: "Status: {{.Status}}"
eval_cancel_success: "Eval cancelled: {{.ID}}"
eval_compare_title: "{{.AssetID}}: {{.V1}} vs {{.V2}}"
eval_score_delta: "Score delta: {{.Delta}}"

# Server
serve_starting: "Starting server: http://{{.Addr}}"
serve_started: "Server started"
serve_api_endpoint: "API endpoint: http://{{.Addr}}/mcp/v1"
serve_sse_endpoint: "SSE endpoint: http://{{.Addr}}/mcp/v1/sse"
serve_opening_browser: "Opening browser..."

# Initialization
init_title: "Initializing prompt assets repository: {{.Path}}"
init_git_complete: "Git repository initialized"
init_lock_added: "Repository added to lock file"
init_complete: "Initialization complete"
init_serve_hint: "Run 'ep serve' to start the server"

# Sync
sync_reconcile_done: "Reconcile complete"
sync_added: "Added: {{.Count}}"
sync_updated: "Updated: {{.Count}}"
sync_deleted: "Deleted: {{.Count}}"
sync_error: "Error: {{.Error}}"

# Common
common_cancel: "Cancel"
common_confirm: "Confirm"
common_error: "Error"
common_loading: "Loading..."
common_success: "Success"
common_warning: "Warning"
common_retry: "Retry"

# Error Messages
err_asset_not_found: "Asset not found"
err_invalid_id: "Invalid ID"
err_git_not_initialized: "No repository initialized. Run 'ep init <path>' first"
err_storage_not_configured: "Storage not configured"
```

---

## 5. Web UI 界面文本

### 5.1 需要翻译的界面

| 界面 | Key 前缀 | 示例 |
|------|----------|------|
| 侧边栏 | `sidebar_` | sidebar_assets, sidebar_settings |
| 资产列表 | `asset_list_` | asset_list_title, asset_list_empty |
| 资产编辑 | `asset_edit_` | asset_edit_title, asset_edit_save |
| Eval 面板 | `eval_` | eval_run, eval_compare, eval_report |
| 设置页 | `settings_` | settings_language, settings_theme |
| 通用按钮 | `btn_` | btn_create, btn_delete, btn_cancel |

### 5.2 Web 中文翻译 (zh-CN.json)

```json
{
  "sidebar_assets": "资产",
  "sidebar_settings": "设置",
  "asset_list_title": "Prompt 资产列表",
  "asset_list_empty": "暂无资产",
  "asset_list_create": "创建资产",
  "asset_edit_title": "编辑资产",
  "asset_edit_save": "保存",
  "asset_edit_cancel": "取消",
  "eval_run": "运行 Eval",
  "eval_compare": "对比",
  "eval_report": "报告",
  "settings_language": "语言",
  "settings_theme": "主题",
  "btn_create": "创建",
  "btn_delete": "删除",
  "btn_cancel": "取消",
  "btn_save": "保存",
  "common_loading": "加载中...",
  "common_error": "出错了",
  "common_retry": "重试"
}
```

---

## 6. 实现计划

### Phase 1: 基础设施 (基础框架)

1. 创建 `i18n/locales/` 目录
2. 创建 `internal/i18n/` 包
   - `i18n.go` - 核心 T() 函数
   - `loader.go` - YAML 加载器
   - `messages.go` - Key 常量
3. 创建基础语言包 zh-CN.yaml, en-US.yaml

### Phase 2: CLI 迁移

1. 选择一个命令作为试点（如 asset.go）
2. 将硬编码字符串替换为 `i18n.T("key")`
3. 添加参数支持
4. 验证编译和运行
5. 逐步迁移其他命令

### Phase 3: Web UI 集成

1. 安装 i18next 依赖
2. 创建 `web/src/i18n/` 配置
3. 创建语言 JSON 文件
4. 创建 `useTranslation` hook
5. 迁移一个组件作为试点
6. 逐步迁移其他组件

### Phase 4: 完善

1. 添加语言切换命令
2. Web 添加语言切换 UI
3. 补充遗漏的消息
4. 添加单元测试

---

## 7. 注意事项

### 7.1 Go 字符串格式化

```go
// 不推荐：拼接
fmt.Printf("资产 " + name + " 已创建")

// 推荐：模板参数
fmt.Printf(i18n.T("asset_created_with_name", "name", name))
```

### 7.2 复数支持（未来扩展）

```yaml
# 可定义复数形式
file_selected: "{{.Count}} file selected"
file_selected_plural: "{{.Count}} files selected"
```

### 7.3 占位符规范

Go 使用 Go 模板语法 `{{.FieldName}}`
Web JSON 使用 `{{fieldName}}` 或 `{fieldName}`

---

## 8. 相关资源

- [i18next](https://www.i18next.com/) - Web 国际化框架
- [go-i18n](https://github.com/nicksnyder/go-i18n) - Go 国际化（参考，未采用，自实现更轻量）
- Ant Design 国际化: [文档](https://ant.design/docs/react/i18n)
