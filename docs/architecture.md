# vibe-go - 完整架构规范

## 目标
单次生成可编译、可部署、符合企业工程规范的 Go 生产系统。

---

## 一、技术栈定义

### 核心框架
| 类别 | 选型 | 说明 |
|---|---|---|
| API 协议 | `connect-go` | 同时支持 gRPC（二进制）和 HTTP+JSON，客户端自选 |
| Protobuf | `buf` + `protoc-gen-go` | API 定义、代码生成、lint、breaking change 检测 |
| ORM | `ent` (底层 `pgx/v5`) | 类型安全查询、自动迁移、代码生成、图谱关系 |
| ID 生成 | `oklog/ulid` | 全局唯一、按时间排序、26 字符 Base32 |
| API 文档 | `buf` + `connectrpc/reflection` | Proto 天然是契约，buf 生成文档，reflection 支持运行时探索 |

### 存储
| 类别 | 选型 | 说明 |
|---|---|---|
| PostgreSQL | `pgx/v5` | 主数据库，ent 底层驱动，支持 `pgx` 原生特性 |
| Redis | `go-redis/v9` | 缓存、事件流(Stream)、分布式锁、Rate Limiter |

### 可观测性
| 类别 | 选型 | 说明 |
|---|---|---|
| 日志 | `log/slog` (标准库) | 结构化 JSON 日志，Handler 接口可扩展，零外部依赖 |
| 链路追踪 | `opentelemetry-go` | OTLP exporter，概率/自适应采样 |
| Metrics | `opentelemetry-go` + Prometheus | 请求延迟 P50/P95/P99、错误率、业务计数器 |

### 基础设施
| 类别 | 选型 | 说明 |
|---|---|---|
| 配置管理 | `koanf` | 显式依赖注入（非全局单例），YAML + env + 命令行 |
| 数据库迁移 | `golang-migrate` | 版本化 SQL 迁移，CLI + 库双模式 |
| HTTP | `net/http` (标准库) | connect-go 底层，无需额外框架 |

### 测试
| 类别 | 选型 | 说明 |
|---|---|---|
| 断言 | `testify` | assert/require/suite |
| Mock | `gomock` | `go generate` 生成接口 mock |
| 集成测试 | `testcontainers-go` | PostgreSQL / Redis 独立容器 |
| Redis Mock | `miniredis` | 单元测试替代真实 Redis |

### 不使用的库（及原因）
| 排除项 | 原因 |
|---|---|
| `zap` / `zerolog` | `slog` 是 Go 1.21+ 标准库，性能足够，零外部依赖 |
| `viper` | 全局单例与依赖注入冲突，koanf 更显式 |
| `gin` / `echo` / `fiber` | connect-go 基于 `net/http`，无需额外 HTTP 框架 |
| `gorm` | ent 提供代码生成和类型安全，更适合企业项目 |

---

## 二、架构规范：5层核心 + N插件（接口倒置）

### 依赖铁律
```
L5-Gateway → L3-Authz → L4-Service → L2-Domain → L1-Storage
```

### 核心设计原则
核心层定义接口，插件层实现接口，通过依赖注入连接（**禁止核心层 import 插件层具体实现**）。

---

## 三、分层职责定义

### L5-Gateway：入口网关
- **职责**：TLS终止、协议适配、全局中间件（Recover/Metrics/CORS）、请求路由
- **向下调用**：调用L3-Authz（权限验证通过后才进入业务流）
- **技术栈**：`connect-go`(gRPC/HTTP 双模)、CORS、基础JWT解析（仅解密不验证）
- **中间件注册顺序**：Recover → RequestID → Metrics → Logging → CORS → Auth → Routing

### L3-Authz：权限决策（前置关卡）
- **职责**：请求准入控制、权限检查、Rate Limiting、身份验证
- **向上接收**：接收L5解析的JWT（user_id, claims）
- **向下调用**：验证通过后调用L4-Service，携带user_context
- **强制规则**：所有RPC必须在L3检查完成后才能到达L4（Fail Fast）
- **降级策略**：权限服务不可用时，缓存最近结果（Redis TTL 5min），过期返回 503

### L4-Service：业务编排
- **职责**：输入校验（业务规则）、事务边界、工作流触发、领域协调、插件调度
- **向上依赖**：依赖L3传递的user_context，不重复验证权限
- **向下调用**：调用L2-Domain纯业务逻辑
- **依赖接口（插件实现）**：所有外部系统集成通过 interface 定义在 `interfaces.go`
- **禁止项**：直接 import plugins 包，必须通过 interface + 依赖注入

