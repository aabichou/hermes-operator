# images/hermes-agent/Dockerfile
#
# Option A (upstream runtime): the agent image IS the official NousResearch
# hermes-agent image — an s6-overlay runtime that bundles the gateway, the
# dashboard, the OpenAI-compatible API server, a Playwright/Chromium browser,
# node, and every Python dependency — with operator metadata layered on top.
#
# HERMES_HOME=/opt/data and the upstream ENTRYPOINT (/init → main-wrapper) are
# inherited from the base image. The v0.1.19 operator runs `hermes gateway run`
# against it and decodes the workspace-seed via the image's main-wrapper
# (internal/resources/statefulset.go, docs/runtime.md). We intentionally do NOT
# rebuild the Python env ourselves (see upstream #89 — the hand-rolled uv image
# could never reach Ready under s6 supervision).
#
# Pinned by the multi-arch manifest digest of nousresearch/hermes-agent:v2026.6.19
# (hermes-agent 0.17.0). To bump:
#   docker buildx imagetools inspect nousresearch/hermes-agent:<tag> --format '{{.Manifest.Digest}}'
# then update the digest below and the HERMES_VERSION arg.
ARG HERMES_VERSION=v2026.6.19

FROM nousresearch/hermes-agent@sha256:9f367c7756ef087661a361536a89f438d57a122b958dc23d82d456b1433e6e9e

ARG HERMES_VERSION

# Operator metadata. hermes.agent/version is what the autoupdate controller
# compares against the registry tag. ENTRYPOINT + env (HERMES_HOME=/opt/data)
# are inherited from the base image — do not override.
LABEL org.opencontainers.image.title="hermes-agent" \
      org.opencontainers.image.source="https://github.com/aabichou/hermes-operator" \
      org.opencontainers.image.description="Operator-published NousResearch/hermes-agent (s6 runtime: gateway + API + dashboard + browser) for hermes-operator" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="aabichou" \
      org.opencontainers.image.base.name="docker.io/nousresearch/hermes-agent" \
      hermes.agent/version="${HERMES_VERSION}"
