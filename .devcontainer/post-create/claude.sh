#!/usr/bin/env bash
set -euo pipefail

# Persist Claude Code state across devcontainer rebuilds.
#
# The named volume at ~/.claude (declared in devcontainer.json) covers
# .credentials.json and session/project data, but ~/.claude.json — a sibling
# at the home root that holds onboarding state, project-trust flags, and MCP
# config — is *outside* the volume and gets recreated empty on every rebuild.
# When that happens, Claude Code replays its welcome flow, which looks like a
# re-auth prompt. Relocate the file into the volume and symlink it back.

# Named volumes are owned by root on first mount; reclaim for the current user.
sudo chown -R "$(id -u):$(id -g)" "$HOME/.claude"

if [ ! -L "$HOME/.claude.json" ]; then
    if [ -f "$HOME/.claude.json" ] && [ ! -f "$HOME/.claude/.claude.json" ]; then
        mv "$HOME/.claude.json" "$HOME/.claude/.claude.json"
    fi
    [ -e "$HOME/.claude/.claude.json" ] || touch "$HOME/.claude/.claude.json"
    ln -sf "$HOME/.claude/.claude.json" "$HOME/.claude.json"
fi
