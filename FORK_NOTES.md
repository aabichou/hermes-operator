# Fork notes — `aabichou/hermes-operator`

This fork exists to produce a self-hosted `ghcr.io/aabichou/hermes-agent`
image that actually serves the `hermes dashboard` UI when installed
from a git ref. Consumed by the `homelab-k3s` repo (`tunis` cluster,
`agents/hermes` instance).

The operator code is **mostly** unchanged from upstream
`paperclipinc/hermes-operator`. Only the `images/hermes-agent/` build
context and the `spec.semaphore` CRD integration (see Divergence below)
differ.

## Divergence from upstream

### `spec.semaphore` — Semaphore UI integration (operator code change)

Added a first-class `spec.semaphore` section to the HermesInstance CRD that auto-injects
`SEMAPHORE_URL` and `SEMAPHORE_TOKEN` env vars into agent containers and exposes a
`SemaphoreReady` status condition. Follows the same pattern as `spec.tailscale` and
`spec.profileStore`.

Files changed:
- `api/v1/hermesinstance_types.go` — `SemaphoreSpec` type + `ConditionSemaphoreReady`
- `internal/resources/semaphore.go` — `BuildSemaphoreEnv()` builder
- `internal/resources/statefulset.go` — wiring into agent container env
- `internal/controller/hermesinstance_controller.go` — `reconcileSemaphore` step
- `.github/workflows/operator-image.yaml` — CI to build + push operator image to GHCR

Usage:
```yaml
spec:
  semaphore:
    enabled: true
    url: http://semaphore.semaphore:80
    tokenSecretRef:
      name: semaphore-credentials
      key: api-token
  skills:
    - source: semaphore-ui
```

### `images/hermes-agent/pyproject.toml`

```diff
 dependencies = [
-    "hermes-agent @ git+https://github.com/NousResearch/hermes-agent@v2026.5.29.2",
+    "hermes-agent[web] @ git+https://github.com/aabichou/hermes-agent-src@fix/packages-find-hermes-cli-subpackages",
 ]
-
-[tool.uv]
-frozen = true
```

- **`[web]` extra** — pulls `fastapi==0.133.1` and `uvicorn[standard]==0.41.0`,
  required at import time by `hermes_cli.web_server`. Upstream's image
  installs these via system packages so the extra isn't needed there.
- **Fork ref** — points at
  [`aabichou/hermes-agent-src`](https://github.com/aabichou/hermes-agent-src)
  branch `fix/packages-find-hermes-cli-subpackages`. That branch patches
  one line in `pyproject.toml`'s `[tool.setuptools.packages.find]`
  `include` list to add `hermes_cli.*` and `acp_adapter.*` so wheel
  builds don't drop subpackages. Upstream's own Dockerfile sidesteps
  the bug via `uv pip install -e .` (editable). See the fork's commit
  for the full diff.
- **Removed `[tool.uv] frozen = true`** — `uv 0.11.7` rejects the
  field as unknown and prints a TOML parse warning on every container
  start; `--frozen` is already passed on the `uv sync` command line in
  the Dockerfile so behavior is unchanged.

### `images/hermes-agent/Dockerfile`

Added a `web-builder` stage between `builder` and `runtime`:

```Dockerfile
FROM node:22-bookworm-slim AS web-builder
ARG HERMES_AGENT_FORK_REPO=https://github.com/aabichou/hermes-agent-src.git
ARG HERMES_AGENT_FORK_REF=fix/packages-find-hermes-cli-subpackages
# ... apt git ca-certificates ...
RUN git clone --depth 1 --branch "${HERMES_AGENT_FORK_REF}" "${HERMES_AGENT_FORK_REPO}" hermes-agent
WORKDIR /src/hermes-agent/web
RUN npm ci --no-audit --no-fund
RUN npm run build
RUN test -f /src/hermes-agent/hermes_cli/web_dist/index.html
```

…and a COPY in the `runtime` stage that drops the assets into the venv:

```Dockerfile
ARG PYTHON_VERSION  # re-declared so the path-interpolation below works
COPY --from=web-builder --chown=hermes:hermes \
    /src/hermes-agent/hermes_cli/web_dist \
    /opt/venv/lib/python${PYTHON_VERSION}/site-packages/hermes_cli/web_dist
```

Without these, `uv sync` produces a venv where `hermes_cli/web_dist/`
exists but is empty (the wheel ships the directory but the JS toolchain
never ran), and every dashboard route returns
`{"error":"Frontend not built. Run: cd web && npm run build"}`.

### `images/hermes-agent/entrypoint.sh`

```diff
-    exec hermes-agent run "$@"
+    exec hermes dashboard \
+        --host "${HERMES_DASHBOARD_HOST:-0.0.0.0}" \
+        --port "${HERMES_DASHBOARD_PORT:-8443}" \
+        --no-open \
+        --insecure \
+        "$@"
```

The upstream `hermes-agent` CLI entrypoint runs a one-shot demo, not a
long-running service. We need a service that satisfies the
StatefulSet's `tcpSocket: 8443` readiness probe **and** serves the UI
the operator's `IngressRoute` points at — `hermes dashboard` does both
on one port.

`--insecure` opts out of the dashboard's OAuth gate; the deployment
relies on the trusted-LAN network boundary + Traefik-terminated TLS for
auth.

## Build / release

CI workflow `agent-image` is `workflow_dispatch`-only (no matrix, no
push trigger) and accepts a single `hermes_version` input that becomes
the image tag:

```bash
gh workflow run agent-image \
    --repo aabichou/hermes-operator \
    -f hermes_version=v2026.5.29.2-fixN
```

Tags use suffixes (`-fix1`, `-fix2`, …) rather than overwriting an
existing tag, because the StatefulSet has `imagePullPolicy: IfNotPresent`
and nodes will keep serving cached layers if you push the same tag
twice.

The `hermes_version` value flows into:
- the image tag in GHCR
- the `HERMES_VERSION` build-arg (`pyproject.toml`'s informational label)

It is **not** the upstream hermes-agent version anymore — that's pinned
to the fork branch in `pyproject.toml`. The `hermes_version` arg is
just a build label / image tag.

## Reproducibility caveats

- The fork dependency is pinned by **branch name**, not commit SHA, so
  `uv lock` will resolve to whatever the branch head is at lock time.
  Bump to a SHA before treating the lockfile as reproducible across
  rebuilds.
- The web build clones the same branch by name in the Dockerfile.
  Same caveat: a force-push to the fork branch would change the
  produced UI bundle without changing the image tag.
- `npm ci` honours `web/package-lock.json` from the cloned fork, so
  the JS dependency closure is locked even if the Python lockfile
  isn't.

## Upstreaming candidates

Both upstream patches are small and broadly useful — worth submitting
as PRs to the original projects:

1. **`NousResearch/hermes-agent`** — one-line `pyproject.toml` change
   adding `hermes_cli.*` to `[tool.setuptools.packages.find]`. Affects
   anyone consuming hermes-agent via `pip install` / `uv pip install`
   from git (i.e. anyone who isn't running the upstream Dockerfile's
   editable install).
2. **`paperclipinc/hermes-operator`** — the web-build stage. Their
   `images/hermes-agent/` image currently has the same "no UI" bug for
   the same reason; nobody noticed because the operator's default
   `HermesInstance` doesn't expose the dashboard via IngressRoute by
   default.
