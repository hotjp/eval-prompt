# [Demo] Tag Sense - 任务拆分规范

> **本文档是框架的示例业务任务拆分**，用于验证 vibe-go 脚手架的可用性。不属于框架本身。

本文档定义了 Tag Sense 示例项目的任务拆分策略。
> 所有任务通过 LRA 创建和管理。

---

## 一、模块树

Tag Sense 的业务模块不是扁平列表，而是一棵有层级关系和依赖方向的树。

```
Tag Sense
│
├── 共享内核 (Shared Kernel)
│   ├── 通用类型 (Status, TagSource, TagStatus, EngineType)
│   ├── 错误码定义
│   ├── 领域事件协议 (Event 结构 + Outbox 写入接口)
│   └── ULID 生成
│
├── 标签定义 (Tag Definition)
│   ├── TagType ─────────── 无父依赖，定义适用范围
│   ├── TagGroup ────────── 依赖 TagType (FK: tag_type_id)
│   └── TagValue ────────── 依赖 TagGroup (FK: tag_group_id)
│       └── 版本管理 ────── TagValue 的子关注点，非独立模块
│
├── 打标 (Tagging)
│   ├── 手动打标 ────────── 编排 TagValue → TagGroup → TagType (适用范围校验)
│   ├── 批量打标 ────────── 复用手动打标逻辑，增加部分失败容忍
│   ├── 查询统计 ────────── 只读路径，跨聚合展开标签链
│   └── 自动打标 ────────── 编排 Rule + Tagging.ApplyTag (跨模块编排)
│
├── 规则 (Rule)
│   ├── 规则管理 ────────── CRUD，创建时校验 entity_type 与 TagType.apply_subjects 兼容
│   └── 规则引擎插件 ────── Plugin 层，实现 service 层定义的 RuleMatcher 接口
│       ├── Builtin 匹配器
│       └── External 适配器
│
└── 基础设施 (Infrastructure)
    ├── PostgreSQL (连接 + 迁移 + 事务)
    ├── Redis (缓存 + 事件流)
    └── Outbox (写入器 + 轮询器 + Redis 发布)
```

### 模块间横向依赖

```
TagType ◄──── TagGroup ◄──── TagValue
                                ▲
          ┌─────────────────────┤
          │                     │
        Tagging ──────► TagValue (校验存在性 + 查适用范围)
          ▲
          │
        AutoTag ──► Rule (查询规则) + Tagging.ApplyTag (复用打标)
                        │
                        └──────► TagValue (校验 scope 兼容)
```

**编排归属规则**：

| 编排场景 | 归属模块 | 原因 |
|----------|----------|------|
| ApplyTag 校验 TagValue→TagGroup→TagType 链 | Tagging Service | 打标是触发方，标签定义是被动查询 |
| AutoTag 编排 Rule + ApplyTag | AutoTag Service | 自动打标是触发方，规则和打标是工具 |
| Rule 创建时校验 entity_type 兼容性 | Rule Service | 规则管理是触发方，TagValue 是被动校验 |
| 查询实体标签时展开 TagValue+TagGroup+TagType | Tagging Query | 查询是触发方，标签定义是展开目标 |

---

## 二、共享内核

共享内核是被多个业务模块依赖的基础定义，不属于任何单一模块。

### 2.1 共享内核清单

| 组件 | 包路径 | 依赖方 | 归属 Task |
|------|--------|--------|-----------|
| 枚举类型 (Status, TagSource, TagStatus, EngineType) | `domain/types.go` | 所有 L2-Domain | T03 |
| 错误码 (L1_xxx ~ L5_xxx) | `domain/errors.go` | 所有层 | T04 |
| 领域事件协议 (Event struct + OutboxWriter 接口) | `domain/events.go`, `service/interfaces.go` | L2-Domain, L1-Storage, L4-Service | T05 |
| ULID 生成 | `domain/ulid.go` | 所有 L2-Domain | T03 |
| 状态机框架 | `domain/statemachine.go` | 有状态机的实体 (TagType, TagGroup, TagValue, Tagging, Rule) | T05 |

### 2.2 共享内核的依赖规则

- 共享内核 **零业务模块依赖**，只依赖标准库和工具库（ulid）
- 业务模块 L2-Domain 可 import 共享内核
- 业务模块 L1-Storage / L4-Service 通过 L2-Domain 间接使用共享内核的类型
- 共享内核不定义任何数据库操作或外部调用

---

## 三、任务清单

### Phase 0: 脚手架 + 共享内核

#### T00 项目初始化

**目标**：
- `go mod init`
- 创建完整目录结构
- `.gitignore`, `Makefile`

