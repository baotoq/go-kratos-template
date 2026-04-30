---
name: tilt-go-debug
description: Use when setting up or troubleshooting remote Delve debugging of Go services running under Tilt + Kubernetes from VS Code. Covers Tiltfile dlv entrypoint, Dockerfile.dev.debug, k8s debug overlay, VS Code launch.json with dlv-dap adapter, port forwards across multi-service setups, and common gotchas (api-version mismatch, dlv dap vs exec, probe interference). Triggers: "debug attach not working", "breakpoints not hitting", "delve in tilt", "vscode go remote debug", "add debug to <service>".
---

# Tilt + Delve Remote Debug for Go in VS Code

Verified working setup for remote-debugging Go services running inside Kubernetes (via Tilt) from VS Code. Binary compiled locally, synced into container, Delve runs as process wrapper.

## Architecture

```
Mac (VS Code dlv-dap) ──DAP──▶ Tilt port-forward ──▶ Delve :2345 ──▶ Go binary
```

Binary compiled on Mac with `-gcflags="all=-N -l"`, synced into container via `live_update`. Delve wraps the binary. VS Code attaches directly via DAP protocol.

## 1. Tiltfile Entrypoint

```python
config.define_bool('continue', args=False, usage='Start Delve with --continue')
dlv_continue = config.parse().get('continue', False)

dlv_flags = '--headless --listen=:2345 --accept-multiclient --only-same-user=false --log'
if dlv_continue:
    dlv_flags += ' --continue'

entrypoint_svc = ['sh', '-c', 'exec dlv exec /app/<svc> ' + dlv_flags + ' -- -conf /data/conf']
```

**Critical flags:**
- `dlv exec` — NOT `dlv dap`. `dlv dap` is for local sessions; `dlv exec --headless` is for remote attach.
- `--accept-multiclient` — allows reconnect without restarting the process.
- `--only-same-user=false` — required when container uid ≠ debugger uid.
- **NO `--api-version=2`** — forces JSON-RPC mode; breaks `dlv-dap` adapter with `"invalid character 'C'"` error.

## 2. Compile Command

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -gcflags="all=-N -l" \
  -ldflags "-X main.Version=dev" \
  -o ./dist/<svc> ./app/<svc>/cmd/server
```

- `-gcflags="all=-N -l"` — disables optimizations + inlining so breakpoints land on the correct source lines.
- Match `GOARCH` to container arch (`arm64` on Apple Silicon hosts, `amd64` on Linux/Intel).
- Do NOT use `-trimpath` — it strips absolute paths from debug info, breaking source mapping.

## 3. Dockerfile.dev.debug

Two-stage build. Stage 1 installs `dlv`; stage 2 is a slim runtime.

```dockerfile
FROM golang:<ver> AS tools
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/go-delve/delve/cmd/dlv@latest

FROM debian:stable-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates netbase \
    && rm -rf /var/lib/apt/lists/*
COPY --from=tools /go/bin/dlv  /usr/local/bin/dlv
COPY dist/<svc>               /app/<svc>
WORKDIR /app
EXPOSE 8000 9000 2345
VOLUME /data/conf
```

## 4. K8s Debug Overlay Patch

File: `deploy/k8s/overlays/debug/<svc>-debug-patch.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <svc>
spec:
  template:
    spec:
      containers:
        - name: <svc>
          ports:
            - containerPort: 8000
            - containerPort: 9000
            - containerPort: 2345
              name: dlv
          readinessProbe: null
          livenessProbe: null
          securityContext:
            capabilities:
              add: ["SYS_PTRACE"]
```

- `readinessProbe: null` + `livenessProbe: null` — probes time out while process is paused at breakpoint → pod restart loop. Disable in debug overlay.
- `SYS_PTRACE` — Delve needs ptrace capability to control the process.

## 5. Port Forwards (Multi-Service)

Container port is always `2345`. Each service gets a unique **host** port.

```python
k8s_resource('greeter', port_forwards=['8000:8000', '9000:9000', '2345:2345'], ...)
k8s_resource('catalog', port_forwards=['8001:8000', '9001:9000', '2346:2345'], ...)
# next service: 2347:2345, etc.
```

## 6. VS Code launch.json

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Attach to greeter in k8s",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "debugAdapter": "dlv-dap",
      "host": "127.0.0.1",
      "port": 2345
    },
    {
      "name": "Attach to catalog in k8s",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "debugAdapter": "dlv-dap",
      "host": "127.0.0.1",
      "port": 2346
    }
  ],
  "compounds": [
    {
      "name": "Attach to all",
      "configurations": ["Attach to greeter in k8s", "Attach to catalog in k8s"]
    }
  ]
}
```

- `debugAdapter: "dlv-dap"` (NOT `"legacy"`) — VS Code talks DAP directly to Delve. `"legacy"` spawns a local `dlv` proxy that breaks on version mismatch with the container's Delve.
- `compounds` — one click attaches to all services.

## 7. Two Run Modes

| Mode | Command | Behavior |
|------|---------|----------|
| Attach anytime | `tilt up -- --continue` | Delve starts program immediately; attach later for future requests |
| Wait-for-attach | `tilt up` | Delve halts at process start; useful for init-path breakpoints |

Makefile convention:
```makefile
dev:
	tilt up -- --continue   # run normally, debugger optional

debug:
	tilt up                 # wait for VS Code to attach before starting
```

## 8. Adding Debug to a New Service

1. Create `app/<svc>/Dockerfile.dev.debug` (copy pattern from section 3)
2. Add `deploy/k8s/overlays/debug/<svc>-debug-patch.yaml` (section 4)
3. Register patch in `deploy/k8s/overlays/debug/kustomization.yaml`
4. Add Tiltfile entrypoint + `compile_<svc>` + `k8s_resource` with unique host port
5. Add VS Code config entry with matching port

## 9. Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `error layer=rpc rpc:invalid character 'C' looking for beginning of value` | `--api-version=2` in dlv flags | Remove `--api-version=2` |
| `Expected to connect to external dlv --headless server via DAP` | Used `dlv dap` subcommand | Change to `dlv exec --headless` |
| Pod restarts during debug session | Probes time out on paused process | `readinessProbe: null`, `livenessProbe: null` in overlay |
| Breakpoint dot hollow (unverified) | Source path mismatch | Remove `-trimpath`; compile from repo root |
| Breakpoint hit but VS Code doesn't pause | VS Code window not focused; or `legacy` adapter | Use `dlv-dap`, focus VS Code window |
| Connection refused on attach | Port-forward not active | `lsof -i :<port>` to verify Tilt is forwarding |
| `permission denied` from dlv | Missing `SYS_PTRACE` | Add capability to k8s debug overlay |

## 10. Verification Checklist

```bash
lsof -i :<host_port>   # Tilt process should be LISTENing
```

- Pod logs contain: `API server listening at: [::]:2345`
- VS Code attach completes without error popup
- Breakpoint shows solid red dot (verified)
- HTTP request → VS Code pauses at breakpoint