### L2-Domain：领域核心
- **职责**：领域实体、状态机（声明式实现）、领域事件收集（Outbox）、业务不变量
- **技术要求**：纯Go struct，零外部依赖（除标准库），禁止 import 任何第三方包
- **事件发布**：事务内收集到Outbox（通过L1接口），禁止直接调用外部

### L1-Storage：数据持久
- **职责**：Ent实现、事务管理、Outbox表轮询、事件转发Redis
- **Outbox模式实现**：
  1. **表结构**：outbox_events（id, aggregate_type, aggregate_id, event_type, payload, occurred_at, idempotency_key, status, retry_count, created_at）
  2. **事务边界**：L2在业务事务内写入Outbox（同库同事务）
  3. **转发器**：后台goroutine轮询pending事件，发送到Redis Stream
  4. **并发安全**：多实例部署时使用 `SELECT ... FOR UPDATE SKIP LOCKED` 防止重复消费
  5. **ACK机制**：消费者处理成功后确认，失败保留pending指数退避重试

---

## 四、插件层规范（接口倒置实现）

### 接口定义（在 L4-Service）
```go
// internal/service/interfaces.go
// 每个插件接口定义在此文件，由 plugins/ 下的具体包实现
```

### 插件示例
```go
// 定义在 L4-Service/interfaces.go（核心层定义）
type SearchIndexer interface {
    Index(ctx context.Context, id string, data any) error
    Remove(ctx context.Context, id string) error
    BatchIndex(ctx context.Context, items []IndexItem) error
}

// 实现在 plugins/search/indexer.go（插件层实现）
type MeiliIndexer struct {
    client *meilisearch.Client
}
```

### 插件挂载方式（依赖注入）
```go
// cmd/server/main.go
func main() {
    // 加载配置
    cfg := config.MustLoad("config.yaml")

    // 初始化核心层
    storage := storage.New(cfg.Database, cfg.Redis)
    domainSvc := domain.New(storage)
    authzSvc := authz.New(cfg.Authz)

    // 初始化插件（可选，enabled=false 则用 noop 空实现）
    var indexer service.SearchIndexer = noop.NoopIndexer{}
    if cfg.Plugins.Search.Enabled {
        indexer = search.NewMeiliIndexer(cfg.Plugins.Search.Meili)
    }

    // 注入到 Service 层
    svc := service.New(domainSvc, authzSvc, indexer)  // 接口注入
    gateway.New(svc, authzSvc, cfg.Server).Start()
}
```

---

## 五、配置管理（koanf）

### 配置结构体
```go
// internal/config/config.go
type Config struct {
    Server   ServerConfig   `koanf:"server"`
    Database DatabaseConfig `koanf:"database"`
    Redis    RedisConfig    `koanf:"redis"`
    Authz    AuthzConfig    `koanf:"authz"`
    Telemetry TelemetryConfig `koanf:"telemetry"`
    Plugins  PluginsConfig  `koanf:"plugins"`
}

type ServerConfig struct {
    Port         int           `koanf:"port"`
    MetricsPort  int           `koanf:"metrics_port"`
    PprofPort    int           `koanf:"pprof_port"`
    ReadTimeout  time.Duration `koanf:"read_timeout"`
    WriteTimeout time.Duration `koanf:"write_timeout"`
}

type DatabaseConfig struct {
    DSN          string        `koanf:"dsn"`
    MaxOpen      int           `koanf:"max_open"`
    MaxIdle      int           `koanf:"max_idle"`
    MaxLifetime  time.Duration `koanf:"max_lifetime"`
}

type RedisConfig struct {
    Addr     string `koanf:"addr"`
    Password string `koanf:"password"`
    DB       int    `koanf:"db"`
}

type AuthzConfig struct {
    Endpoint string        `koanf:"endpoint"`
    Timeout  time.Duration `koanf:"timeout"`
    CacheTTL time.Duration `koanf:"cache_ttl"`
}

type TelemetryConfig struct {
    ServiceName string  `koanf:"service_name"`
    Endpoint    string  `koanf:"endpoint"`
    SampleRate  float64 `koanf:"sample_rate"`
}

type PluginsConfig struct {
    Search   SearchPluginConfig   `koanf:"search"`
    // 按需扩展...
}
```

