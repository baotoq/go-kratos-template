#!/usr/bin/env bash
# shellcheck shell=bash
# Shared helpers for .devcontainer/test/*.sh. Source — do not execute directly.

set -euo pipefail

log() { printf '\033[1;34m[devcontainer-test]\033[0m %s\n' "$*" >&2; }
ok()  { printf '\033[1;32m  ✓\033[0m %s\n' "$*" >&2; }
warn(){ printf '\033[1;33m  !\033[0m %s\n' "$*" >&2; }
fail(){ printf '\033[1;31m  ✗\033[0m %s\n' "$*" >&2; exit 1; }

# Repo root = parent of .devcontainer (resolved from this lib's location).
repo_root() {
    cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd
}

# Require a binary on PATH or fail with install hint.
require_cmd() {
    local cmd=$1 hint=${2:-}
    command -v "$cmd" >/dev/null 2>&1 || fail "missing dependency: $cmd${hint:+ (install: $hint)}"
}
