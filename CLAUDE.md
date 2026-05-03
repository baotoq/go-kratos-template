# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Root-level (run from repo root)

```bash
make init        # install protoc plugins, wire
make api         # generate Go/gRPC/HTTP code + OpenAPI from api/**/*.proto
make config      # generate Go structs from internal conf.proto files
make generate    # run go generate ./... + go mod tidy (wire)
make build       # build binary → ./bin/coffee
make all         # api + config + generate
make dev         # tilt up --continue (dev env, Delve starts immediately)
make debug       # tilt up (Delve waits for debugger to attach on :7000)
```

### Per-app (run from app/coffee/)

```bash
make wire        # regenerate wire_gen.go
make run         # go run ./cmd/server
make test        # go test -v ./...
```

### Run a single test
```bash
cd app/coffee && go test -v -run TestFunctionName ./internal/...
```

## Architecture

This is a **go-kratos** microservice monorepo. One app (`coffee`) is the template; copy the pattern for new services.

### Layer stack (top → bottom)

```
api/coffee/v1/              ← protobuf contracts (source of truth)
  └─ generated: *.pb.go, *_grpc.pb.go, *_http.pb.go

app/coffee/
  cmd/server/                  ← entrypoint + Wire DI wiring
  internal/
    conf/                      ← config proto → generated conf.pb.go
    server/                    ← HTTP + gRPC server setup
    service/                   ← implements generated proto server interface
    biz/                       ← domain logic + Dapr workflow lifecycle
    data/                      ← infra primitives only (Dapr workflow client)
```

**Dependency rule:** `service → biz → data`. `data` exposes only infra primitives (`*workflow.Client`); `biz` owns domain logic and never imports `data` types beyond those primitives. The coffee workflow demonstrates the pattern: `biz` owns the entire workflow lifecycle (registry, worker, schedule, fetch) and `data` exposes only `NewWorkflowClient`.

### Dependency injection

Google Wire is used. `wire.go` (build-tag-guarded) declares provider sets; `wire_gen.go` is the generated output. After any change to providers, run `make wire` from `app/coffee/`.

### Config & secrets

Config is loaded from `app/coffee/configs/config.yaml` via kratos `config/file`. **Secrets are injected at runtime by Dapr** (`secretstore` component). `main.go` retries the Dapr sidecar connection up to 12× (60s total) on startup, then loads the secret bundle declared on `conf.Secrets` (struct tags name each required key). The yaml ships those fields blank — provide values via the secret store.

### Dapr integration

The app depends on a Dapr sidecar (gRPC on `DAPR_GRPC_PORT`, default `50001`). Three Dapr components are declared in `deploy/k8s/base/infra/dapr/`:
- `secretstore` — secret injection on startup
- `pubsub` — Redis pub/sub for event publishing
- `statestore` — Redis state store with `actorStateStore=true`, required for Dapr workflow runtime persistence

### Local dev environment (Tilt)

`tilt up` / `make dev` targets Docker Desktop or OrbStack (`allow_k8s_contexts`). The workflow:
1. `compile` local resource builds a Linux binary into `./dist/coffee` on every Go source change
2. Binary is synced into the running container — no image rebuild
3. Delve debugger runs inside the container; VS Code launch config at `.vscode/launch.json` connects to `:7000`
4. Helm chart at `deploy/helm/` provisions Redis in the `coffee` namespace
5. Dapr is installed via Helm into `dapr-system` namespace

Port forwards: HTTP `8000`, gRPC `9000`, Delve `7000`, Redis `6379`.

### Proto conventions

- API protos: `api/<app>/<domain>/v<N>/<name>.proto` → `make api`
- Internal config proto: `internal/conf/conf.proto` → `make config`
- `third_party/` holds vendored proto imports (google, validate)

## Testing

Use TDD: write tests first, confirm they fail for the right reason, then implement the minimal fix and re-run. Do not write maintenance-heavy tests (no exhaustive mocks, no tests that re-assert framework behavior, no tests that break on every refactor). Test behavior, not implementation.

Tests use **testcontainers-go** to spin up real Redis containers — there are no infra mocks. `make test` therefore requires Docker (Docker Desktop / OrbStack / Colima) to be running. Helper `startRedis(t)` lives in `app/coffee/internal/data/testhelper_test.go`; containers are torn down via `t.Cleanup`.

Use `github.com/stretchr/testify/assert` for assertions. Structure every test with AAA comments:
```go
// Arrange
// Act
// Assert
```

## Further reading

Topic-specific docs live in `docs/`:
- `docs/dapr.md` — Dapr setup and component conventions
- `docs/dapr-workflow.md` — Dapr workflow example (coffee) layered across biz/data
