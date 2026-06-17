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

# When invoked without a subcommand (k8s CMD = "serve"), run the messaging
# gateway in foreground — that's the long-running service the operator's
# StatefulSet exposes on port 8443.
if [[ "${1:-serve}" == "serve" ]]; then
    shift || true
    exec hermes gateway "$@"
fi

# Pass through to `hermes` for other subcommands (migrate, version, etc.).
# `hermes-agent` (the standalone demo runner) is intentionally not used here.
exec hermes "$@"
