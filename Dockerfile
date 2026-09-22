# ==========================================
# Stage 1: Build Svelte Frontend Assets
# ==========================================
FROM oven/bun:alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install --frozen-lockfile || bun install

COPY frontend/ ./
RUN bun run build

# ==========================================
# Stage 2: Build Go Backend Binary
# ==========================================
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app/backend

# Install git/certificates if needed
RUN apk add --no-cache git ca-certificates

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copy compiled frontend dist into backend embed directory
COPY --from=frontend-builder /app/frontend/dist ./cmd/server/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /redora ./cmd/server/main.go

# ==========================================
# Stage 3: Minimal Production Runtime Image
# ==========================================
FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates tzdata sqlite

# Non-root user setup
RUN addgroup -S redora && adduser -S redora -G redora

WORKDIR /app
RUN mkdir -p /data && chown -R redora:redora /app /data

COPY --from=backend-builder /redora /app/redora

USER redora

ENV PORT=8080 \
    DB_PATH=/data/redora.db \
    LOG_LEVEL=info

EXPOSE 8080

VOLUME ["/data"]

ENTRYPOINT ["/app/redora"]
