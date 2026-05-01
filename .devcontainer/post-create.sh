#!/usr/bin/env bash
set -euo pipefail

# Named volumes are owned by root on first mount; reclaim ~/.claude for vscode.
sudo chown -R "$(id -u):$(id -g)" "$HOME/.claude"
sudo chown -R "$(id -u):$(id -g)" "$HOME/go"

# Install Go protoc plugins + Wire + ent CLI (defined in Makefile `init` target)
#make init

# Run modular post-create steps.
bash "$(dirname "$0")/post-create/k8s.sh"
