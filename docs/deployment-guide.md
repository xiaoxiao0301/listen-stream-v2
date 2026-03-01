# Listen Stream v2 — Deployment Guide

## Platform Selection Analysis

| Platform | Verdict | Reason |
|---|---|---|
| **Local Docker Compose** | ✅ Best for dev | Full stack, one command |
| **VPS (any provider)** | ✅ Recommended for prod | Full Docker support |
| **Serv00** | ⚠️ Partial only | FreeBSD shared hosting — no Docker, no root, Go binary only |
| **Hugging Face Spaces** | ❌ Not suitable | Designed for ML demos; no persistent storage, no Redis, no custom TCP ports |
| **Vercel** | ❌ Not suitable | Serverless only — no WebSockets, no long-lived connections, no stateful data |

**Recommendation:** Use Docker Compose for local development and a VPS (e.g., DigitalOcean, Hetzner, Vultr, Contabo) for production. Serv00 is documented below as a fallback for budget deployments.

---

## Option 1: Local Development (Docker Compose)

### Prerequisites
- Docker Desktop ≥ 24 / Docker Engine ≥ 24
- Docker Compose v2 (`docker compose` not `docker-compose`)
- Go 1.22+
- `make`

### Steps

```bash
# 1. Clone and enter the project
git clone <repo>
cd test

# 2. Copy environment file and edit secrets
cp infra/.env.example infra/.env
# Edit infra/.env — change passwords, add Slack webhook, SMTP creds etc.

# 3. Start the full infrastructure (Consul + OTel + Prometheus + ELK)
make infra-up
# Wait ~60 seconds for Elasticsearch to initialise

# 4. Initialise Consul KV (business config)
make infra-consul-reinit

# 5. Run database migrations for auth-svc
cd server/services/auth-svc
go run cmd/main.go migrate up
# or: make migrate-up DATABASE_URL="postgresql://listen_stream:listen_stream_pass_2026@localhost:5432/listen_stream?sslmode=disable"

# 6. Run services locally (without Docker)
#    Each service in a separate terminal:
cd server/services/auth-svc  && go run cmd/main.go
cd server/services/proxy-svc && go run cmd/main.go
# etc.
```

### Service UI endpoints after `make infra-up`

| Service | URL | Credentials |
|---|---|---|
| Consul UI | http://localhost:8500 | none (ACL disabled in dev) |
| Jaeger UI | http://localhost:16686 | none |
| Prometheus | http://localhost:9090 | none |
| Grafana | http://localhost:3000 | admin / `GF_SECURITY_ADMIN_PASSWORD` from .env |
| AlertManager | http://localhost:9093 | none |
| Kibana | http://localhost:5601 | none |
| Elasticsearch | http://localhost:9200 | none |

---

## Option 2: VPS / Cloud Server (Recommended for Production)

### Prerequisites
- Ubuntu 22.04 / Debian 12 VPS (≥ 2 vCPU, 4 GB RAM for full stack; 8 GB recommended)
- Root or sudo access
- A domain name (optional but recommended for TLS)

### Step 1 — Install Docker

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker
docker compose version   # must be v2
```

### Step 2 — Copy project to server

```bash
# On your local machine:
rsync -avz --exclude='.git' /Users/aji/test/ user@your-server:/opt/listen-stream/
# or use git:
git clone <repo> /opt/listen-stream
```

### Step 3 — Configure environment

```bash
cd /opt/listen-stream
cp infra/.env.example infra/.env
vim infra/.env
```

Critical changes for production (in `.env`):
```
ENVIRONMENT=production
POSTGRES_PASSWORD=<strong-random-password>
REDIS_PASSWORD=<strong-random-password>
CONSUL_TOKEN=<uuid>
GF_SECURITY_ADMIN_PASSWORD=<strong-password>
SLACK_WEBHOOK_URL=https://hooks.slack.com/...
SMTP_FROM=alerts@yourdomain.com
SMTP_HOST=smtp.yourdomain.com
SMTP_PORT=587
SMTP_USER=...
SMTP_PASSWORD=...
```

### Step 4 — Start infrastructure

```bash
cd /opt/listen-stream
make infra-up

# Wait for Elasticsearch (takes ~2 minutes on first start):
docker compose -f infra/docker-compose.yml logs -f elasticsearch 2>&1 | grep -m1 "started"
```

### Step 5 — Initialise data stores

```bash
# Consul KV (business configuration)
make infra-consul-reinit

# Elasticsearch ILM + index templates
docker exec listen-stream-elasticsearch-init /init-elasticsearch.sh \
  || docker run --rm --network listen-stream-backend \
       -v $(pwd)/infra/elasticsearch/init-elasticsearch.sh:/init.sh \
       curlimages/curl:latest /bin/sh /init.sh
```

### Step 6 — Build and start Go services

```bash
# Build all services
make build   # produces binaries in server/services/*/bin/

# Start Go services via Docker Compose
make services-up

# Or start everything at once:
make up
```

### Step 7 — Run database migrations

```bash
# Get the Postgres container IP or port-forward
make migrate-up DATABASE_URL="postgresql://listen_stream:${POSTGRES_PASSWORD}@localhost:5432/listen_stream?sslmode=disable"
```

### Step 8 — (Optional) Reverse proxy with Caddy

```bash
apt install -y caddy

cat > /etc/caddy/Caddyfile << 'EOF'
api.yourdomain.com {
    reverse_proxy localhost:8002    # proxy-svc
}