**契约参考**：DESIGN.md §2.4 项目结构

**依赖接口**：无

**约定**：Go 1.23+, `github.com/{org}/tag-sense`

**验收标准**：
- `go build ./...` 无错误
- `make build` 正常执行

---

#### T01 配置系统

**目标**：`internal/config/config.go`，koanf 加载 YAML + 环境变量

**契约参考**：
```go
type Config struct {
    Server     ServerConfig
    PostgreSQL PostgreSQLConfig
    Redis      RedisConfig
    AuthCenter AuthCenterConfig
    OTel       OTelConfig
}
// 子配置详见 DESIGN.md §配置管理
```

**依赖接口**：T00

**验收标准**：YAML + 环境变量均可加载，required 字段缺失时明确报错

---

#### T02 Protobuf 定义 + 代码生成

**目标**：`api/tagsense/v1/tagsense.proto`, `buf.yaml`, `buf.gen.yaml`

**契约参考**：DESIGN.md §5.1-5.2 完整 Protobuf 定义

**依赖接口**：T00

**验收标准**：`buf generate` 成功，`go build ./gen/...` 通过

---

#### T03 共享内核: 通用类型 + ULID

**目标**：
- `internal/domain/types.go`：Status / TagSource / TagStatus / EngineType 枚举 + `String()` 方法
- `internal/domain/ulid.go`：`NewID() string`

**契约参考**：
```go
type Status    int32  // 0=Unspecified, 1=Active, 2=Disabled
type TagSource int32  // 0=Unspecified, 1=Manual, 2=AutoBuiltin, 3=AutoExternal
type TagStatus int32  // 0=Unspecified, 1=Active, 2=Revoked
type EngineType int32 // 0=Unspecified, 1=Builtin, 2=External
```
值来源：DESIGN.md §3.2-3.6 各实体的字段定义

**依赖接口**：T00（引入 oklog/ulid）

**本层产出**：
- `domain.Status`, `domain.TagSource`, `domain.TagStatus`, `domain.EngineType`
- `domain.NewID() string`

**约定**：零外部依赖（ulid 除外）

**验收标准**：`go test ./internal/domain/...` 通过，枚举值与 DESIGN.md 一致

---

#### T04 共享内核: 错误码

**目标**：`internal/domain/errors.go`

**契约参考**：DESIGN.md §8 完整错误码表

**依赖接口**：T03（domain 包已存在）

**本层产出**：
```go
// L1-Storage
var ErrDBConnection    = NewDomainError("L1_001", "数据库连接失败")
var ErrUniqueViolation = NewDomainError("L1_002", "唯一约束冲突")
var ErrFKViolation     = NewDomainError("L1_003", "外键约束违反")
var ErrOutboxPoll      = NewDomainError("L1_010", "Outbox 轮询失败")

// L2-Domain
var ErrInvalidTransition = NewDomainError("L2_201", "状态转换非法")
var ErrNameDuplicate     = NewDomainError("L2_202", "名称重复")
var ErrScopeViolation    = NewDomainError("L2_203", "业务规则校验失败")
var ErrVersionConflict   = NewDomainError("L2_204", "乐观锁版本冲突")

// L3-Authz
var ErrPermissionDenied = NewDomainError("L3_401", "权限拒绝")
var ErrAuthFailed       = NewDomainError("L3_402", "身份验证失败")

// L4-Service
var ErrInvalidInput       = NewDomainError("L4_601", "输入校验失败")
var ErrRuleMatchFailed    = NewDomainError("L4_602", "规则引擎匹配失败")
var ErrExternalTimeout    = NewDomainError("L4_603", "外部规则服务调用超时")

// L5-Gateway
var ErrBadRequest  = NewDomainError("L5_801", "请求参数解析失败")
var ErrInternal    = NewDomainError("L5_802", "内部服务错误")
```

**约定**：`NewDomainError(code, message)` 返回携带 layer 和 code 信息的 error

**验收标准**：错误码覆盖 DESIGN.md §8 全部条目，支持 `errors.Is` 匹配

---

#### T05 共享内核: 事件协议 + 状态机框架 + 接口契约

**目标**：
- `internal/domain/events.go`：8 种领域事件构造函数
- `internal/domain/statemachine.go`：通用状态机（states, transitions, guards, actions）
- `internal/service/interfaces.go`：所有层间接口定义

**契约参考**：
- 事件类型：DESIGN.md §7.1（8 种事件 + payload 字段）
- 状态机：CLAUDE.md §状态机（声明式定义，每次转换自动 increment_version）
- 接口：DESIGN.md §6.2（RuleMatcher, RuleEngineService）

