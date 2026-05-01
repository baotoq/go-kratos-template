#!/usr/bin/env bash
set -euo pipefail

# Install a shell hook that syncs the host kubeconfig into the container on
# every new shell: host ~/.kube is bind-mounted (read-only) at
# /usr/local/share/kube-localhost; this hook copies it to ~/.kube and rewrites
# localhost / 127.0.0.1 → host hostname so kubectl can reach the host API server.
#
# Auto-detection picks the right hostname per runtime:
#   - OrbStack:       k8s.orb.local       (cert valid for *.orb.local, NOT host.docker.internal)
#   - Docker Desktop: host.docker.internal (cert valid for host.docker.internal)
# Override with KUBE_HOST_REWRITE=<hostname> in remoteEnv if needed.

sudo tee /usr/local/share/sync-kube-config.sh > /dev/null <<'EOF'
# Sync the host kubeconfig into the container only when the host copy is newer
# than the container copy. This preserves any in-container edits a developer
# makes (e.g. `kubectl config set-context`, switching namespaces, adding extra
# contexts) until the host kubeconfig itself actually changes.
if [ "${SYNC_LOCALHOST_KUBECONFIG:-true}" = "true" ] \
        && [ -f /usr/local/share/kube-localhost/config ]; then
    src=/usr/local/share/kube-localhost/config
    dst="$HOME/.kube/config"
    if [ ! -f "$dst" ] || [ "$src" -nt "$dst" ]; then
        if [ -n "${KUBE_HOST_REWRITE:-}" ]; then
            target="$KUBE_HOST_REWRITE"
        elif getent hosts k8s.orb.local >/dev/null 2>&1; then
            target="k8s.orb.local"
        else
            target="host.docker.internal"
        fi
        mkdir -p "$HOME/.kube"
        sudo cp -r /usr/local/share/kube-localhost/. "$HOME/.kube/"
        sudo chown -R "$(id -u)" "$HOME/.kube"
        sed -i -e "s/localhost/${target}/g" "$dst"
        sed -i -e "s/127.0.0.1/${target}/g" "$dst"
    fi
fi
EOF
sudo chmod 0644 /usr/local/share/sync-kube-config.sh

for rc in "$HOME/.bashrc" "$HOME/.zshrc"; do
    if [ -f "$rc" ] && ! grep -q "sync-kube-config.sh" "$rc"; then
        echo "source /usr/local/share/sync-kube-config.sh" >> "$rc"
    fi
done
