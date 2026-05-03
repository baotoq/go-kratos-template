load('ext://restart_process', 'docker_build_with_restart')
load('ext://helm_resource', 'helm_resource', 'helm_repo')
allow_k8s_contexts(['docker-desktop', 'orbstack'])
docker_prune_settings(num_builds=1, keep_recent=1)
secret_settings(disable_scrub=True)
# Tilt doesn't auto-detect OrbStack as a local cluster (and the rewritten
# kubeconfig URL inside the devcontainer hides the localhost hint), so it tries
# to push built images. Use ttl.sh — Tilt's recommended ephemeral registry —
# so push/pull works without provisioning local registry infra.
# Images expire automatically (default 1h). Replace with a self-hosted
# registry if offline dev is required.
default_registry('ttl.sh')

# Usage:
#   tilt up                    Delve waits for debugger to attach
#   tilt up -- --continue      Delve starts immediately (no wait)
config.define_bool('continue', args=False, usage='Start Delve with --continue')
dlv_continue = config.parse().get('continue', False)

dlv_flags = '--headless --listen=:7000 --accept-multiclient --only-same-user=false --log'
if dlv_continue:
    dlv_flags += ' --continue'

entrypoint = ['sh', '-c', 'exec dlv exec /app/coffee ' + dlv_flags + ' -- -conf /data/conf']

compile_cmd = 'mkdir -p dist && GOOS=linux GOARCH=$(go env GOHOSTARCH) CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags "-X main.Name=coffee -X main.Version=dev" -o ./dist/coffee ./app/coffee/cmd/server'

# Compile locally on every Go source change.
# Result is synced into the running container — no full image rebuild needed.
local_resource('compile',
    cmd=compile_cmd,
    deps=['./app', './api', 'go.mod', 'go.sum'],
    labels=['build'],
)

# Helm chart tarball for redis. helm() below is evaluated at
# Tiltfile load time, so this fetch must run synchronously here — a
# local_resource would fire too late to affect the current load.
local('[ -d deploy/helm/charts ] || helm dependency update deploy/helm', quiet=True)

helm_repo('dapr-repo', 'https://dapr.github.io/helm-charts/', labels=['infra'])

# If Dapr is not yet installed via Helm, delete any pre-existing CRDs (e.g. from
# `dapr init`) so Helm can claim field ownership cleanly. No-ops when Helm already
# manages the release.
local_resource('patch-dapr-crds',
    cmd="""
    helm status dapr -n dapr-system 2>/dev/null | grep -q 'STATUS: deployed' || \
        kubectl get crds -o name 2>/dev/null | grep dapr.io | xargs -r kubectl delete --ignore-not-found
    """,
    resource_deps=['dapr-repo'],
    labels=['infra'],
)

helm_resource(
    'dapr',
    'dapr-repo/dapr',
    namespace='dapr-system',
    flags=[
        '--version=1.17.6',
        '--create-namespace',
        '--set=global.ha.enabled=false',
    ],
    resource_deps=['dapr-repo', 'patch-dapr-crds'],
    labels=['infra'],
)

# Tilt-optimised Dockerfile contains no Go toolchain — it just copies ./dist/coffee.
# only=['./dist'] means docker_build watches *only* that dir, so Go source
# changes never trigger an image rebuild; they go through compile → sync instead.
docker_build_with_restart(
    'coffee',
    '.',
    entrypoint=entrypoint,
    dockerfile='app/coffee/Dockerfile.dev.debug',
    only=['./dist'],
    live_update=[
        sync('./dist/coffee', '/app/coffee'),
    ],
)

k8s_yaml(helm(
    'deploy/helm',
    name='deps',
    namespace='coffee',
    values=['deploy/helm/values.yaml'],
))
k8s_yaml(kustomize('deploy/k8s/overlays/debug', flags=['--load-restrictor=LoadRestrictionsNone']))

k8s_resource('redis', port_forwards=['6379:6379'], labels=['infra'])

k8s_resource(
    objects=[
        'pubsub:Component:coffee',
        'secretstore:Component:coffee',
        'statestore:Component:coffee',
    ],
    new_name='dapr-components',
    resource_deps=['dapr'],
    labels=['infra'],
)

k8s_resource('coffee',
    port_forwards=['8000:8000', '9000:9000', '7000:7000'],
    resource_deps=['redis', 'compile', 'dapr', 'dapr-components'],
    labels=['app'],
)
