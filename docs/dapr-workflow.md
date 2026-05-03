# Dapr Workflow

Minimal Dapr Workflow example adapted from the [Dapr workflow quickstart](https://docs.dapr.io/getting-started/quickstarts/workflow-quickstart/). The greeter app exposes a tiny `OrderProcessing` workflow:

```
notify ("Received …") → process payment → notify ("Fulfilled …")
```

## Layout

| File | Purpose |
| --- | --- |
| `api/greeter/orders/v1/orders.proto` | gRPC + HTTP contract: `StartOrder`, `GetOrder` |
| `app/greeter/internal/biz/order.go` | `Order` / `OrderStatus` types, `OrderUsecase` (registers workflow, runs worker, schedules, fetches metadata) |
| `app/greeter/internal/biz/order_workflow.go` | Workflow + activity functions |
| `app/greeter/internal/data/workflow.go` | `*workflow.Client` factory — the only infra primitive `data` exposes for orders |
| `app/greeter/internal/service/order.go` | gRPC/HTTP service implementation |
| `deploy/k8s/base/infra/dapr/statestore.yaml` | Redis state store (`actorStateStore=true`), required for workflow persistence |

`data` does not import `biz` for the order workflow: it only provides the `*workflow.Client` primitive. All workflow-specific code lives in `biz`.

## How a request flows

### Startup

`main.go` opens a single Dapr sidecar gRPC connection. Wire builds the graph: `dapr.Client → *workflow.Client → OrderUsecase`. `NewOrderUsecase` registers the workflow + activities, then calls `client.StartWorker(ctx, registry)` so this same process is also the worker that executes scheduled instances. The wire-generated cleanup func cancels the worker context on shutdown.

```
                 (data/workflow.go)        (biz/order.go)
                 ┌──────────────────┐      ┌────────────────┐
 dapr.Client ──► │ workflow.NewClient│ ──► │ NewOrderUsecase│
                 └──────────────────┘      └───────┬────────┘
                                                   │ AddWorkflowN
                                                   │ AddActivity   ┌──────────────┐
                                                   └─StartWorker──►│ Dapr sidecar │
                                                                   └──────────────┘
```

### POST /v1/orders — schedule

```
 Client    OrdersService          OrderUsecase           workflow.Client    Dapr sidecar    Redis
   │            │                      │                       │                │            │
   │ POST /v1/orders                   │                       │                │            │
   │  {item_name,total_cost}           │                       │                │            │
   │───────────►│                      │                       │                │            │
   │            │ Start(ctx, &Order)   │                       │                │            │
   │            │─────────────────────►│                       │                │            │
   │            │                      │ ScheduleWorkflow      │                │            │
   │            │                      │  ("OrderProcessing…", │                │            │
   │            │                      │   WithInput(order))   │                │            │
   │            │                      │──────────────────────►│                │            │
   │            │                      │                       │ ScheduleNew…   │            │
   │            │                      │                       │───────────────►│ persist    │
   │            │                      │                       │                │───────────►│
   │            │                      │                       │   instance_id  │            │
   │            │                      │                       │◄───────────────│            │
   │            │                      │     instance_id       │                │            │
   │            │                      │◄──────────────────────│                │            │
   │            │     instance_id      │                       │                │            │
   │            │◄─────────────────────│                       │                │            │
   │ 200 {"instance_id":"..."}         │                       │                │            │
   │◄───────────│                      │                       │                │            │
```

### Worker — execute

The worker (same process) picks the new instance up from the sidecar and runs `OrderProcessingWorkflow`. Each `CallActivity(...).Await(...)` is a *durable* yield: the workflow function returns, the sidecar persists the pending step, then later replays the workflow with the activity result fed in.

```
   ┌───────────────────────────┐
   │  schedule received        │
   └─────────────┬─────────────┘
                 ▼
   GetInput → Order{ItemName, TotalCost}
                 │
                 ▼
   NotifyActivity("Received car ($15000.00)")
                 │
                 ▼
   ProcessPaymentActivity(order) ──► returns "receipt-car"
                 │
                 ▼
   NotifyActivity("Fulfilled car (receipt-car)")
                 │
                 ▼
   return "receipt-car"   →   runtime_status=COMPLETED, output="receipt-car"
```

Between every activity call, runtime state (history events, pending tasks, output) is checkpointed to the `statestore` Redis component — that is why the component must have `actorStateStore: "true"`.

### GET /v1/orders/{id} — poll status

```
 Client    OrdersService          OrderUsecase           workflow.Client    Dapr sidecar
   │            │                      │                       │                │
   │ GET /v1/orders/{id}               │                       │                │
   │───────────►│                      │                       │                │
   │            │ Get(ctx, id)         │                       │                │
   │            │─────────────────────►│                       │                │
   │            │                      │ FetchWorkflowMetadata │                │
   │            │                      │  (id, WithFetchPayloads)              │
   │            │                      │──────────────────────►│                │
   │            │                      │                       │ GetInstance    │
   │            │                      │                       │───────────────►│
   │            │                      │                       │ OrchestrationMetadata
   │            │                      │                       │◄───────────────│
   │            │                      │ *WorkflowMetadata     │                │
   │            │                      │◄──────────────────────│                │
   │            │ &OrderStatus{...}    │                       │                │
   │            │◄─────────────────────│                       │                │
   │ 200 {"instance_id":"...","runtime_status":"COMPLETED","output":"\"receipt-car\""}
   │◄───────────│                      │                       │                │
```

If the sidecar returns `api.ErrInstanceNotFound`, `Get` translates it to `biz.ErrOrderNotFound` (a kratos `NotFound` error), which the HTTP transport renders as `404`.

## Try it

```bash
make dev

curl -X POST http://localhost:8000/v1/orders \
     -H 'Content-Type: application/json' \
     -d '{"item_name":"car","total_cost":15000.0}'
# => {"instance_id":"..."}

curl http://localhost:8000/v1/orders/<instance_id>
# => {"instance_id":"...","runtime_status":"COMPLETED","output":"\"receipt-car\""}
```

## Extending

- Add an external-event step with `ctx.WaitForExternalEvent("approval", timeout)` (see the order-processor sample for the high-value-order pattern).
- Replace `ProcessPaymentActivity` with a real `dapr.Client.SaveState` call against the configured state store.
