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
- Go 和 Web 各自用自己的加载器读取同一套语言定义文件（JSON 格式）

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
```go
// 使用 pongo2 模板语法 {{var}}
asset_delete_success: "资产已删除: {{id}}"
eval_run_progress: "正在运行 Eval ({{current}}/{{total}})"
```

---

## 3. 架构设计

### 3.1 文件结构

```
eval-prompt/
├── i18n/
│   └── locales/
│       ├── zh-CN.json       # 中文语言包（Go embed + Web import 共用）
│       └── en-US.json       # 英文语言包（Go embed + Web import 共用）
│
├── internal/
│   └── i18n/
│       └── i18n.go          # 核心: T() 函数、语言切换，纯标准库无第三方依赖
│
└── web/
    └── src/
        ├── i18n/
        │   └── index.ts     # i18next 配置，直接 import 同目录 JSON
        └── hooks/
            └── useTranslation.ts
```

### 3.2 依赖策略

| 端 | 加载方式 | 第三方依赖 |
|----|---------|-----------|
| Go CLI | `//go:embed` + `encoding/json` + `pongo2` | 项目已有 pongo2（模板引擎） |
| Web | i18next import 同个 JSON 文件 | i18next + react-i18next |

- **统一使用 JSON 格式**：避免引入 YAML 解析器（如 go-yaml）增加二进制体积
- **复用项目已有依赖**：Go 端使用项目现有的 pongo2 做模板渲染，无需引入新依赖
- **语言文件一份，多端共用**：避免维护两份不同格式的翻译

### 3.3 Go i18n 实现

使用项目已有的 **pongo2** 模板库，保持技术栈统一。

```go
// internal/i18n/i18n.go

package i18n

import (
    "embed"
    "encoding/json"
    "sync"

    "github.com/flosch/pongo2/v6"
)

//go:embed locales/*.json
var fs embed.FS

var (
    locales  map[string]map[string]string  // lang -> key -> message
    current  string = "zh-CN"
    initOnce sync.Once
)

// Init 线程安全，只执行一次。语言包缺失不报错，运行时 SetLang 会验证有效性。
func Init() error {
    var err error
    initOnce.Do(func() {
        locales = make(map[string]map[string]string)
        for _, lang := range []string{"zh-CN", "en-US"} {
            data, e := fs.ReadFile("locales/" + lang + ".json")
            if e != nil {
                continue
            }
            json.Unmarshal(data, &locales[lang])
        }
        if _, ok := locales[current]; !ok {
            current = "en-US"
        }
    })
    return err
}

// SetLang 切换当前语言，lang 无效时静默忽略
func SetLang(lang string) {
    if _, ok := locales[lang]; ok {
        current = lang
    }
}

// T 返回本地化字符串，支持 pongo2 模板语法 {{var}}。
// 翻译文件中可调整占位符顺序以适应不同语种的语法。
//
// 使用方式：
//
//	i18n.T("asset_create_success", pongo2.Context{"id": id})
//	i18n.T("eval_compare_title", pongo2.Context{"asset_id": id, "v1": v1, "v2": v2})
//
// 无对应 key 时返回 key 本身；模板解析失败时返回原文。
func T(key string, args pongo2.Context) string {
    msg, ok := locales[current][key]
    if !ok {
        return key
    }
    if len(args) == 0 {
        return msg
    }
    tpl, err := pongo2.FromString(msg)
    if err != nil {
        return msg
    }
    return tpl.Execute(args)
}
```

**语言文件示例：**
```json
{
  "asset_create_success": "资产已创建: {{id}}",
  "eval_compare_title": "{{asset_id}}: {{v1}} vs {{v2}}"
}
```

**使用方式：**
```go
// Before
fmt.Printf("资产已创建: %s\n", id)

// After
fmt.Println(i18n.T("asset_create_success", pongo2.Context{"id": id}))
```

### 3.4 Web i18n 实现

