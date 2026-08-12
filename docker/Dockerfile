# Multi-stage build for the Debian-based image.
FROM node:22-bookworm AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-bookworm AS backend-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
# P2-65: PR builds cannot be given the signing key (untrusted), so CI passes
# PLUGIN_ALLOW_UNSIGNED=1 and the plugin bundle is built unsigned via the
# script's --allow-unsigned escape hatch. Release/push builds leave this empty:
# the key comes in as a BuildKit secret and a missing key fails the build
# (fail-closed).
ARG PLUGIN_ALLOW_UNSIGNED=

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY script/build-plugins.sh script/plugin-sources.txt /app/script/

# The plugin signing key is injected as a BuildKit secret (id=plugin_sign_key)
# and read by build-plugins.sh from /run/secrets/plugin_sign_key — never a build
# ARG, so it cannot leak into the image history. The mount is optional
# (required=false): when absent, build-plugins.sh fails-closed on a release build
# or skips signing for PR builds via --allow-unsigned.
RUN --mount=type=secret,id=plugin_sign_key \
    set -e; \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
      go build -buildvcs=false -trimpath -o /app/server ./cmd/server; \
    if [ -n "$PLUGIN_ALLOW_UNSIGNED" ]; then \
      /app/script/build-plugins.sh --allow-unsigned "$TARGETOS" "$TARGETARCH" /app/runtime-plugins; \
    else \
      /app/script/build-plugins.sh "$TARGETOS" "$TARGETARCH" /app/runtime-plugins; \
    fi; \
    test "$(find /app/runtime-plugins -type f -name manifest.json | wc -l)" -gt 0

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      passwd \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 xiaohei \
    && useradd --system --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin xiaohei

WORKDIR /app
COPY --from=frontend-builder /app/frontend/dist ./static
COPY --from=backend-builder /app/server ./server
COPY --from=backend-builder /app/runtime-plugins ./bundled-plugins
COPY docker/entrypoint.sh /usr/local/bin/xiaohei-entrypoint
COPY docker/sync-plugins.sh /usr/local/bin/xiaohei-sync-plugins

RUN mkdir -p config data uploads plugins \
    && chown -R xiaohei:xiaohei /app \
    && chmod 0755 /usr/local/bin/xiaohei-entrypoint /usr/local/bin/xiaohei-sync-plugins

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD status="$(curl --fail --silent --show-error http://localhost:8080/api/v1/install/status)" \
    && { ! printf '%s' "$status" | grep -q '"installed":true' \
      || curl --fail --silent --show-error http://localhost:8080/api/v1/site/settings >/dev/null; }

ENTRYPOINT ["/usr/local/bin/xiaohei-entrypoint"]
CMD ["./server"]
