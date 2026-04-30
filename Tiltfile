load('ext://restart_process', 'docker_build_with_restart')
load('ext://helm_resource', 'helm_resource', 'helm_repo')
allow_k8s_contexts(['docker-desktop', 'orbstack'])
docker_prune_settings(num_builds=1, keep_recent=1)

# Usage:
#   tilt up                    Delve waits for debugger to attach
#   tilt up -- --continue      Delve starts immediately (no wait)
config.define_bool('continue', args=False, usage='Start Delve with --continue')
dlv_continue = config.parse().get('continue', False)

dlv_flags = '--headless --listen=:2345 --api-version=2 --accept-multiclient --only-same-user=false --log'
if dlv_continue:
    dlv_flags += ' --continue'

entrypoint = ['sh', '-c', 'exec dlv exec /app/greeter ' + dlv_flags + ' -- -conf /data/conf']

compile_cmd = 'mkdir -p dist && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -gcflags="all=-N -l" -ldflags "-X main.Version=dev" -o ./dist/greeter ./app/greeter/cmd/server'

# Compile locally on every Go source change.
# Result is synced into the running container — no full image rebuild needed.
local_resource('compile',
    cmd=compile_cmd,
    deps=['./app', './api', 'go.mod', 'go.sum'],
    labels=['build'],
)

# Helm chart tarballs for postgres/redis. helm() below is evaluated at
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
        kubectl get crds -o name 2>/dev/null | grep dapr.io | xargs kubectl delete --ignore-not-found
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

# Tilt-optimised Dockerfile contains no Go toolchain — it just copies ./dist/greeter.
# only=['./dist'] means docker_build watches *only* that dir, so Go source
# changes never trigger an image rebuild; they go through compile → sync instead.
docker_build_with_restart(
    'greeter',
    '.',
    entrypoint=entrypoint,
    dockerfile='app/greeter/Dockerfile.dev.debug',
    only=['./dist'],
    live_update=[
        sync('./dist/greeter', '/app/greeter'),
    ],
)

k8s_yaml(helm(
    'deploy/helm',
    name='deps',
    namespace='greeter',
    values=['deploy/helm/values.yaml'],
))
k8s_yaml(kustomize('deploy/k8s/overlays/debug', flags=['--load-restrictor=LoadRestrictionsNone']))

k8s_resource('postgres', port_forwards=['5432:5432'], labels=['infra'])
k8s_resource('redis',    port_forwards=['6379:6379'], labels=['infra'])
k8s_resource('pgadmin',  port_forwards=['5050:80'],   labels=['infra'], resource_deps=['postgres'])

k8s_resource(
    objects=[
        'pubsub:Component:greeter',
        'secretstore:Component:greeter',
    ],
    new_name='dapr-components',
    resource_deps=['dapr'],
    labels=['infra'],
)

k8s_resource('greeter',
    port_forwards=['8000:8000', '9000:9000', '2345:2345'],
    resource_deps=['postgres', 'redis', 'compile', 'dapr', 'dapr-components'],
    labels=['app'],
)