使用 `i18next` + `react-i18next`，直接 import 同目录的语言 JSON 文件：

```typescript
// web/src/i18n/index.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from '../../../i18n/locales/zh-CN.json';
import enUS from '../../../i18n/locales/en-US.json';

i18n
  .use(initReactI18next)
  .init({
    lng: localStorage.getItem('lang') || 'zh-CN',
    resources: {
      'zh-CN': { translation: zhCN },
      'en-US': { translation: enUS },
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

### 3.5 语言检测顺序

| 优先级 | CLI | Web |
|--------|-----|-----|
| 1 | 环境变量 `EP_LANG` | `localStorage.lang` |
| 2 | `LANG` 环境变量 | 浏览器 `navigator.language` |
| 3 | 默认 `zh-CN` | 默认 `zh-CN` |

---

## 4. 语言包设计

统一使用 JSON 格式，Go 和 Web 共用同一份文件，通过 `//go:embed` 和 `import` 各自加载。

### 4.1 zh-CN.json 结构

```json
{
  "asset_create_success": "资产已创建: {{id}}",
  "asset_archive_success": "资产已归档: {{id}}",
  "asset_restore_success": "资产已恢复: {{id}}",
  "asset_delete_success": "资产已删除: {{id}}",
  "asset_not_found": "资产不存在: {{id}}",
  "asset_state_conflict": "请先 archive",

  "eval_run_started": "Eval 运行已启动: {{id}}",
  "eval_run_status": "状态: {{status}}",
  "eval_cancel_success": "Eval 已取消: {{id}}",
  "eval_compare_title": "{{asset_id}}: {{v1}} vs {{v2}}",
  "eval_score_delta": "得分差: {{delta}}",

  "serve_starting": "启动服务: http://{{addr}}",
  "serve_started": "服务已启动",
  "serve_api_endpoint": "API 端点: http://{{addr}}/mcp/v1",
  "serve_sse_endpoint": "SSE 端点: http://{{addr}}/mcp/v1/sse",
  "serve_opening_browser": "正在打开浏览器...",

  "init_title": "初始化 prompt assets 仓库: {{path}}",
  "init_git_complete": "Git 仓库初始化完成",
  "init_lock_added": "仓库已添加到锁文件",
  "init_complete": "初始化完成",
  "init_serve_hint": "运行 'ep serve' 启动服务",

  "sync_reconcile_done": "对账完成",
  "sync_added": "新增: {{count}}",
  "sync_updated": "更新: {{count}}",
  "sync_deleted": "删除: {{count}}",
  "sync_error": "错误: {{error}}",

  "common_cancel": "取消",
  "common_confirm": "确认",
  "common_error": "错误",
  "common_loading": "加载中",
  "common_success": "成功",
  "common_warning": "警告",
  "common_retry": "重试",

  "err_asset_not_found": "资产不存在",
  "err_invalid_id": "无效的 ID",
  "err_git_not_initialized": "尚未初始化任何仓库，请先运行 'ep init <path>'",
  "err_storage_not_configured": "存储未配置"
}
```

### 4.2 en-US.json 结构