**依赖接口**：T03, T04

**本层产出**：
```go
// domain/events.go
type DomainEvent struct {
    EventID        string
    AggregateType  string
    AggregateID    string
    EventType      string
    Payload        map[string]any
    OccurredAt     time.Time
    IdempotencyKey string
    Version        int
}

func NewTagTypeCreatedEvent(tt *TagType) DomainEvent { ... }
func NewTagTypeUpdatedEvent(tt *TagType, changedFields []string) DomainEvent { ... }
// ... 8 种事件

// domain/statemachine.go
type StateMachine struct { ... }
type Transition struct { From, To Status; Guard func() error; Action func() error }

// service/interfaces.go — 所有仓库接口 + 插件接口
type TagTypeRepo interface { ... }
type TagGroupRepo interface { ... }
type TagValueRepo interface { ... }
type TaggingRepo interface { ... }
type RuleRepo interface { ... }
type OutboxWriter interface { ... }
type CacheProvider interface { ... }
type RuleMatcher interface { ... }
type RuleEngineService interface { ... }
```

**约定**：
- DomainEvent 格式遵循 CLAUDE.md §领域事件（必须包含 event_id, aggregate_type, aggregate_id, event_type, payload, occurred_at, idempotency_key, version）
- 状态机每次转换自动 increment_version（乐观锁）
- `service/interfaces.go` 是唯一的跨模块接口契约，L4-Service 只依赖这些接口

**验收标准**：
- 8 种事件构造函数均可正确生成符合协议的 DomainEvent
- 状态机支持合法转换 + 拒绝非法转换 + 自动 version 递增
- 接口签名完整，覆盖所有 L1-Storage 和 Plugin 需要实现的操作

---

### Phase 1: 基础设施

#### T10 PostgreSQL 连接 + 迁移框架 + 事务管理

**目标**：
- `internal/storage/postgres/connect.go`：pgxpool 初始化
- `internal/storage/postgres/migrate.go`：golang-migrate 框架
- `internal/storage/postgres/tx.go`：TransactionManager 接口实现

**依赖接口**：
- 上游：T01（PostgreSQLConfig）, T04（错误码 L1_001）
- 本层产出：`postgres.Connect()`, `postgres.RunMigrations()`, `TransactionManager.InTx()`

**验收标准**：testcontainers 集成测试，连接 + 迁移 + 事务回滚均可工作

---

#### T11 数据库迁移脚本（全部表）

**目标**：`migrations/001~007` 共 7 组 up/down SQL

**契约参考**：DESIGN.md §4.1 全部 7 张表 DDL（直接复制）

**依赖接口**：T10

**验收标准**：up 顺序执行 + down 逆序执行均无错误，最终 schema 与 DESIGN.md 一致

---

#### T12 Redis 客户端 + 缓存层

**目标**：
- `internal/storage/redis/client.go`：go-redis 初始化
- `internal/storage/redis/cache.go`：Get/Set/Delete，支持 TTL 和 key 前缀

**契约参考**：DESIGN.md §4.2.1 Key 模式 + 过期时间 + 更新策略

**依赖接口**：
- 上游：T01（RedisConfig）
- 本层产出：`Cache.Get/Set/Delete`，实现 `service.CacheProvider` 接口

**约定**：JSON 序列化，单元测试用 miniredis

**验收标准**：miniredis 单元测试覆盖 miss / hit / set / delete

---

#### T13 Outbox 系统

**目标**：
- `internal/storage/postgres/outbox_writer.go`：事务内写入 outbox_events
- `internal/storage/postgres/outbox_poller.go`：后台轮询 PENDING 事件
- `internal/storage/redis/event_publisher.go`：发布到 Redis Stream `tag-sense:events`

**契约参考**：
- 表结构：DESIGN.md §4.1.7 outbox_events
- 事件流：DESIGN.md §7.2 事件流转（Domain → Storage → Outbox Poller → Redis Stream）
- 状态：0=PENDING, 1=PUBLISHED, 2=FAILED，失败重试

**依赖接口**：
- 上游：T10（TransactionManager）, T12（Redis client）, T05（DomainEvent 协议 + OutboxWriter 接口定义）
- 本层产出：实现 `service.OutboxWriter` 接口的具体类型

**验收标准**：
- 写入 + 轮询 + 发布到 Redis Stream 完整链路
- 失败重试 + 状态流转正确
- `go test ./internal/storage/... -run TestOutbox`

---

### Phase 2: 标签定义模块（TagType → TagGroup → TagValue）

> 依赖方向：TagType ◄── TagGroup ◄── TagValue
> 开发顺序：TagType → TagGroup → TagValue（沿依赖方向正向实现）