### 配置加载
```go
// internal/config/load.go
func MustLoad(path string) *Config {
    k := koanf.New(".")
    // 1. 默认配置
    k.Load(structs.Provider(DefaultConfig(), "koanf"), nil)
    // 2. YAML 文件
    k.Load(file.Provider(path), yaml.Parser())
    // 3. 环境变量覆盖（APP_ 前缀，下划线分隔层级）
    k.Load(env.Provider("APP_", ".", func(s string) string {
        return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "APP_")), "_", ".")
    }), nil)
    // 4. 解码到结构体
    var cfg Config
    k.Unmarshal("", &cfg)
    return &cfg
}
```

### 配置文件示例（config.yaml）
```yaml
server:
  port: 8080
  metrics_port: 9090
  pprof_port: 6060
  read_timeout: 10s
  write_timeout: 30s

database:
  dsn: "postgres://user:pass@localhost:5432/app?sslmode=disable"
  max_open: 25
  max_idle: 10
  max_lifetime: 5m

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

authz:
  endpoint: "http://auth-center:8080"
  timeout: 3s
  cache_ttl: 5m

telemetry:
  service_name: "vibe-go"
  endpoint: "http://otel-collector:4317"
  sample_rate: 0.1

plugins:
  search:
    enabled: false
    # host: "http://localhost:7700"
    # api_key: ""
```

---

## 六、日志规范（slog）

### 初始化
```go
// internal/gateway/middleware.go 或 cmd/server/main.go
func initLogger(cfg TelemetryConfig) *slog.Logger {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })
    // 可选：包装为 OTel Handler，自动注入 trace_id / span_id
    return slog.New(otelHandler.New(handler))
}
```

### 日志使用规范
```go
// 正确：结构化日志
slog.Info("tag applied",
    "layer", "L4",
    "entity_type", req.EntityType,
    "entity_id", req.EntityID,
    "tag_value_id", req.TagValueID,
    "trace_id", span.SpanContext().TraceID(),
)

// 禁止
log.Println("tag applied: " + req.EntityID)  // 非结构化
fmt.Println(req)                              // 禁止用于日志
```

### 敏感字段脱敏
```go
// 脱敏 Handler 包装
type SensitiveHandler struct {
    next    slog.Handler
    sensitiveFields map[string]bool
}

func (h *SensitiveHandler) Handle(ctx context.Context, r slog.Record) error {
    // 替换 password, token, api_key 等字段为 "***"
    // ...
}
```

---

## 七、可观测性（OpenTelemetry）

### 初始化
```go
// internal/telemetry/telemetry.go
func Init(cfg TelemetryConfig) (func(context.Context) error, error) {
    res, _ := resource.New(context.Background(),
        resource.WithAttributes(
            semconv.ServiceNameKey.String(cfg.ServiceName),
        ),
    )

    // TracerProvider
    exporter, _ := otlptracegrpc.New(context.Background(),
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
    )
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
        sdktrace.WithBatcher(exporter),
    )
    otel.SetTracerProvider(tp)

    // MeterProvider (Prometheus)
    meterExporter, _ := prometheus.New()
    mp := sdkmetric.NewMeterProvider(
        sdkmetric.WithReader(meterExporter),
        sdkmetric.WithResource(res),
    )
    otel.SetMeterProvider(mp)

    return func(ctx context.Context) error {
        tp.Shutdown(ctx)
        mp.Shutdown(ctx)
        return nil
    }, nil
}
```

### Metrics 端点
```go
// 独立端口暴露 Prometheus metrics + pprof
mux := http.NewServeMux()
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    promhttp.Handler().ServeHTTP(w, r)
})
mux.HandleFunc("/healthz", healthzHandler)
mux.HandleFunc("/readyz", readyzHandler(db, redis))
go http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.MetricsPort), mux)
```

### 健康检查
```go
func readyzHandler(db *ent.Client, rdb *redis.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
        defer cancel()

        // 检查 PostgreSQL
        if _, err := db.DB().ExecContext(ctx, "SELECT 1"); err != nil {
            http.Error(w, "database unhealthy", http.StatusServiceUnavailable)
            return
        }

        // 检查 Redis
        if err := rdb.Ping(ctx).Err(); err != nil {
            http.Error(w, "redis unhealthy", http.StatusServiceUnavailable)
            return
        }

        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    }
}
```

---

## 八、存储层规范