```json
{
  "asset_create_success": "Asset created: {{id}}",
  "asset_archive_success": "Asset archived: {{id}}",
  "asset_restore_success": "Asset restored: {{id}}",
  "asset_delete_success": "Asset deleted: {{id}}",
  "asset_not_found": "Asset not found: {{id}}",
  "asset_state_conflict": "Please archive first",

  "eval_run_started": "Eval run started: {{id}}",
  "eval_run_status": "Status: {{status}}",
  "eval_cancel_success": "Eval cancelled: {{id}}",
  "eval_compare_title": "{{asset_id}}: {{v1}} vs {{v2}}",
  "eval_score_delta": "Score delta: {{delta}}",

  "serve_starting": "Starting server: http://{{addr}}",
  "serve_started": "Server started",
  "serve_api_endpoint": "API endpoint: http://{{addr}}/mcp/v1",
  "serve_sse_endpoint": "SSE endpoint: http://{{addr}}/mcp/v1/sse",
  "serve_opening_browser": "Opening browser...",

  "init_title": "Initializing prompt assets repository: {{path}}",
  "init_git_complete": "Git repository initialized",
  "init_lock_added": "Repository added to lock file",
  "init_complete": "Initialization complete",
  "init_serve_hint": "Run 'ep serve' to start the server",

  "sync_reconcile_done": "Reconcile complete",
  "sync_added": "Added: {{count}}",
  "sync_updated": "Updated: {{count}}",
  "sync_deleted": "Deleted: {{count}}",
  "sync_error": "Error: {{error}}",

  "common_cancel": "Cancel",
  "common_confirm": "Confirm",
  "common_error": "Error",
  "common_loading": "Loading...",
  "common_success": "Success",
  "common_warning": "Warning",
  "common_retry": "Retry",

  "err_asset_not_found": "Asset not found",
  "err_invalid_id": "Invalid ID",
  "err_git_not_initialized": "No repository initialized. Run 'ep init <path>' first",
  "err_storage_not_configured": "Storage not configured"
}
```

### 4.3 消息参数格式

- **Go 端**：使用 pongo2 模板语法 `{{var}}`，与项目现有模板风格统一
- **Web 端**：i18next 默认使用 `{{name}}` 占位符

两种格式语法一致（都是 `{{var}}`），可以保持语言文件同一份 JSON 被两方使用。

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

> **注意**：Web UI 翻译已合并到 `i18n/locales/` 下的统一语言文件中，与 CLI 共用同一份 JSON。

---

## 6. 实现计划

### Phase 1: 基础设施

1. 创建 `i18n/locales/` 目录，放入 `zh-CN.json` 和 `en-US.json`
2. 创建 `internal/i18n/i18n.go`（纯标准库，`embed` + `encoding/json`）
3. 在 `serve.go` 启动时调用 `i18n.Init()`

### Phase 2: CLI 迁移

1. 选择一个命令作为试点（如 `asset.go`）
2. 将硬编码字符串替换为 `i18n.T("key", args...)`
3. 验证编译和运行
4. 逐步迁移其他命令

### Phase 3: Web UI 集成

1. 安装 i18next + react-i18next
2. 创建 `web/src/i18n/index.ts` 配置
3. 迁移一个组件作为试点
4. 逐步迁移其他组件

### Phase 4: 完善

1. 添加语言切换命令（`ep serve --lang=en-US`）
2. Web 添加语言切换 UI
3. 补充遗漏的消息

---

## 7. 注意事项

### 7.1 Go 字符串格式化

```go
// 不推荐：拼接
fmt.Printf("资产 " + name + " 已创建")

// 推荐：参数替换
fmt.Printf(i18n.T("asset_created_with_name", name))
```

### 7.2 复数支持（未来扩展）

```json
{
  "file_selected": "{{count}} file selected",
  "file_selected_plural": "{{count}} files selected"
}
```

i18next 支持 `plural` 特性，Go 端如需复数可维护两份 key 或自行处理。

### 7.3 体积控制

- Go 端**严禁引入** `go-yaml`、`go-i18n` 等额外第三方库
- 使用项目已有的 `pongo2` 模板库（已在 go.mod 中）
- 语言文件使用 JSON 格式，Go 用标准库 `encoding/json` 解析
- 两种语言全部 embed，总增量预计 < 30KB（gzip）

---

## 8. 相关资源

- [i18next](https://www.i18next.com/) - Web 国际化框架
- [react-i18next](https://react.i18next.com/) - React 绑定
- [pongo2](https://github.com/flosch/pongo2) - Go 模板引擎（项目已有）
- Ant Design 国际化: [文档](https://ant.design/docs/react/i18n)
