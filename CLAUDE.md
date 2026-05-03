# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Root-level (run from repo root)

```bash
make init        # install protoc plugins, wire, ent CLI
make api         # generate Go/gRPC/HTTP code + OpenAPI from api/**/*.proto
make config      # generate Go structs from internal conf.proto files
make generate    # run go generate ./... + go mod tidy (wire + ent)
make build       # build binary → ./bin/greeter
make all         # api + config + generate
make dev         # tilt up --continue (dev env, Delve starts immediately)
make debug       # tilt up (Delve waits for debugger to attach on :7000)
```

### Per-app (run from app/greeter/)

```bash
make wire        # regenerate wire_gen.go
make ent         # regenerate ent ORM from schema
make run         # go run ./cmd/server
make test        # go test -v ./...
```

### Run a single test
```bash
cd app/greeter && go test -v -run TestFunctionName ./internal/...
```

## Architecture

This is a **go-kratos** microservice monorepo. One app (`greeter`) is the template; copy the pattern for new services.

### Layer stack (top → bottom)

```
api/greeter/helloworld/v1/     ← protobuf contracts (source of truth)
  └─ generated: *.pb.go, *_grpc.pb.go, *_http.pb.go

app/greeter/
  cmd/server/                  ← entrypoint + Wire DI wiring
  internal/
    conf/                      ← config proto → generated conf.pb.go
    server/                    ← HTTP + gRPC server setup
    service/                   ← implements generated proto server interface
    biz/                       ← domain logic, repo interfaces (no infra deps)
    data/                      ← repo implementations, ent ORM client, Dapr client
      ent/schema/              ← ent schema definitions (source of truth for DB)
      ent/                     ← generated ORM code (do not edit by hand)
```

**Dependency rule:** `service → biz → data`. `biz` defines repository interfaces; `data` implements them. `biz` never imports `data`, and (by convention going forward) `data` never imports `biz` either — `data` only provides infra primitives (`*ent.Client`, `*workflow.Client`, etc.). `GreeterRepo` is the legacy pattern (`data/greeter.go` still imports `biz`); the order workflow is the new pattern: `biz` owns the entire workflow lifecycle (registry, worker, schedule, fetch) and `data` exposes only `NewWorkflowClient`.

### Dependency injection

Google Wire is used. `wire.go` (build-tag-guarded) declares provider sets; `wire_gen.go` is the generated output. After any change to providers, run `make wire` from `app/greeter/`.

### Config & secrets

Config is loaded from `app/greeter/configs/config.yaml` via kratos `config/file`. **Secrets are injected at runtime by Dapr** (`secretstore` component). `main.go` retries the Dapr sidecar connection up to 12× (60s total) on startup, then overwrites `bc.Data.Database.Source` (from secret key `DATABASE_CONNECTION_STRING`) and `bc.Data.Redis.Addr` (from `REDIS_HOST`). The yaml ships those fields blank — leave them blank in committed config and provide values via the secret store.

### Dapr integration

The app depends on a Dapr sidecar (gRPC on `DAPR_GRPC_PORT`, default `50001`). Three Dapr components are declared in `deploy/k8s/base/infra/dapr/`:
- `secretstore` — secret injection on startup
- `pubsub` — Redis pub/sub for event publishing
- `statestore` — Redis state store with `actorStateStore=true`, required for Dapr workflow runtime persistence

### Database

ent ORM with PostgreSQL (`lib/pq` driver). Schema lives in `internal/data/ent/schema/`. After editing schema, run `make ent`. `data.NewData` calls `client.Schema.Create` on startup (auto-migrate).

### Local dev environment (Tilt)

`tilt up` / `make dev` targets Docker Desktop or OrbStack (`allow_k8s_contexts`). The workflow:
1. `compile` local resource builds a Linux binary into `./dist/greeter` on every Go source change
2. Binary is synced into the running container — no image rebuild
3. Delve debugger runs inside the container; VS Code launch config at `.vscode/launch.json` connects to `:7000`
4. Helm chart at `deploy/helm/` provisions Postgres, Redis, pgAdmin in the `greeter` namespace
5. Dapr is installed via Helm into `dapr-system` namespace

Port forwards: HTTP `8000`, gRPC `9000`, Delve `7000`, Postgres `5432`, Redis `6379`, pgAdmin `5050`.

### Proto conventions

- API protos: `api/<app>/<domain>/v<N>/<name>.proto` → `make api`
- Error reasons: defined in `error_reason.proto` as an enum; errors use `v1.ErrorReason_XXX.String()` as the reason field
- Internal config proto: `internal/conf/conf.proto` → `make config`
- `third_party/` holds vendored proto imports (google, validate)

## Testing

Use TDD: write tests first, confirm they fail for the right reason, then implement the minimal fix and re-run. Do not write maintenance-heavy tests (no exhaustive mocks, no tests that re-assert framework behavior, no tests that break on every refactor). Test behavior, not implementation.

Tests use **testcontainers-go** to spin up real Postgres/Redis containers — there are no DB mocks. `make test` therefore requires Docker (Docker Desktop / OrbStack / Colima) to be running. Helpers `startPostgres(t)` / `startRedis(t)` live in `app/greeter/internal/data/testhelper_test.go`; containers are torn down via `t.Cleanup`.

Use `github.com/stretchr/testify/assert` for assertions. Structure every test with AAA comments:
```go
// Arrange
// Act
// Assert
```

See `docs/integration-tests.md` for the full pattern.

## Further reading

Topic-specific docs live in `docs/`:
- `docs/dapr.md` — Dapr setup and component conventions
- `docs/dapr-workflow.md` — Dapr workflow example (orders) layered across biz/data
- `docs/ent-go.md` — ent schema authoring and migration workflow
- `docs/integration-tests.md` — testcontainers patterns
