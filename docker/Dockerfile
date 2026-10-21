# Multi-stage build for Debian-based image
# Stage 1: Build frontend
FROM node:20-bookworm as frontend-builder

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json* ./frontend/

WORKDIR /app/frontend
RUN npm ci
COPY frontend/src ./src
COPY frontend/public ./public
COPY frontend/tsconfig.json frontend/index.html frontend/vite.config.ts ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-bookworm as backend-builder

WORKDIR /app

COPY backend/go.mod backend/go.sum* ./backend/
WORKDIR /app/backend
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 3: Create runtime image
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy frontend dist
COPY --from=frontend-builder /app/frontend/dist ./static

# Copy backend binary
COPY --from=backend-builder /app/backend/server .

# Copy runtime plugin directory from backend
COPY --chown=nobody:nogroup backend/plugin ./plugin

# Create necessary directories
RUN mkdir -p uploads config data plugin && chmod 755 uploads config data plugin

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./server"]
