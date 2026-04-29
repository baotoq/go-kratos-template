# Dapr

Dapr (Distributed Application Runtime) is a sidecar-based runtime that provides portable APIs for communication, state, pub/sub, workflows, and secrets — abstracting the underlying infrastructure. Each app gets a `daprd` sidecar on `localhost:3500` (HTTP) or `localhost:50001` (gRPC). Apps never talk to each other directly; all traffic routes through sidecars.

## Architecture

```
[App :8000/:9000] <--gRPC--> [daprd sidecar :3500/:50001]
                                      |
                          [Dapr component (Redis/Kafka/etc)]
```

**Control plane (Kubernetes mode):**
| Service | Role |
|---------|------|
| `dapr-operator` | Manages component CRDs, notifies sidecars of updates |
| `dapr-sidecar-injector` | Injects `daprd` container on annotated pods |
| `dapr-sentry` | Certificate Authority — issues mTLS certs to all sidecars |
| `dapr-placement` | Actor placement table (only needed for Actors) |

## Building Blocks

| Block | API | Common use |
|-------|-----|-----------|
| **Service invocation** | `POST /v1.0/invoke/{appId}/method/{method}` | RPC between services with retries + tracing |
| **State management** | `POST /v1.0/state/{store}` | Key-value store (Redis, Postgres, etc) |
| **Pub/sub** | `POST /v1.0/publish/{pubsub}/{topic}` | Event-driven messaging (Kafka, Redis Streams, NATS) |
| **Bindings** | `POST /v1.0/bindings/{name}` | Trigger on/from external systems (cron, S3, SMTP) |
| **Actors** | `PUT /v1.0/actors/{type}/{id}/method/{method}` | Stateful virtual actors with timers |
| **Secrets** | `GET /v1.0/secrets/{store}/{key}` | Fetch secrets from Vault/K8s secrets/env |
| **Configuration** | `GET /v1.0-alpha1/configuration/{store}` | Subscribe to config changes |
| **Distributed lock** | `POST /v1.0-beta1/lock/{store}` | Mutex across instances |
| **Workflow** | `POST /v1.0-beta1/workflows/{backend}/{name}/start` | Durable saga/orchestration |
| **Jobs** | `POST /v1.0-alpha1/jobs/{name}` | Scheduled/delayed job execution |
| **Cryptography** | `POST /v1.0-alpha1/crypto/{component}` | Encrypt/decrypt/sign via key store |

## Component YAML

Components define which backing technology each block uses. Applied via `kubectl apply` or placed in `~/.dapr/components/` (self-hosted).

```yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: statestore
  namespace: default
spec:
  type: state.redis
  version: v1
  metadata:
    - name: redisHost
      value: "redis:6379"
    - name: redisPassword
      secretKeyRef:
        name: redis-secret
        key: password
```

Common `spec.type` values: `state.redis`, `state.postgresql`, `pubsub.redis`, `pubsub.kafka`, `bindings.cron`, `secretstores.kubernetes`.

## Sidecar Configuration

```yaml
apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: appconfig
spec:
  tracing:
    samplingRate: "1"
    otel:
      endpointAddress: "otel-collector:4317"
      isSecure: false
      protocol: grpc
  metrics:
    enabled: true
  accessControl:
    defaultAction: deny
    policies:
      - appId: order-processor
        defaultAction: allow
        namespace: default
        operations:
          - name: /v1/orders
            httpVerb: ["POST"]
            action: allow
```

## Go SDK

```bash
go get github.com/dapr/go-sdk
```

### Client init

```go
import dapr "github.com/dapr/go-sdk/client"

// Reads DAPR_GRPC_PORT (default 50001) or DAPR_GRPC_ENDPOINT
client, err := dapr.NewClient()
if err != nil {
    panic(err)
}
defer client.Close()

// Wait for sidecar readiness
if err := client.Wait(ctx, 30*time.Second); err != nil {
    panic("sidecar not ready: " + err.Error())
}
```

Named variants: `dapr.NewClientWithAddress("127.0.0.1:50002")`, `dapr.NewClientWithSocket("/tmp/dapr.socket")`.

### Service invocation

```go
// No content
resp, err := client.InvokeMethod(ctx, "order-processor", "/process", "POST")

// With body
resp, err := client.InvokeMethodWithContent(ctx, "order-processor", "/process", "POST",
    &dapr.DataContent{Data: []byte(`{"id":1}`), ContentType: "application/json"},
)
```

Receive invocations by registering a handler on the service side (see Service section below).

### State management

```go
store := "statestore"

// Save
client.SaveState(ctx, store, "user:42", []byte(`{"name":"Alice"}`), nil)

// Get
item, _ := client.GetState(ctx, store, "user:42", nil)
fmt.Println(string(item.Value))

// Delete
client.DeleteState(ctx, store, "user:42", nil)

// Conditional update (optimistic concurrency)
client.SaveStateWithETag(ctx, store, "user:42", []byte(`{"name":"Bob"}`), item.Etag, nil)

// Bulk save
client.SaveBulkState(ctx, store, []*dapr.SetStateItem{
    {Key: "k1", Value: []byte("v1")},
    {Key: "k2", Value: []byte("v2"), Metadata: map[string]string{"ttl": "30s"}},
}...)
```

### Pub/sub — publish

```go
client.PublishEvent(ctx, "pubsub", "orders", []byte(`{"orderId":"abc"}`))
```

### Pub/sub — subscribe (HTTP service)