---

#### T20 TagType L2-Domain

**目标**：`internal/domain/tag_type.go`

**契约参考**：DESIGN.md §3.2 TagType struct + 业务规则

```go
type TagType struct {
    ID            string
    Name          string
    Description   string
    ApplySubjects []string  // ["artist", "influencer", "brand", "product"]
    ApplyChannels []string  // ["weibo", "douyin", "douban", "kuaishou", "weixin_video"]
    Status        Status    // ACTIVE | DISABLED
    Version       int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**依赖接口**：
- 上游：T03（Status, NewID）, T04（L2_201, L2_202）, T05（StateMachine, DomainEvent）
- 本层产出：`domain.TagType`, `NewTagType()`, `Disable()`, `UpdateMetadata()`, `CollectEvents()`

**约定**：
- Name 全局唯一，不允许重复
- 禁用后不可用于新打标（状态机约束）
- 不提供硬删，统一走禁用
- 使用 T05 的通用状态机框架

**验收标准**：
- `go test ./internal/domain/... -run TestTagType`
- 覆盖：NewTagType 正确初始化 / Disable 仅 ACTIVE→DISABLED / 重复禁用返回 L2_201 / 状态转换自动递增 version

---

#### T21 TagType L1-Storage

**目标**：
- Ent schema：`internal/storage/postgres/ent/schema/tag_type.go`
- Repository：`internal/storage/postgres/tag_type_repo.go`

**契约参考**：DESIGN.md §4.1.1 tag_type DDL（含 GIN 索引 on apply_subjects, apply_channels）

**依赖接口**：
- 上游：T10（Ent client / pgxpool）, T20（domain.TagType）, T04（L1_002, L1_003）
- 本层产出：实现 `service.TagTypeRepo` 接口

```go
// T05 定义的接口，本 Task 实现
type TagTypeRepo interface {
    Create(ctx context.Context, tt *domain.TagType) error
    GetByID(ctx context.Context, id string) (*domain.TagType, error)
    GetByName(ctx context.Context, name string) (*domain.TagType, error)
    List(ctx context.Context, filter TagTypeFilter) ([]*domain.TagType, string, int, error)
    Update(ctx context.Context, tt *domain.TagType) error
}
```

**约定**：Ent↔Domain 模型转换在 repo 内部完成，对外只暴露 domain 类型

**验收标准**：
- testcontainers 集成测试覆盖 CRUD + 分页 + 按 status/subject/channel 过滤
- 唯一约束冲突返回 L1_002
- 外键约束违反返回 L1_003

---

#### T22 TagType L4-Service

**目标**：`internal/service/tag_type.go`

**契约参考**：DESIGN.md §5（CreateTagType / GetTagType / ListTagTypes / UpdateTagType / DisableTagType RPC）

**依赖接口**：
- 上游：T21（TagTypeRepo）, T12（CacheProvider）, T13（OutboxWriter）, T05（事件构造函数）
- 本层产出：`TagTypeService` 及其 5 个方法

**跨模块编排**：无。TagType 不依赖其他业务模块。

**约定**：
- 写操作后同步刷新缓存（删除对应 key）
- 创建/更新/禁用后写入 Outbox 事件（TagTypeCreatedV1 / TagTypeUpdatedV1）
- Update 使用乐观锁（version 字段不匹配返回 L2_204）

**验收标准**：
- mock 测试覆盖 5 个 RPC 的正常路径 + 校验失败路径
- Create 校验 name 非空 + 不重复
- List 支持分页 + 过滤
- 缓存刷新和 Outbox 写入被正确调用

---

#### T30 TagGroup L2-Domain

**目标**：`internal/domain/tag_group.go`

**契约参考**：DESIGN.md §3.3 TagGroup struct + 业务规则

**依赖接口**：
- 上游：T03, T04, T05
- 本层产出：`domain.TagGroup`, `NewTagGroup()`, `Disable()`

**约定**：Name 在同一 TagTypeID 下唯一，适用范围继承自 TagType（不重复存储）

**验收标准**：同 T20 模式

---

#### T31 TagGroup L1-Storage

**目标**：Ent schema + Repository

**契约参考**：DESIGN.md §4.1.2 tag_group DDL（含 UNIQUE(tag_type_id, name)）

**依赖接口**：
- 上游：T10, T30（domain.TagGroup）, T04
- 本层产出：实现 `service.TagGroupRepo` 接口

**验收标准**：CRUD + 按 tag_type_id 过滤 + 分页，唯一约束 (tag_type_id, name)

---

#### T32 TagGroup L4-Service

**目标**：`internal/service/tag_group.go`

**契约参考**：DESIGN.md §5 TagGroup 相关 RPC

**依赖接口**：
- 上游：T31（TagGroupRepo）, T21（TagTypeRepo，校验 TagTypeID 存在且 ACTIVE）, T12, T13, T05

**跨模块编排**：
- 创建 TagGroup 时查询 TagType（调用 TagTypeRepo.GetByID），校验存在且 ACTIVE
- **编排归属**：TagGroup Service 是触发方，TagType 是被动校验。不调用 TagTypeService，直接用 TagTypeRepo

**验收标准**：
- 创建时校验 TagTypeID 存在且 ACTIVE
- CRUD + 缓存刷新 + Outbox 事件

---

#### T40 TagValue L2-Domain

**目标**：`internal/domain/tag_value.go`

**契约参考**：DESIGN.md §3.4 TagValue struct + 版本控制规则

**依赖接口**：
- 上游：T03, T04, T05
- 本层产出：
  - `domain.TagValue`
  - `NewTagValue()`
  - `UpdateMetadata()` — 名称/描述/排序变更，**不递增 version**
  - `SemanticChange(reason, operatorID string)` — 语义变更，**递增 version + 产出 ChangeLogEntry**
  - `Disable()`

**约定**：
- 区分两种更新路径（原地更新 vs 语义变更），这是本模块的核心复杂度
- 语义变更产出 ChangeLogEntry（old_version, new_version, change_type, snapshot）

**验收标准**：
- `go test ./internal/domain/... -run TestTagValue`
- 覆盖：NewTagValue / UpdateMetadata 不递增 version / SemanticChange 递增 version 并产出 changelog / Disable

---

#### T41 TagValue L1-Storage

**目标**：
- Ent schema：tag_value + tag_value_changelog 两张表
- Repository：TagValueRepo（CRUD + changelog 读写）

**契约参考**：DESIGN.md §4.1.3 tag_value DDL + §4.1.6 tag_value_changelog DDL

**依赖接口**：
- 上游：T10, T40, T04
- 本层产出：实现 `service.TagValueRepo` 接口 + `CreateChangelog()` / `ListChangelogs()`

**验收标准**：
- CRUD + 按 tag_group_id 过滤 + 分页
- Changelog 写入 + 按 tag_value_id 时间倒序查询
- 唯一约束 (tag_group_id, name)

---

#### T42 TagValue L4-Service

**目标**：`internal/service/tag_value.go`

**契约参考**：DESIGN.md §5 TagValue 相关 RPC + §6.3 标签值更新流程

**依赖接口**：
- 上游：T41（TagValueRepo）, T31（TagGroupRepo，校验 TagGroupID 存在且 ACTIVE）, T12, T13, T05

**跨模块编排**：
- 创建时校验 TagGroupID 存在且 ACTIVE（调 TagGroupRepo，不调 TagGroupService）
- Update 路径：根据 is_semantic_change 参数走 `UpdateMetadata()` 或 `SemanticChange()`
- SemanticChange 路径额外写入 changelog + 发布 TagValueUpdatedV1 事件

**验收标准**：
- 创建校验 TagGroupID 存在且 ACTIVE
- Update 正确区分两条路径
- SemanticChange 写 changelog + 发 Outbox 事件
- GetValueChangeLog 查询正确

---

### Phase 3: 打标模块

> Tagging 是核心业务模块，Service 层包含 4 条独立代码路径，拆为 4 个 Task。
> 所有 Service 子任务共享同一个 L2-Domain 和 L1-Storage。

---

#### T50 Tagging L2-Domain

**目标**：`internal/domain/tagging.go`

**契约参考**：DESIGN.md §3.5 Tagging struct + 业务规则

```go
type Tagging struct {
    ID         string
    EntityType string
    EntityID   string
    TagValueID string
    Source     TagSource    // MANUAL | AUTO_BUILTIN | AUTO_EXTERNAL
    OperatorID string
    Confidence float64     // 手动=1.0, 自动按规则
    Status     TagStatus   // ACTIVE | REVOKED
    Remark     string
    Version    int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

**依赖接口**：
- 上游：T03, T04, T05
- 本层产出：
  - `domain.Tagging`
  - `NewTagging()` — 默认 status=ACTIVE, 手动时 confidence=1.0
  - `Revoke()` — ACTIVE → REVOKED
  - `IsIdempotentWith(other *Tagging) bool` — 幂等判断（同 entity_type + entity_id + tag_value_id + status=ACTIVE）

**约定**：
- 幂等行为：同 (EntityType, EntityID, TagValueID) 只能一条 ACTIVE
- 撤销不物理删除，改状态
- 撤销后可再次打标同一标签（新 ACTIVE 记录）

**验收标准**：
- NewTagging 默认值正确
- Revoke 仅 ACTIVE→REVOKED
- IsIdempotentWith 正确判断幂等

---

#### T51 Tagging L1-Storage

**目标**：Ent schema + Repository

**契约参考**：DESIGN.md §4.1.4 tagging DDL（含部分唯一索引 WHERE status=1）

**依赖接口**：
- 上游：T10, T50, T04
- 本层产出：实现 `service.TaggingRepo` 接口

```go
type TaggingRepo interface {
    Create(ctx context.Context, t *domain.Tagging) error
    GetByID(ctx context.Context, id string) (*domain.Tagging, error)
    FindActive(ctx context.Context, entityType, entityID, tagValueID string) (*domain.Tagging, error) // 幂等查询
    ListByEntity(ctx context.Context, entityType, entityID string, status *TagStatus) ([]*domain.Tagging, error)
    Update(ctx context.Context, t *domain.Tagging) error
    BatchCreate(ctx context.Context, tags []*domain.Tagging) error
    Count(ctx context.Context, filter TaggingFilter) (int64, error)
    GroupByTagValue(ctx context.Context, filter TaggingFilter) ([]TagValueStat, error)
}
```

**验收标准**：
- 部分唯一索引正确工作（同 entity+tag 只一条 ACTIVE）
- 批量写入支持
- 按 entity 查询 + 按 tag_value 聚合统计

---

#### T52 Tagging L4-Service: ApplyTag

**目标**：`internal/service/tagging_apply.go`

**契约参考**：DESIGN.md §6.1 手动打标流程

**依赖接口**：
- 上游：T51（TaggingRepo）, T41（TagValueRepo）, T31（TagGroupRepo）, T21（TagTypeRepo）, T12（Cache）, T13（OutboxWriter）

**跨模块编排**（核心逻辑）：
```
ApplyTag(entity_type, entity_id, tag_value_id)
  1. TagValueRepo.GetByID(tag_value_id) → 校验存在且 ACTIVE
  2. TagGroupRepo.GetByID(tag_value.TagGroupID) → 获取 tag_type_id
  3. TagTypeRepo.GetByID(tag_group.TagTypeID) → 获取 apply_subjects
  4. 校验 entity_type ∈ apply_subjects → 否则返回 L2_203
  5. TaggingRepo.FindActive(entity_type, entity_id, tag_value_id) → 幂等判断
  6. 如果已存在 → 返回已有记录（幂等）
  7. domain.NewTagging() → TaggingRepo.Create() (事务内)
  8. OutboxWriter.Write(TaggingAppliedV1) (同事务)
  9. Cache.Delete(tag:entity:{entity_type}:{entity_id})
```

**约定**：
- 不调用其他模块的 Service，直接用 Repo 接口（避免循环依赖）
- 事务边界：步骤 7-8 必须在同一事务内

**验收标准**：
- mock 测试覆盖：正常打标 / TagValue 不存在 / TagValue 已禁用 / entity_type 不在 apply_subjects / 幂等返回已有记录
- 事务内 Create + Outbox Write 被一起调用

---

#### T53 Tagging L4-Service: RevokeTag

**目标**：`internal/service/tagging_revoke.go`

**依赖接口**：
- 上游：T51, T12, T13

**跨模块编排**：无跨模块查询，仅操作 Tagging 自身数据

**验收标准**：
- 校验 Tagging 记录存在且 ACTIVE
- Revoke + Outbox(TaggingRevokedV1) + 缓存刷新
- 重复撤销返回错误

---

#### T54 Tagging L4-Service: 查询与统计

**目标**：`internal/service/tagging_query.go`（GetEntityTags / BatchGetEntityTags / GetTaggingStats）

**依赖接口**：
- 上游：T51, T41, T31, T21, T12

**跨模块编排**：
- GetEntityTags：查 TaggingRepo.ListByEntity → 对每条 tagging 展开标签链（TagValueRepo → TagGroupRepo → TagTypeRepo）
- 优先走缓存，miss 时查 DB 并回填

**验收标准**：
- 返回完整标签链（Tagging + TagValue + TagGroup + TagType）
- BatchGet 支持多实体
- Stats 支持按 entity_type / tag_value_id / 时间范围过滤和聚合

---

#### T55 Tagging L4-Service: 批量打标

**目标**：`internal/service/tagging_batch.go`

**依赖接口**：
- 上游：T52（复用 ApplyTag 核心逻辑）

**验收标准**：
- BatchApplyTags（单实体多标签）：部分失败不影响其他，返回每条结果
- BatchApplyTagToEntities（多实体单标签）：同上
- 返回 success_count / failure_count

---

### Phase 4: 规则模块

---

#### T60 Rule L2-Domain

**目标**：`internal/domain/rule.go`

**契约参考**：DESIGN.md §3.6 Rule struct + MatchConfig + 业务规则

**依赖接口**：
- 上游：T03, T04, T05
- 本层产出：
  - `domain.Rule`
  - `domain.MatchConfig`
  - `NewRule()`
  - `ValidateScope(applySubjects []string) error` — 校验 entity_type ∈ applySubjects
  - `Disable()`

**约定**：
- ValidateScope 是跨模块校验的 domain 表达：Rule.entity_type 必须与 TagType.apply_subjects 兼容
- 不在 Domain 层直接依赖 TagType，由调用方传入 applySubjects

**验收标准**：
- NewRule 校验必填字段
- ValidateScope 正确判断 entity_type 兼容性
- 禁用状态机

---

#### T61 Rule L1-Storage

**目标**：Ent schema + Repository

**契约参考**：DESIGN.md §4.1.5 rule DDL

**依赖接口**：T10, T60, T04

**验收标准**：
- CRUD + 按 (engine_type, status, entity_type) 过滤
- name 唯一约束
- JSONB match_config 存储

---

#### T62 Rule L4-Service: CRUD

**目标**：`internal/service/rule.go`

**依赖接口**：
- 上游：T61, T41, T31, T21, T12, T13

**跨模块编排**：
- 创建/更新时：查 TagValueRepo → TagGroupRepo → TagTypeRepo 获取 apply_subjects → 调用 `rule.ValidateScope(applySubjects)`
- **编排归属**：Rule Service 是触发方，标签定义模块是被动校验

**验收标准**：
- 创建时校验 TagValueID 存在且 ACTIVE
- 创建时校验 entity_type 与 TagType.apply_subjects 兼容（不兼容返回 L2_203）
- 乐观锁更新
- CRUD + 缓存 + Outbox

---

### Phase 5: 规则引擎插件

---

#### T70 插件: Builtin 关键词匹配器

**目标**：`plugins/ruleengine/builtin/keyword.go`

**契约参考**：DESIGN.md §6.2 BUILTIN 引擎 + §3.6 MatchConfig（Keywords, MatchMode: ANY|ALL）

**依赖接口**：
- 上游：T05（`service.RuleMatcher` 接口定义）
- 本层产出：`KeywordMatcher` struct，实现 `RuleMatcher` 接口

**约定**：
- 纯单元测试，无外部依赖
- 大小写不敏感

**验收标准**：
- ANY 模式：任一关键词命中即匹配
- ALL 模式：所有关键词命中才匹配
- 空内容 / 空关键词列表正确处理

---

#### T71 插件: External 规则服务适配器

**目标**：`plugins/ruleengine/external/client.go`

**契约参考**：DESIGN.md §6.2 EXTERNAL 引擎 + §3.6 MatchConfig（Endpoint, Method, Headers, Timeout）

**依赖接口**：T05（`service.RuleMatcher` 接口）

**约定**：超时返回 L4_603，httptest mock 测试

**验收标准**：
- HTTP 调用外部服务，支持自定义 Headers/Method/Timeout
- 超时和 5xx 正确处理
- httptest mock 覆盖正常/超时/错误

---

### Phase 6: 自动打标（跨模块编排）

---

#### T80 AutoTag L4-Service

**目标**：`internal/service/auto_tag.go`

**契约参考**：DESIGN.md §6.2 自动打标流程（双通路）

**跨模块编排**（核心逻辑）：
```
AutoTag(entity_type, entity_id, content, context)
  1. RuleRepo.List(filter: entity_type, status=ACTIVE) → 获取所有匹配规则
  2. errgroup 并发执行规则匹配：
     - BUILTIN 规则 → KeywordMatcher.Match(content, rule.MatchConfig)
     - EXTERNAL 规则 → ExternalClient.Match(content, rule.MatchConfig)
  3. 收集匹配结果，按 Priority 排序取最高
  4. 调用 T52.ApplyTag() 完成打标（source=AUTO_BUILTIN 或 AUTO_EXTERNAL）
  5. 返回匹配结果
```

**依赖接口**：
- 上游：T61（RuleRepo）, T52（ApplyTag，复用打标逻辑）, T70/T71（RuleMatcher 插件实例）

**约定**：
- 错误码：L4_602（匹配失败）, L4_603（外部超时）
- errgroup 限制并发数
- 单条规则匹配失败不阻塞其他规则

**验收标准**：
- mock 测试覆盖：无匹配 / 单匹配 / 多匹配按优先级 / 外部超时不阻塞 / 最终调用 ApplyTag

---

### Phase 7: Gateway + Authz

---

#### T90 Auth-center 客户端 + 拦截器

**目标**：
- `internal/authz/client.go`：HTTP 客户端，调用 auth-center CheckPermission
- `internal/authz/interceptor.go`：Connect 拦截器（JWT → user_id → 权限校验）

**契约参考**：DESIGN.md §9（对接方式 + 降级策略 + 资源-动作映射）

**依赖接口**：
- 上游：T01（AuthCenterConfig）, T12（权限结果缓存，key: `auth:perm:{user_id}:{resource}:{action}`）
- 本层产出：`AuthInterceptor` connect interceptor

**约定**：
- 错误码：L3_401 / L3_402
- 降级：auth-center 不可用 → 查缓存 → miss → 503 Fail Closed
- 熔断：30s 内 10 次失败

**验收标准**：
- JWT 解析正确提取 user_id
- CheckPermission 正确调用 auth-center
- 降级和熔断逻辑正确
- httptest mock 测试

---

#### T91 Gateway: Handler + 路由 + 中间件

**目标**：
- `internal/gateway/handler.go`：Connect handler 初始化
- `internal/gateway/routes.go`：注册所有 RPC 路由
- `internal/gateway/middleware.go`：Recovery / Metrics / CORS / Logging / RequestID

**契约参考**：DESIGN.md §5.3 HTTP 路由映射

**依赖接口**：
- 上游：T02（生成的 Connect handler）, T90（AuthInterceptor）, 所有 Service（T22, T32, T42, T52-T55, T62, T80）

**验收标准**：
- 所有 RPC 注册到 Connect handler
- 路由路径与 §5.3 一致
- panic 不导致 server 崩溃
- Prometheus 指标采集

---

#### T92 Gateway: Health Endpoints

**目标**：`internal/gateway/health.go`（/healthz + /readyz）

**依赖接口**：T10, T12

**验收标准**：/healthz 始终 200，/readyz 检查 PostgreSQL + Redis 连通性

---

#### T93 main.go: 依赖注入组装

**目标**：`cmd/server/main.go`

**依赖接口**：所有 T01-T92

**验收标准**：
- 按依赖图顺序组装所有组件
- 优雅关闭（signal handling）
- `go build ./cmd/server/` 成功
- 启动后 /healthz 返回 200

---

## 四、依赖关系总图

```
Phase 0 (可并行: T01, T02, T03)
  T00 ──► T01, T02
  T00 ──► T03 ──► T04 ──► T05

Phase 1 (基础设施)
  T01 ──► T10, T12
  T10 ──► T11, T13
  T05 + T12 + T10 ──► T13

Phase 2 (标签定义，沿依赖方向串行)
  T05 ──► T20 ──► T21 ──► T22
              T20 ──► T30 ──► T31 ──► T32
                          T30 ──► T40 ──► T41 ──► T42

Phase 3 (打标)
  T05 ──► T50 ──► T51
  T51 + T41 + T31 + T21 ──► T52
  T51 ──► T53
  T51 + T41 + T31 + T21 ──► T54
  T52 ──► T55

Phase 4 (规则)
  T05 ──► T60 ──► T61 ──► T62

Phase 5 (插件，可并行)
  T05 ──► T70
  T05 ──► T71

Phase 6 (自动打标，跨模块编排)
  T61 + T52 + T70 + T71 ──► T80

Phase 7 (Gateway + Authz)
  T01 + T12 ──► T90
  T02 + T90 + 所有Service ──► T91
  T10 + T12 ──► T92
  所有 ──► T93
```

### 可并行任务组

| 阶段 | 可并行的任务 | 前提条件 |
|------|-------------|----------|
| Phase 0 | T01 ∥ T02 ∥ T03 | T00 完成 |
| Phase 1 | T10 ∥ T12 | T01 完成 |
| Phase 1 | T11 ∥ T13 | T10 完成 |
| Phase 2 | T20 ∥ T30 ∥ T40 ∥ T50 ∥ T60 | T05 完成（Domain 层互不依赖） |
| Phase 2 | T21 ∥ T31 ∥ T41 ∥ T51 ∥ T61 | 各自 Domain 完成 |
| Phase 2 | T22 ∥ T32 ∥ T42 ∥ T53 ∥ T62 | 各自 Storage 完成 |
| Phase 5 | T70 ∥ T71 | T05 完成 |
