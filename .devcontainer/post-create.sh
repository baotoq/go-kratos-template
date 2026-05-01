#!/usr/bin/env bash
set -euo pipefail

# Install Go protoc plugins + Wire + ent CLI (defined in Makefile `init` target)
make init

# Run modular post-create steps.
bash "$(dirname "$0")/post-create/k8s.sh"
