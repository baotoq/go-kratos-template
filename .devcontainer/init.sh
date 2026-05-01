#!/bin/bash

# Ensures that the source mount points in devcontainer.json exist on the host.

echo "Ensuring mount points exist..."
mkdir -p "$HOME/.kube"
