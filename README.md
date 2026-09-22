# Redora 🚀

**Redora** is a lightweight, self-hosted, production-oriented web-based Redis administration and inspection tool built with Go, Fiber, SQLite, Svelte 5, Bun, TypeScript, and Tailwind CSS.

---

## 🌟 Features

- **Lightweight & Fast**: Pure Go API with Fiber engine and embedded Svelte 5 frontend.
- **Connection Management**: Multi-connection manager with encrypted credentials (AES-256 GCM) in SQLite.
- **Safe Key Browser**: Cursor-based key scanning (`SCAN`), never blocking Redis with `KEYS *`.
- **Value Viewers**: Specialized editors and viewers for String, Hash, List, Set, Sorted Set, and Streams.
- **Realtime Dashboard**: Server metrics, memory footprint, active clients, and keyspace statistics.
- **Interactive Console**: Command autocomplete, execution timing, result formatting, and safety checks on destructive commands (`FLUSHALL`, `FLUSHDB`, `CONFIG`).
- **Container Ready**: Minimal multi-stage Docker and Podman deployment support.

---

## 🏗️ Architecture

```
Browser (Svelte 5 + Tailwind CSS)
            │
            ▼
     Go HTTP API (Fiber)
            │
  ┌─────────┴─────────┐
  ▼                   ▼
SQLite         Redis Connections
(Encrypted Config) (go-redis/v9 Pool)
```

- SQLite is used solely for metadata and connection settings (never stores Redis key data).
- Redis credentials are encrypted using application secrets supplied via environment variables (`ENCRYPTION_KEY`).

---

## ⚙️ Environment Variables

| Variable         | Description                                      | Default                                  |
| :--------------- | :----------------------------------------------- | :--------------------------------------- |
| `PORT`           | Web server listening port                        | `8080`                                   |
| `DB_PATH`        | Path to SQLite database file                     | `/data/redora.db`                        |
| `ENCRYPTION_KEY` | Secret key used for credential encryption        | `default-32-byte-secret-key-change-me!!` |
| `LOG_LEVEL`      | Logging level (`debug`, `info`, `warn`, `error`) | `info`                                   |

---

## 🐳 Deployment

### Using Docker Compose

`docker-compose.yml`:

```yaml
services:
  redora:
    image: ghcr.io/teddys48/redora:latest
    container_name: redora
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - DB_PATH=/data/redora.db
      - ENCRYPTION_KEY=super-secret-production-encryption-key-32bytes
      - LOG_LEVEL=info
    volumes:
      - redora-data:/data
    restart: unless-stopped

volumes:
  redora-data:
    driver: local
```

Run container:

```bash
docker compose up -d
```

### Using Podman Compose

```bash
podman compose up -d
```

### Manual Docker Run

```bash
docker run -d \
  --name redora \
  -p 8080:8080 \
  -v redora-data:/data \
  -e ENCRYPTION_KEY="super-secret-production-encryption-key-32bytes" \
  ghcr.io/teddys48/redora:latest
```

---

## 💻 Local Development Setup

### Requirements

- Go 1.25+
- Bun 1.2+

### 1. Run Backend

```bash
cd backend
go run cmd/server/main.go
```

### 2. Run Frontend Dev Server

```bash
cd frontend
bun install
bun run dev
```

Visit `http://localhost:3000` in your browser. API requests are automatically proxied to `http://localhost:8080`.

---

## 🔌 API Overview

- `GET /api/health` - Server health and version status.
- `GET /api/connections` - List configured Redis connections.
- `POST /api/connections` - Add new connection metadata.
- `POST /api/connections/:id/test` - Test connection reachability.

---

## 🔒 Security Considerations

- Passwords are encrypted before persisting to SQLite.
- Redis instances are never directly exposed to client-side JavaScript.
- Inputs are sanitized and bounded to prevent excessive memory consumption.