```go
import (
    "github.com/dapr/go-sdk/service/common"
    daprd "github.com/dapr/go-sdk/service/http"
)

s := daprd.NewService(":8080")

s.AddTopicEventHandler(&common.Subscription{
    PubsubName: "pubsub",
    Topic:      "orders",
    Route:      "/orders",
}, func(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
    log.Printf("order: id=%s data=%v", e.ID, e.Data)
    return false, nil
})

s.Start()
```

gRPC variant: use `daprd "github.com/dapr/go-sdk/service/grpc"` — same `AddTopicEventHandler` API, lower latency.

### Bindings

```go
// Output (fire-and-forget)
client.InvokeOutputBinding(ctx, &dapr.InvokeBindingRequest{
    Name:      "smtp-binding",
    Operation: "create",
    Data:      []byte(`{"to":"user@example.com","subject":"Hello"}`),
})

// Input binding handler (on service side)
s.AddBindingInvocationHandler("/cron", func(ctx context.Context, in *common.BindingEvent) ([]byte, error) {
    log.Printf("cron: %s", string(in.Data))
    return nil, nil
})
```

### Actors

```go
type MyActor struct{ actors.Actor }

func (a *MyActor) MyMethod(ctx context.Context, req *actors.Message) (string, error) {
    return "hello", nil
}

actors.RegisterActor(&MyActor{})
actorClient := actors.NewClient(daprClient)
actorID := actors.NewActorID("myactor")

actorClient.SaveActorState(ctx, "myactorstore", actorID, map[string]interface{}{"x": 1})
resp, _ := actorClient.InvokeActorMethod(ctx, "myactorstore", actorID, "MyMethod",
    &actors.Message{Data: []byte("ping")})
actorClient.DeleteActor(ctx, "myactorstore", actorID)
```

### Secrets

```go
secret, _ := client.GetSecret(ctx, "kubernetes", "db-password", nil)
fmt.Println(secret["db-password"])

bulk, _ := client.GetBulkSecret(ctx, "kubernetes", nil)
```

### Service — full handler registration

```go
s := daprd.NewService(":8080")
s.AddServiceInvocationHandler("/echo", func(ctx context.Context, in *common.InvocationEvent) (*common.Content, error) {
    return &common.Content{Data: in.Data, ContentType: in.ContentType}, nil
})
s.AddJobEventHandler("db-backup", func(ctx context.Context, job *common.JobEvent) error {
    log.Printf("backup job: type=%s", job.JobType)
    return nil
})
s.Start()
```

## Deployment

### Self-hosted

```bash
dapr init            # pulls daprd, Redis, Zipkin containers
dapr run --app-id myapp --app-port 8000 --dapr-http-port 3500 -- ./bin/greeter -conf ./configs
dapr dashboard       # opens http://localhost:8080
dapr stop --app-id myapp
```

### Kubernetes

```bash
dapr init -k                         # installs control plane via Helm
dapr status -k                       # check operator/sentry/injector/placement
dapr dashboard -k                    # port-forwards dashboard
```

Add sidecar annotations to any Deployment pod template:

```yaml
annotations:
  dapr.io/enabled: "true"
  dapr.io/app-id: "greeter"
  dapr.io/app-port: "8000"          # HTTP
  dapr.io/app-protocol: "grpc"      # set for gRPC apps
  dapr.io/config: "appconfig"       # reference a Configuration resource
  dapr.io/inject-pluggable-components: "true"
```

## Observability

**Tracing** — OpenTelemetry export. Set `dapr.io/config` to a `Configuration` with `spec.tracing.otel.endpointAddress`. Propagates W3C trace context headers automatically.

**Metrics** — Prometheus endpoint at `daprd:9090/metrics`. Add `spec.metrics.enabled: true` in Configuration. Dashboard at `dapr.io/enable-metrics: "true"` annotation.

**Logging** — JSON structured logs from `daprd`. Inject `dapr.io/log-as-json: "true"` and `dapr.io/log-level: "debug"` annotations.

## Security

**mTLS** — On by default in Kubernetes. Sentry CA issues SPIFFE-format certs to all sidecars. Cert rotation automatic. Disable per-app: `dapr.io/disable-mtls: "true"`.

**Access control** — Define policy in Configuration (`spec.accessControl`). Default deny + per-appId allow rules on methods and HTTP verbs.

**Secret scoping** — In Configuration `spec.secrets.scopes`, set `defaultAccess: deny` and allowlist specific keys per store.

**Namespace isolation** — Add `dapr.io/sidecar-namespace: <ns>` annotation. Cross-namespace invocation: `appId.namespace` format.

## Repo fit — greeter

Current stack: go-kratos v2 layered monolith, Wire DI, no Dapr yet.

| Dapr block | Where to wire in |
|-----------|-----------------|
| State (Redis) | `internal/data/data.go` — replace direct Redis client with `dapr.NewClient().SaveState/GetState`. Remove `conf.Data.Redis` if Dapr manages the connection. |
| Pub/sub | `internal/data/` — new repo for event publishing; `internal/server/` — add a second port for Dapr subscription callbacks |
| Service invocation | `internal/server/` — `NewHTTPServer` already on `:8000`; just add `dapr.io/app-port: "8000"` annotation |
| Secrets | `internal/conf/` — fetch secrets from Dapr on startup instead of plain config.yaml values |
| Workflow | `internal/biz/` — saga-style multi-step logic (e.g., recursive-buy flow) |

K8s: `deploy/k8s/base/greeter.yaml` pod template needs the 4 `dapr.io/*` annotations. Add component YAMLs (statestore, pubsub) alongside in `deploy/k8s/base/`. No new overlay needed for basic sidecar injection.

Wire: add `dapr.NewClient()` as a constructor in `internal/data/data.go`, wire it into `data.ProviderSet`, inject into repo constructors that need state/pubsub access.