admin.yourdomain.com {
    reverse_proxy localhost:8005    # admin-svc
}

grafana.yourdomain.com {
    reverse_proxy localhost:3000
}

consul.yourdomain.com {
    reverse_proxy localhost:8500
}
EOF

systemctl reload caddy
```

### Verify deployment

```bash
make infra-status          # all containers running
make infra-consul-status   # 3-node cluster, all alive
make infra-es-status       # { "status": "green" }
curl http://localhost:8002/health   # {"status":"ok"}
```

---

## Option 3: Serv00 (Budget Shared Hosting, No Docker)

Serv00 is a FreeBSD-based shared hosting platform. It supports:
- Custom compiled Go binaries
- Supervised processes via `supervisord`
- MySQL/PostgreSQL (managed, single instance)
- Redis (single instance, shared)

It does **NOT** support: Docker, root access, custom network namespaces, or Consul clustering.

### What you can deploy on Serv00

| Component | Status |
|---|---|
| auth-svc | ✅ Go binary |
| proxy-svc | ✅ Go binary |
| user-svc | ✅ Go binary |
| sync-svc | ✅ Go binary (WebSockets work) |
| admin-svc | ✅ Go binary |
| PostgreSQL | ✅ Use Serv00's managed PostgreSQL |
| Redis | ✅ Use Serv00's managed Redis |
| Consul | ❌ Use file-based config instead |
| Jaeger / OTel | ❌ Disable telemetry |
| Prometheus / Grafana | ❌ Not available |
| Elasticsearch / Kibana | ❌ Not available |

### Steps

#### 1. Create Serv00 account and databases

In the Serv00 panel:
- Create a PostgreSQL database: `listen_stream`
- Enable Redis (note the host/port/password)
- Note your SSH credentials

#### 2. Cross-compile Go binaries on your Mac

```bash
# Serv00 is FreeBSD amd64
export GOOS=freebsd GOARCH=amd64

cd server/services/auth-svc
go build -o bin/auth-svc-freebsd ./cmd

cd ../proxy-svc
go build -o bin/proxy-svc-freebsd ./cmd

# Repeat for other services
```

#### 3. Upload binaries and config

```bash
scp server/services/auth-svc/bin/auth-svc-freebsd \
    docs/config-example-local.yaml \
    userXXXXX@s1.serv00.com:~/listen-stream/auth-svc/
```

#### 4. Create a local config file (no Consul)

On Serv00, create `~/listen-stream/auth-svc/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8001
  grpc_port: 9001

database:
  host: "localhost"
  port: 5432
  user: "serv00_user"
  password: "your-password"
  dbname: "listen_stream"
  sslmode: require

redis:
  host: "localhost"
  port: 6379
  password: "your-redis-password"

jwt:
  secret: "your-very-long-random-secret"
  issuer: "listen-stream"

telemetry:
  enabled: false   # no OTel on Serv00

consul:
  enabled: false   # use file-based config only
```

#### 5. Run migrations

```bash
./auth-svc-freebsd migrate up
```

#### 6. Set up supervisord

```ini
# ~/etc/supervisor/supervisord.conf or ask Serv00 support
[program:auth-svc]
command=/home/userXXXXX/listen-stream/auth-svc/auth-svc-freebsd serve
directory=/home/userXXXXX/listen-stream/auth-svc
autostart=true
autorestart=true
environment=CONFIG_FILE="/home/userXXXXX/listen-stream/auth-svc/config.yaml"
stdout_logfile=/home/userXXXXX/logs/auth-svc.log
stderr_logfile=/home/userXXXXX/logs/auth-svc-err.log
```

#### 7. Port forwarding via SSH tunnel (dev/test only)

Serv00 only exposes specific ports. Use their "Open Port" feature in the panel, or SSH tunnel:

```bash
ssh -L 8001:localhost:8001 userXXXXX@s1.serv00.com
```

### Serv00 limitations summary

- No Docker → no Consul cluster, no monitoring stack
- Single PostgreSQL instance → no connection pooling via PgBouncer
- Shared Redis → no Redis Cluster
- Limited ports → proxy via Serv00's built-in HTTP proxy
- **Suitable for:** demo/testing of individual services
- **Not suitable for:** production workloads or the full microservices stack

---

## Common Operations

```bash
# View service logs
make infra-logs
docker compose -f infra/docker-compose.yml logs -f auth-svc

# Restart a service
docker compose -f infra/docker-compose.yml -f infra/docker-compose.services.yml restart auth-svc

# Scale a service
docker compose -f infra/docker-compose.yml -f infra/docker-compose.services.yml up -d --scale proxy-svc=3

# Database backup
docker exec listen-stream-postgres pg_dump -U listen_stream listen_stream | gzip > backup_$(date +%Y%m%d).sql.gz

# Update a service (rebuild + redeploy)
docker compose -f infra/docker-compose.services.yml build auth-svc
docker compose -f infra/docker-compose.yml -f infra/docker-compose.services.yml up -d auth-svc

# Full teardown (keeps data volumes)
make down

# Full teardown + delete all data (DESTRUCTIVE)
make infra-clean
```

---

## go mod tidy (required after OTel dependency additions)

The `server/shared/go.mod` was updated with new OTel packages. Before building, run:

```bash
cd server/shared
go mod tidy
go mod download

# Propagate to services
cd ../services/auth-svc  && go mod tidy
cd ../proxy-svc          && go mod tidy
# etc.
```
