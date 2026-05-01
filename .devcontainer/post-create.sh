#!/usr/bin/env bash
set -euo pipefail

# Install Go protoc plugins + Wire + ent CLI (defined in Makefile `init` target)
make init