### Ent 使用规范
```go
// internal/storage/client.go
type Client struct {
    ent  *ent.Client
    redis *redis.Client
}

func New(cfg DatabaseConfig, redisCfg RedisConfig) (*Client, error) {
    // Ent 客户端（底层 pgx）
    drv, err := sql.Open("pgx", cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("open db: %w", err)
    }
    drv.SetMaxOpenConns(cfg.MaxOpen)
    drv.SetMaxIdleConns(cfg.MaxIdle)
    drv.SetConnMaxLifetime(cfg.MaxLifetime)

    client := ent.NewClient(ent.Driver(drv))

    // Redis 客户端
    rdb := redis.NewClient(&redis.Options{
        Addr:     redisCfg.Addr,
        Password: redisCfg.Password,
        DB:       redisCfg.DB,
    })

    return &Client{ent: client, redis: rdb}, nil
}
```

### Redis 使用规范
```go
// 缓存操作
func (c *Client) GetCached(ctx context.Context, key string, dest any, ttl time.Duration, fetch func() (any, error)) error {
    // 1. 查 Redis
    val, err := c.redis.Get(ctx, key).Result()
    if err == nil {
        return json.Unmarshal([]byte(val), dest)
    }
    if err != redis.Nil {
        slog.Warn("cache get error", "key", key, "error", err)
    }

    // 2. 未命中，查数据源
    data, err := fetch()
    if err != nil {
        return err
    }

    // 3. 写入 Redis
    raw, _ := json.Marshal(data)
    c.redis.Set(ctx, key, raw, ttl)

    return nil
}
```

### 数据库迁移
```bash
# 创建迁移文件
migrate create -ext sql -dir storage/migrations -seq create_initial_tables

# 执行迁移
migrate -path storage/migrations -database "$DATABASE_URL" up
```

---

## 九、DSL规范（声明式 + 明确映射）

### 领域事件Schema（强制统一结构）
```yaml
domain_events:
  schema_version: "1.0"
  required_fields:
    event_id:
      type: ulid
      desc: "全局唯一事件ID（ULID，按时间排序）"
    aggregate_type:
      type: string
      desc: "实体类型"
    aggregate_id:
      type: ulid
      desc: "实体ID"
    event_type:
      type: string
      pattern: "{Aggregate}{Action}V{Version}"
      desc: "事件类型"
    payload:
      type: json
      desc: "事件数据（向前兼容）"
    occurred_at:
      type: timestamp_iso8601
      desc: "发生时间（UTC）"
    idempotency_key:
      type: string
      formula: "{aggregate_id}:{event_type}:{aggregate_version}"
      desc: "幂等键"
    version:
      type: int
      desc: "聚合根版本号（乐观锁）"
```

### 状态机（声明式规则模板库）
```yaml
state_machines:
  entity: ExampleEntity
  initial: CREATED
  version_field: version
  states: [CREATED, ACTIVE, DISABLED, ARCHIVED]
  transitions:
    - event: Activate
      from: CREATED
      to: ACTIVE
      guards:
        - rule: field_not_zero
          field: Name
      actions:
        - action: set_timestamp
          field: ActivatedAt
        - action: increment_version
        - action: publish_event
          event: ExampleEntityActivatedV1
```

#### 规则模板库（内置实现，DSL只引用）
- `field_not_zero`：检查字段 != 零值
- `field_range`：数值在[min, max]
- `time_within`：时间距离创建时间在duration内
- `time_after`：当前时间在指定时间之后
- `relation_exists`：关联实体存在（检查外键）
- `unique_in_scope`：在指定范围内唯一
- `set_timestamp`：设置当前时间戳
- `increment_version`：版本号+1（乐观锁）
- `publish_event`：发布领域事件（写入Outbox）

### 业务规则（声明式校验）
```yaml
business_rules:
  - name: NameValidation
    entity: ExampleEntity
    trigger: before_create
    rules:
      - rule: length_between
        field: Name
        min: 1
        max: 100
        error:
          code: L2_201
          message: "名称长度必须在1-100字符之间"
```

---

## 十、错误码规范（分层分配）

