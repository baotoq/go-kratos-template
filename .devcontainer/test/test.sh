#!/usr/bin/env bash
# In-container assertions. Designed to run *inside* the devcontainer.
#
# Two callers:
#   1. .../smoke.sh           local: invoked via `devcontainer exec`.
#   2. devcontainers/ci@v0.3  CI:    invoked via the action's `runCmd`.
#
# Both share the same assertion list, so a CI failure reproduces locally with
# `make test-devcontainer-smoke`.

set -euo pipefail

# shellcheck source=.devcontainer/test/lib.sh
. "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

FAILS=0
assert() {
    local label=$1 cmd=$2 out
    if out=$(bash -lc "$cmd" 2>&1); then
        ok "$label"
    else
        warn "FAIL: $label"
        warn "  cmd: $cmd"
        printf '%s\n' "$out" | sed 's/^/      /' >&2
        FAILS=$((FAILS + 1))
    fi
}

log "tool presence"
assert "go"      "go version"
assert "docker"  "docker version --format '{{.Client.Version}}'"
assert "gh"      "gh --version"
assert "kubectl" "kubectl version --client=true -o yaml"
assert "helm"    "helm version --short"
assert "dapr"    "dapr --version"
assert "protoc"  "protoc --version"
assert "tilt"    "tilt version"
assert "node"    "node --version"
assert "npm"     "npm --version"
assert "claude"  "claude --version"

# Single-quoted commands intentional: $HOME expands inside the runtime shell,
# not whichever shell sourced this file.
# shellcheck disable=SC2016
{
log "post-create effects"
assert "claude.json is a symlink"   'test -L "$HOME/.claude.json"'
assert "kube sync hook installed"   'test -f /usr/local/share/sync-kube-config.sh'
assert "/go writable by vscode"     'test -w /go'
assert "go-build cache writable"    'test -w "$HOME/.cache/go-build"'
}

[ "$FAILS" -eq 0 ] || fail "$FAILS in-container check(s) failed"
log "in-container checks passed"
