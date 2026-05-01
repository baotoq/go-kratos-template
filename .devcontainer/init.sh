#!/usr/bin/env bash
set -euo pipefail

# Ensures that the source mount points in devcontainer.json exist on the host
# and seeds .devcontainer/.env (required by `--env-file` in runArgs) on first run.

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Ensuring mount points exist..."
mkdir -p "$HOME/.kube"
# Bind mount target — Docker fails the bind if the source doesn't exist.
touch "$HOME/.gitconfig"

if [ ! -f "$script_dir/.env" ]; then
    echo "Seeding .devcontainer/.env from .env.example..."
    cp "$script_dir/.env.example" "$script_dir/.env"
fi