### 错误码注册表
```yaml
error_code_registry:
  format: "L{layer_number}{sequence:3d}"
  ranges:
    L1_Storage:
      range: [001, 199]
      examples:
        "001": "数据库连接失败"
        "002": "唯一约束冲突"
        "003": "外键约束违反"
        "010": "Outbox轮询失败"

    L2_Domain:
      range: [200, 399]
      examples:
        "201": "状态机转换非法"
        "202": "业务规则校验失败"
        "203": "聚合根版本冲突（乐观锁）"
        "204": "领域事件序列化失败"

    L3_Authz:
      range: [400, 599]
      examples:
        "401": "权限拒绝"
        "402": "身份验证失败（JWT无效/过期）"
        "403": "Rate Limit exceeded"
        "404": "权限服务不可用"

    L4_Service:
      range: [600, 799]
      examples:
        "601": "输入校验失败"
        "602": "插件调用失败"
        "603": "工作流启动失败"

    L5_Gateway:
      range: [800, 999]
      examples:
        "801": "请求参数解析失败"
        "802": "内部服务错误"
        "803": "上游服务不可用"
```

### 错误包装规范
```go
// 每层错误必须包装，携带层级信息
if err != nil {
    return nil, domainerror.New(
        code:    "L2_201",
        message: "状态机转换非法",
        details: map[string]any{
            "current_state": entity.State,
            "target_event":  event,
        },
        cause: err,
    )
}
```

---

## 十一、测试规范

### 测试数据隔离
```yaml
test_isolation_strategy:
  unit_tests:
    policy: "零外部依赖"
    tools: [gomock, testify, miniredis]
    rule: "所有L1-Storage接口必须可mock，单元测试不连接真实DB/Redis"

  integration_tests:
    policy: "Testcontainers（每测试一套新实例）"
    implementation: |
      containers := map[string]testcontainers.Container{
        "postgres": mustStartPostgres(),
        "redis":    mustStartRedis(),
      }
      defer containers["postgres"].Terminate(ctx)
      // 独立 schema 隔离
      db.Exec("CREATE SCHEMA test_" + testID)
      db.Exec("SET search_path TO test_" + testID)
    fixtures: "testdata/fixtures.yaml"

  e2e_tests:
    policy: "独立命名空间"
```

### 测试文件组织
```
internal/
  domain/
    entity_test.go        # 单元测试
  storage/
    postgres/
      entity_test.go      # 集成测试 (//go:build integration)
tests/
  e2e/
    api_test.go           # E2E 测试
testdata/
  fixtures.yaml           # 测试数据
```

---

## 十二、基础设施配置参考

```yaml
server:
  port: 8080
  metrics_port: 9090
  pprof_port: 6060
  read_timeout: 10s
  write_timeout: 30s

database:
  dsn: "postgres://user:pass@localhost:5432/app?sslmode=disable"
  max_open: 25
  max_idle: 10
  max_lifetime: 5m

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

telemetry:
  service_name: "vibe-go"
  endpoint: "http://localhost:4317"
  sample_rate: 0.1

# 日志（通过环境变量或命令行控制，不需要配置文件）
# APP_LOG_LEVEL=debug
```

---

## 十三、生成脚本规范

### scripts/check_architecture.sh
```bash
#!/bin/bash
set -e
echo "Checking layer dependencies..."

ALLOWED_DEPS=(
  "gateway:authz"
  "authz:service"
  "service:domain"
  "service:storage"
  "domain:storage"
)

MODULE=$(go list -m)
for dir in gateway authz service domain storage; do
  deps=($(go list -f '{{ join .Deps "\n" }}' ./internal/$dir 2>/dev/null | grep "^$MODULE/" || true))
  for dep in "${deps[@]}"; do
    allowed=false
    for rule in "${ALLOWED_DEPS[@]}"; do
      from=${rule%%:*}; to=${rule##*:}
      if [[ "$dir" == "$from" && "$dep" == *"/$to"* ]]; then
        allowed=true; break
      fi
    done
    if [[ "$allowed" == false && -n "$dep" ]]; then
      echo "ERROR: Layer '$dir' illegally depends on '$dep'"
      exit 1
    fi
  done
done

echo "✓ Architecture check passed"
```

### scripts/verify.sh
```bash
#!/bin/bash
set -e
echo "=== 1. Architecture Compliance ==="
./scripts/check_architecture.sh

echo "=== 2. Build & Static Analysis ==="
go build ./...
go vet ./...
staticcheck ./...

echo "=== 3. Unit Tests ==="
go test -race -count=1 -coverprofile=coverage.out ./internal/...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
echo "Coverage: $COVERAGE%"

echo "=== 4. Integration Tests ==="
go test -tags=integration -v ./tests/integration/...

echo "✓ All checks passed."
```
