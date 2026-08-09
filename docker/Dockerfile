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

WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY script/build-plugins.sh script/plugin-sources.txt /app/script/

RUN CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" \
      go build -buildvcs=false -trimpath -o /app/server ./cmd/server \
    && /app/script/build-plugins.sh "$TARGETOS" "$TARGETARCH" /app/runtime-plugins \
    && test "$(find /app/runtime-plugins -type f -name manifest.json | wc -l)" -gt 0

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=frontend-builder /app/frontend/dist ./static
COPY --from=backend-builder /app/server ./server
COPY --from=backend-builder /app/runtime-plugins ./bundled-plugins
COPY docker/entrypoint.sh /usr/local/bin/xiaohei-entrypoint

RUN mkdir -p config data uploads plugins \
    && chmod 0755 /usr/local/bin/xiaohei-entrypoint

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
  CMD status="$(curl --fail --silent --show-error http://localhost:8080/api/v1/install/status)" \
    && { ! printf '%s' "$status" | grep -q '"installed":true' \
      || curl --fail --silent --show-error http://localhost:8080/api/v1/site/settings >/dev/null; }

ENTRYPOINT ["/usr/local/bin/xiaohei-entrypoint"]
CMD ["./server"]
