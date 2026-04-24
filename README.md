# vibe-go

**[中文](README_zh.md)** | English

Go production framework scaffold.

> **Purpose-built for AI agents and vibe coding.**
> vibe-go is designed from the ground up for the era where autonomous agents write your codebase.
> It provides the architectural constraints that AI needs but doesn't have —
> so the code it produces is production-grade, not just "it compiles."

## What

vibe-go is a compilable, deployable Go backend framework scaffold with 5-layer architecture and plugin inversion. Fork it, fill in your business logic, and ship a production-grade service.

## Tech Stack

| Category | Choice | Purpose |
|---|---|---|
| API | `connect-go` | gRPC + HTTP dual-mode |
| ORM | `ent` (pgx) | Type-safe query, code generation |
| Config | `koanf` | YAML + env, explicit DI |
| Logging | `log/slog` | Structured JSON, stdlib |
| ID | `oklog/ulid` | Globally unique, time-sortable |
| Cache | `go-redis/v9` | Cache, event stream, distributed lock |
| Observability | OpenTelemetry + Prometheus | Tracing, metrics |
| DB Migration | `golang-migrate` | Versioned SQL migration |
| Testing | testify + gomock + testcontainers + miniredis | Unit / integration / E2E |

## Architecture

```
L5-Gateway → L3-Authz → L4-Service → L2-Domain → L1-Storage
```

- Core layers define interfaces, plugin layers implement them
- Core MUST NOT import plugin implementations
- L2-Domain has zero external dependencies

```
cmd/server/main.go           # Entry point, DI assembly
internal/
  gateway/                   # L5: Connect handler, middleware
  authz/                     # L3: Authorization
  service/                   # L4: Business orchestration (interfaces.go)
  domain/                    # L2: Domain core (zero deps)
  storage/                   # L1: Ent + PostgreSQL + Redis
plugins/                     # Plugin implementations
api/{package}/v1/            # Protobuf definitions
```

## Task Management with LRA (Recommended)

vibe-go's task breakdown and agent workflow are designed around [LRA (Long-Running Agent)](https://hotjp.github.io/long-run-agent/) — a task management tool purpose-built for AI agent multi-turn development.

LRA is optional. Use your own workflow if you prefer. But we strongly recommend pairing it with vibe-go:

- Task definitions carry 5-section context (goal / contract / deps / conventions / acceptance) — agent starts cold and still delivers
- Atomic claim + lock mechanism — multiple agents work in parallel without conflicts
- Constitution quality gates — every task must pass before marked complete
- Seamless integration with `TASK-BREAKDOWN.md` methodology

**Install:**

```bash
# Check if installed
lra --version

# Install
pip install long-run-agent
```

**Learn more:** [Documentation](https://hotjp.github.io/long-run-agent/) | [GitHub](https://github.com/hotjp/long-run-agent)

## Quick Start

```bash
# 1. Fork and rename
# 2. Fill CLAUDE.md (Project, Description, LRA profile)
# 3. Run task breakdown for your business domain
# 4. Start coding
```

## For Agent

Read docs in this order. Each doc serves a single purpose — don't scan everything upfront.

```
CLAUDE.md               <- Architecture constraints & coding rules (always loaded)
    |
docs/TASK-BREAKDOWN.md  <- Pick task -> get self-contained 5-section context
    |                         (no need to read other docs unless task references them)
docs/DESIGN.md          <- Business details: entities, API proto, DDL, flows
docs/architecture.md    <- Technical details: config, logging, telemetry, testing
```

### Agent Workflow

```
lra ready                              # Find available tasks
lra claim <id>                         # Claim atomically
lra show <id>                          # Read task details
    |
Read TASK-BREAKDOWN.md section TaskID  # Self-contained context
    |
Implement -> Test -> Commit
    |
lra set <id> completed
lra check <id>                         # Run Constitution quality gates
lra set <id> truly_completed           # Done
```

### Task Dependency Map

```
Phase 0:  T00 -> T01 || T02 || T03 -> T04 -> T05
Phase 1:  T01 -> T10 || T12;  T10 -> T11 || T13
Phase 2:  T05 -> T20||T30||T40||T50||T60          (Domain, parallel)
                 -> T21||T31||T41||T51||T61          (Storage, after each Domain)
                 -> T22||T32||T42||T53||T62          (Service, after each Storage)
Phase 5:  T05 -> T70 || T71                     (Plugins, parallel)
Phase 6:  T61+T52+T70+T71 -> T80               (AutoTag orchestration)
Phase 7:  All -> T90 -> T91 -> T92 -> T93         (Gateway + Authz + Assembly)
```

## Documentation

| Document | Purpose | When to read |
|---|---|---|
| [CLAUDE.md](CLAUDE.md) | Architecture constraints, coding rules | Always (auto-loaded) |
| [docs/TASK-BREAKDOWN.md](docs/TASK-BREAKDOWN.md) | Task definitions with full context | Before each task |
| [docs/DESIGN.md](docs/DESIGN.md) | Business design (entities, API, DDL, flows) | When task references business details |
| [docs/architecture.md](docs/architecture.md) | Technical specs (config, logging, telemetry) | When task references technical details |
| [docs/TASK-PROMPT.md](docs/TASK-PROMPT.md) | Task splitting methodology | When creating new task breakdowns |
| [lra.md](lra.md) | LRA command reference | When managing tasks |

## Demo

The `docs/DESIGN.md` and `docs/TASK-BREAKDOWN.md` contain a complete tag management system (Tag Sense) as a demo business to validate the framework. They are marked with `[Demo]` and do not belong to the framework itself.

## API Docs: buf + proto → TypeScript

**Skip Swagger.** All APIs are defined in `.proto` and generated to TypeScript clients via `buf`. Frontend imports directly.

```
.proto → buf generate → @bufbuild/connect-web (TS client)
                                   ↓
                             Browser import
```

**Why not OpenAPI/Swagger:**
- Proto → TS type generation, zero drift between frontend and backend
- Agent gets TS client code directly — no need to parse HTTP semantics
- Compile-time type checking, not runtime documentation

**Core commands:**
```bash
# Install tools
brew install buf protobuf

# Generate TS client
buf generate

# Frontend deps
npm install @bufbuild/connect-web @bufbuild/protobuf
```

**Key files:**
- `api/{package}/v1/*.proto` — API definitions
- `buf.yaml` — Generation config
- `api/{package}/v1/generated/ts/` — Generated TS client

## License

MIT
