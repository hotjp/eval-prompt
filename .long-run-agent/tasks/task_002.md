# task_002

## ⚠️ 重要提示（Agent 必读）

**当前位置**: `.long-run-agent/tasks/task_002.md`（任务描述文件）

**工作目录**: 项目根目录（`.long-run-agent` 的同级目录）

**产出物**: 请在项目根目录或适当子目录创建交付物

**这是配置文件**，不是最终产出！

## 描述

F02 配置系统 — koanf 初始化、配置结构体、YAML 加载

## 需求 (requirements)

- 使用 koanf 加载配置（YAML + 环境变量）
- 配置结构体（参考 docs/DESIGN.md Section 九）：
  - ServerConfig: port, host, metrics_port, pprof_port, read_timeout, write_timeout
  - DatabaseConfig: dsn, max_open, max_idle, max_lifetime
  - SandboxConfig: allowed_commands, forbidden_patterns, max_execution_time, max_file_size, max_file_count
  - PromptAssetsConfig: repo_path, assets_dir, evals_dir, traces_dir, eval_threshold
  - PluginsConfig: llm, search
- 默认 config.yaml
- APP_ 前缀环境变量覆盖

## 验收标准 (acceptance)

- [ ] koanf 配置加载正常
- [ ] 环境变量可覆盖 YAML 配置
- [ ] 配置结构体完整

## 交付物 (deliverables)

- `internal/config/config.go`
- `config.yaml`

## 验证证据（完成前必填）

- [ ] **实现证明**: 使用 koanf 多源加载
- [ ] **测试验证**: 环境变量覆盖测试
- [ ] **影响范围**: 所有层依赖此配置

### 测试步骤
1. 默认配置加载成功
2. APP_SERVER_PORT=9090 覆盖 port

### 验证结果
