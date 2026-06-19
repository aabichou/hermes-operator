#!/usr/bin/env bash
# images/hermes-agent/entrypoint.sh
#
# Wrapper around the hermes-agent CLI:
#   - Verifies ~/.hermes/config.yaml exists and is readable.
#   - Refuses to start if the StatefulSet builder failed to mount the
#     ConfigMap (a frequent symptom of operator regressions).
#   - Passes through to `hermes-agent <cmd>` so `docker run <img> migrate ...`
#     works for one-shot Jobs (the Plan 5 migration init container relies on this).
set -euo pipefail

HERMES_CONFIG="${HERMES_CONFIG:-/home/hermes/.hermes/config.yaml}"

if [[ ! -r "${HERMES_CONFIG}" ]]; then
    echo "hermes-entrypoint: config not readable at ${HERMES_CONFIG}" >&2
    echo "hermes-entrypoint: this usually means the operator failed to mount the ConfigMap subPath." >&2
    exit 78  # EX_CONFIG, matches sysexits.h
fi

# Point hermes at the operator-mounted config. The `hermes` CLI uses
# $HERMES_HOME to discover ~/.hermes/config.yaml; since we mount the file
# directly we set HERMES_HOME to its parent dir.
export HERMES_HOME="$(dirname "${HERMES_CONFIG}")"

# ── Credential bundle projection ──────────────────────────────────────
# When the HermesInstance includes a Secret mounted at /etc/hermes/creds/
# (via spec.extraVolumes), copy its entries into HOME with the modes
# OpenSSH and git's `store` credential helper require (0600 on private
# material, 0644 on public/config).
#
# The image's hermes user has its passwd-entry home set to
# /home/hermes/.hermes (= the operator's PVC mount, the only
# per-instance writable path under readOnlyRootFilesystem=true). HOME
# and HERMES_HOME both point here. ssh resolves "~" via getpwuid(),
# git/gh respect $HOME — both end up reading from the PVC.
#
# Read access on the source:
#   The Secret volume is projected with defaultMode 0440 + operator
#   fsGroup=1000, so the unprivileged hermes user (gid 1000) can read
#   it. The directory-mode constraints ssh applies to its own identity
#   files are why we copy rather than symlink.
#
# Idempotent: re-runs on every container start, overwriting in place.
# Rotation: requires a pod restart to take effect (the projected files
# refresh, but the copies in HOME don't until this hook runs again).
# Tolerant of missing keys: a partial bundle is allowed; the agent's
# skill bodies tell it to escalate on auth errors rather than improvise.
CREDS_DIR="${HERMES_CREDS_DIR:-/etc/hermes/creds}"
if [[ -d "${CREDS_DIR}" ]]; then
    install -d -m 700 "${HOME}/.ssh"
    _copy_cred() {
        local src=$1 dst=$2 mode=$3
        [[ -f "$src" ]] || return 0
        install -m "$mode" "$src" "$dst"
    }
    _copy_cred "${CREDS_DIR}/ssh-private-key"  "${HOME}/.ssh/id_ed25519"   600
    _copy_cred "${CREDS_DIR}/ssh-known-hosts"  "${HOME}/.ssh/known_hosts"  644
    _copy_cred "${CREDS_DIR}/gitconfig"        "${HOME}/.gitconfig"        644
    _copy_cred "${CREDS_DIR}/git-credentials"  "${HOME}/.git-credentials"  600
fi

# When invoked without a subcommand (k8s CMD = "serve"), run the web
# dashboard in foreground, bound to the StatefulSet's port 8443. This is
# the long-running HTTP service the operator's IngressRoute targets and
# what the readiness probe (tcpSocket: gateway) checks. The dashboard is
# always-on regardless of which messaging platforms are configured.
#
# `--insecure` opts out of the OAuth gate for trusted-LAN / behind-proxy
# deployments. Front it with a TLS-terminating IngressRoute and an
# external auth layer if you need stronger guarantees.
if [[ "${1:-serve}" == "serve" ]]; then
    shift || true
    exec hermes dashboard \
        --host "${HERMES_DASHBOARD_HOST:-0.0.0.0}" \
        --port "${HERMES_DASHBOARD_PORT:-8443}" \
        --no-open \
        --insecure \
        "$@"
fi

# Pass through to `hermes` for other subcommands (migrate, gateway, version, etc.).
# `hermes-agent` (the standalone demo runner) is intentionally not used here.
exec hermes "$@"
