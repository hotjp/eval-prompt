# Future Ideas (Long-shots)

记录尚未确定落地的想法，不承诺实现，仅供参考。

---

## Per-Project Repo Configuration

### 背景

当前 `ep repo switch` 是全局的，每个用户机器只有一个"当前仓库"。但实际可能有以下场景需要 per-project 配置：

- **外包团队**：每个客户项目对接不同资产仓库
- **Monorepo**：多个子项目各自关联不同资产
- **多租户 SaaS**：每个 workspace 独立资产库

### 设想

在项目目录下放一个 `.epconfig` 文件：

```yaml
# /project-root/.epconfig
repo: /Users/kingj/client-a-assets
assets_dir: prompts
evals_dir: .evals
```

### Auto-discovery 流程

```
cd /project-x
ep assets list
  → 查找 .epconfig (当前目录)
    → 查找 .epconfig (父目录)
      → ... → 找到或用全局 ~/.ep/lock.json
```

### 实现路径

1. `lock.json` 加一个字段 `project_config: true`
2. 启动时从当前目录向上查找 `.epconfig`
3. 找到则用 `.epconfig` 的值覆盖 lock.json 的 current
4. 作为 pip/npm 包分发 `ep` CLI

### 前提

需要先有真实用户需求驱动，当前只是 long-shot 想法。

---

## 其他 Long-shot

（待补充）
