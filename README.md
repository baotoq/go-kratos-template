# go-kratos-template

A batteries-included **[go-kratos](https://go-kratos.dev/)** monorepo template for shipping production microservices fast. Clone it, rename one app, and you've got proto-driven gRPC + HTTP APIs, Wire DI, Dapr secrets/pubsub, a durable Dapr Workflow example, a one-command Tilt dev loop with live reload and Delve, and integration tests against real Redis via testcontainers.

## What's in the box

- **Layered architecture** — `service → biz → data`, DDD-friendly, infra primitives only in `data/`
- **Proto-first APIs** — gRPC + HTTP + OpenAPI generated from `api/**/*.proto`
- **Wire** for compile-time dependency injection
- **Dapr** sidecar for secret injection (`secretstore`), event publishing (`pubsub`), and durable workflow state (`statestore`)
- **Dapr Workflow** worked example — durable, replayable orchestration of activities
- **Tilt** for local Kubernetes dev — live reload + remote Delve on `:7000`
- **testcontainers-go** integration tests (real Redis, no infra mocks)
- **Devcontainer** with all toolchains pinned (Go, kratos, protoc, wire, kubectl, helm, dapr, tilt, gh, claude-code)

## Quick start

### 1. Clone & open in the devcontainer

```bash
git clone https://github.com/<you>/<your-repo>.git
cd <your-repo>
code .                                  # VS Code → "Reopen in Container"
cp .devcontainer/.env.example .devcontainer/.env   # if present, fill in per-developer overrides
```

The devcontainer mounts the host Docker socket (testcontainers + Tilt builds) and the host kubeconfig (rewritten to `host.docker.internal`). Bring your own local Kubernetes — Docker Desktop or OrbStack.

### 2. Install toolchain & generate code

```bash
make init        # protoc plugins, kratos, wire — all pinned
make all         # api + config + generate (wire)
```

### 3. Run it

```bash
make dev         # tilt up --continue   (Delve attaches in background)
# or
make debug       # tilt up              (Delve waits for VS Code on :7000)
```

Forwards: HTTP `:8000`, gRPC `:9000`, Delve `:7000`, Redis `:6379`.

```bash
# Brew a cup
curl -X POST http://localhost:8000/v1/coffee \
     -H 'Content-Type: application/json' \
     -d '{"beans":"arabica","size":"large"}'
# => {"instanceId":"..."}
```

## Renaming for your project

The template ships one app called `coffee`. To make it yours:

1. **Module name** — edit [go.mod](go.mod) (`module coffee` → `module github.com/you/yourapp`) and run `go mod tidy`.
2. **App directory** — `mv app/coffee app/<yourapp>`. Update [Makefile](Makefile) (`./bin/coffee` → `./bin/<yourapp>`) and [Tiltfile](Tiltfile) (`./app/coffee/...`, `dist/coffee`).
3. **API namespace** — `mv api/coffee api/<yourapp>` and update the `package` / `option go_package` lines in `api/<yourapp>/**/*.proto`. Re-run `make api`.
4. **Helm release** — rename the chart in [deploy/helm/](deploy/helm/) and the namespace in [deploy/k8s/](deploy/k8s/) (default: `coffee`).
5. **Dapr app-id** — update component `metadata.namespace` in [deploy/k8s/base/infra/dapr/](deploy/k8s/base/infra/dapr/).
6. Run `make all && make wire` (from `app/<yourapp>/`) and you're done.

To add **a second service** instead, copy `app/coffee/` to `app/<newapp>/`, copy the proto tree under `api/`, regenerate, and add a Tilt resource for the new binary.

## Layout

```
api/<app>/<domain>/v<N>/      protobuf contracts (source of truth)
  ├─ *.proto
  └─ generated *.pb.go, *_grpc.pb.go, *_http.pb.go

app/<app>/
  cmd/server/                 entrypoint + Wire
  internal/
    conf/                     config proto → conf.pb.go
    server/                   HTTP + gRPC bootstrap
    service/                  proto server impl (transport layer)
    biz/                      domain logic, workflow lifecycle
    data/                     infra primitives only (Dapr workflow client)

deploy/
  helm/                       Redis chart
  k8s/                        base + overlays (Dapr components live here)

docs/                         topic-specific guides
.devcontainer/                pinned toolchain + Docker-out-of-Docker
```

**Dependency rule:** `service → biz → data`. `data` only provides infra primitives (Dapr clients, etc.); `biz` owns domain logic and never imports `data` types beyond those primitives.

## Commands

### Root

| Command         | What it does |
| --------------- | ------------ |
| `make init`     | Install pinned protoc plugins, wire |
| `make api`      | Generate Go/gRPC/HTTP + OpenAPI from `api/**/*.proto` |
| `make config`   | Generate Go structs from `internal/conf/*.proto` |
| `make generate` | `go generate ./...` + `go mod tidy` (wire) |
| `make all`      | api + config + generate |
| `make build`    | Build binary → `./bin/<app>` |
| `make test`     | `go test -race ./...` (requires Docker) |
| `make dev`      | `tilt up --continue` |
| `make debug`    | `tilt up` (Delve waits for debugger) |

### Per-app (from `app/<app>/`)

| Command      | What it does |
| ------------ | ------------ |
| `make wire`  | Regenerate `wire_gen.go` |
| `make run`   | `go run ./cmd/server` |
| `make test`  | `go test -v ./...` |

## Configuration & secrets

Static config lives in [app/&lt;app&gt;/configs/config.yaml](app/coffee/configs/config.yaml). **Secrets are injected at runtime by Dapr** — `main.go` waits up to 60 s for the Dapr sidecar, then loads the secret bundle via [internal/conf/secrets.go](app/coffee/internal/conf/secrets.go). Secret keys are declared as struct tags on `conf.Secrets` and validated non-empty at startup.

Provide secret values via the Dapr `secretstore` component in [deploy/k8s/base/infra/dapr/](deploy/k8s/base/infra/dapr/).

## Testing

TDD-friendly: write the test first, watch it fail for the right reason, implement the minimum, re-run.

Integration tests use **testcontainers-go** to spin up real Redis — there are no infra mocks. `make test` therefore needs Docker running. Helper `startRedis(t)` lives in `app/<app>/internal/data/testhelper_test.go`; containers tear down via `t.Cleanup`.

Use `github.com/stretchr/testify/assert` and structure each test with AAA comments:

```go
// Arrange
// Act
// Assert
```

Single test:

```bash
cd app/<app> && go test -v -run TestFunctionName ./internal/...
```

## Further reading

- [docs/dapr.md](docs/dapr.md) — Dapr setup, components, sidecar wiring
- [docs/dapr-workflow.md](docs/dapr-workflow.md) — Coffee workflow walkthrough
- [CLAUDE.md](CLAUDE.md) — agent-oriented guide to the same architecture

## License

Use it however you like — fork it, rebrand it, ship it.
