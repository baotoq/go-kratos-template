# Testing

This template uses [testcontainers-go](https://golang.testcontainers.org/) for
tests that need real infrastructure (Postgres, Redis, …) instead of mocks.
Examples live alongside the production code in `app/greeter/internal/data/`:

| File | What it shows |
| ---- | ------------- |
| `testhelper_test.go` | Reusable `startPostgres(t)` / `startRedis(t)` helpers |
| `greeter_test.go`    | End-to-end test of `GreeterRepo` against ent + Postgres |
| `redis_test.go`      | Standalone Redis SET/GET example |

## Running

```bash
cd app/greeter && make test
```

…which resolves to:

```bash
go test -count=1 -v ./...
```

`-count=1` disables Go's test cache so containers actually start each run
instead of replaying a cached PASS. **Docker (or OrbStack/Podman) must be
running** — there is no separate "unit only" mode in this template; container
tests live next to unit tests and run together.

## Conventions

- **AAA structure** — every test body is split into `// Arrange`, `// Act`,
  `// Assert` blocks (see `CLAUDE.md` → Testing).
- **`testify/assert` for outcomes, `require` for setup** — use
  `assert.X(t, …)` to verify the thing under test (so a single failure
  doesn't mask the others), and `require.NoError(t, err)` only for
  preconditions where continuing past failure would panic (container start,
  DB connect).
- **Cleanup-before-error-check** — call
  `testcontainers.CleanupContainer(t, c)` *before* `require.NoError(t, err)` so
  partially-started containers are still torn down.
- **No `time.Sleep`** — rely on the module's wait strategies
  (`postgres.BasicWaitStrategies()` etc.). The Redis module has built-in
  readiness checks via its image.
- **External test package** — tests live in `package data_test`; they go
  through the public `NewData` / `NewGreeterRepo` constructors, exactly like
  production wiring. This both validates the public API and serves as
  copy-paste documentation.
- **`t.Parallel()`** — each test gets its own container, so parallelism is
  safe and recommended.

## Adding a new test

1. Create `*_test.go` next to the code under test.
2. Use the helpers in `testhelper_test.go`, or add a new helper there if you
   need a different module (Kafka, RabbitMQ, …).

## CI

Most CI runners (GitHub Actions `ubuntu-latest`, GitLab shared runners, etc.)
have Docker preinstalled, so `make -C app/greeter test` works as-is. On
self-hosted runners, ensure the `docker` socket is reachable by the test
process.
