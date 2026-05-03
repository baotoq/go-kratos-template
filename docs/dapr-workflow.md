# Dapr Workflow — Coffee example

A toddler-simple example of a [Dapr Workflow](https://docs.dapr.io/getting-started/quickstarts/workflow-quickstart/):

```
brew(beans, size)
   └─ step 1: GrindBeansActivity(beans)        → "ground arabica"
   └─ step 2: BrewActivity(grounds, size)      → "a large cup of ground arabica coffee"
```

Why this matters: each step is *durable*. If the pod dies between step 1 and step 2, Dapr replays the workflow from the persisted history and resumes from where it left off.

## Files

| File | Purpose |
| --- | --- |
| `api/coffee/v1/coffee.proto` | gRPC + HTTP contract: `Brew`, `Check` |
| `app/coffee/internal/biz/coffee.go` | `CoffeeOrder`/`CoffeeStatus` types, `CoffeeUsecase` (registers workflow + activities, runs the worker, exposes `Brew`/`Check`) |
| `app/coffee/internal/biz/coffee_workflow.go` | The two-step workflow + the two activity functions |
| `app/coffee/internal/data/workflow.go` | Just the `*workflow.Client` factory (one-line wrapper around the existing Dapr sidecar conn) |
| `app/coffee/internal/service/coffee.go` | gRPC/HTTP service implementation |
| `deploy/k8s/base/infra/dapr/statestore.yaml` | Redis state store with `actorStateStore=true` — required so Dapr can persist workflow state |
| `http/coffee.http` | REST Client / JetBrains HTTP requests to drive the demo |

## Try it

```bash
make dev   # tilt up

# 1. Start brewing
curl -X POST http://localhost:8000/v1/coffee \
     -H 'Content-Type: application/json' \
     -d '{"beans":"arabica","size":"large"}'
# => {"instanceId":"..."}

# 2. Peek at the cup
curl http://localhost:8000/v1/coffee/<instance_id>
# => {"instanceId":"...","status":"COMPLETED","cup":"\"a large cup of ground arabica coffee\""}
```

## Layering

The workflow lives in `biz` (it is business logic). `data` only provides the `*workflow.Client` primitive — it never imports `biz`. The dependency direction is:

```
service → biz → workflow.Client (provided by data)
                     │
                     ▼
                 Dapr sidecar (gRPC) → Redis statestore
```

## Adding more steps

Pattern for a new activity:

```go
// 1. Add the activity in biz/coffee_workflow.go
func PourActivity(actx workflow.ActivityContext) (any, error) {
    var cup string
    if err := actx.GetInput(&cup); err != nil { return nil, err }
    return cup + " in a paper cup", nil
}

// 2. Register it in NewCoffeeUsecase (biz/coffee.go) — add to the slice:
for _, a := range []workflow.Activity{GrindBeansActivity, BrewActivity, PourActivity} { ... }

// 3. Call it from MakeCoffeeWorkflow:
var poured string
if err := ctx.CallActivity(PourActivity, workflow.WithActivityInput(cup)).Await(&poured); err != nil {
    return nil, err
}
```
