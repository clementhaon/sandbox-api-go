# sandbox-api-go

Go REST API boilerplate with JWT authentication, PostgreSQL, MinIO and WebSocket.

## Stack

- **Go 1.26** — backend
- **PostgreSQL** — database with automatic migrations
- **MinIO** — S3-compatible object storage
- **JWT** — authentication
- **Prometheus** — metrics
- **Docker / Docker Compose** — containerization

## Included features

- JWT authentication (register, login, logout)
- User and profile management
- Kanban board (columns, tasks, reordering)
- Time tracking
- Notifications
- Media upload/download via MinIO presigned URLs
- WebSocket for real-time notifications
- Prometheus metrics at `/metrics`
- Structured JSON logs
- Automatic migrations on startup

## Local setup

### Prerequisites

- Docker
- Docker Compose

### 1. Configure the environment

```bash
cp .env.example .env
```

Edit the values in `.env` if needed.

### 2. Start the services

The compose file uses **profiles** to avoid duplication when part of the infrastructure already exists.

| Profile | Included services |
|---------|------------------|
| _(none)_ | API only |
| `db` | PostgreSQL, pgAdmin, MinIO, postgres-exporter |
| `monitoring` | Prometheus, Grafana, Loki, Promtail, node-exporter |

```bash
# Full standalone project (DB + monitoring included)
docker compose --profile db --profile monitoring up -d --build

# API only (DB and monitoring managed by existing infrastructure)
docker compose up -d --build

# API + DB only
docker compose --profile db up -d --build
```

Available services depending on active profiles:

| Service | URL | Profile |
|---------|-----|---------|
| API | http://localhost:8080 | always |
| MinIO Console | http://localhost:9001 | `db` |
| pgAdmin | http://localhost:5050 | `db` |
| Prometheus | http://localhost:9090 | `monitoring` |
| Grafana | http://localhost:3001 | `monitoring` |

## Production deployment

Deployment relies on a **private Docker registry** and **nginxproxy/nginx-proxy** for routing.

### Server prerequisites

- Docker + Docker Compose
- [nginxproxy/nginx-proxy](https://github.com/nginx-proxy/nginx-proxy) running
- Access to a private Docker registry

### 1. Configure production variables

Create `.env.prod` on the server from `.env.example`:

```bash
cp .env.example .env.prod
chmod 600 .env.prod
```

Important variables to update:

```env
DB_PASSWORD=strong_password
JWT_SECRET=long_random_jwt_secret
REGISTRY_URL=registry.example.com
REGISTRY_USER=user
REGISTRY_PASSWORD=password
IMAGE_NAME=sandbox-api-go
IMAGE_TAG=latest
```

Generate strong secrets:

```bash
openssl rand -base64 32
```

### 2. Build and push the image

From the development machine:

```bash
# Load variables
source .env.prod

# Build for AMD64
docker buildx build --platform linux/amd64 \
  -t ${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG} --load .

# Login and push
echo "$REGISTRY_PASSWORD" | docker login "$REGISTRY_URL" \
  -u "$REGISTRY_USER" --password-stdin

docker push ${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG}
```

### 3. Configure compose.prod.yaml

The service must expose the `VIRTUAL_HOST` and `LETSENCRYPT_HOST` variables for nginx-proxy:

```yaml
services:
  api:
    image: ${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG}
    restart: unless-stopped
    environment:
      - VIRTUAL_HOST=api.example.com
      - LETSENCRYPT_HOST=api.example.com
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - JWT_SECRET=${JWT_SECRET}
      - MINIO_ENDPOINT=${MINIO_ENDPOINT}
      - MINIO_ROOT_USER=${MINIO_ROOT_USER}
      - MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}
      - MINIO_BUCKET=${MINIO_BUCKET}
    depends_on:
      - postgres
    networks:
      - proxy
      - internal

  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - internal

volumes:
  postgres_data:

networks:
  proxy:
    external: true   # nginx-proxy network
  internal:
```

### 4. Deploy on the server

```bash
# Copy config files
scp compose.prod.yaml .env.prod user@server:/opt/app/

# Connect and deploy
ssh user@server
cd /opt/app

docker login $REGISTRY_URL -u $REGISTRY_USER
docker compose -f compose.prod.yaml --env-file .env.prod pull
docker compose -f compose.prod.yaml --env-file .env.prod up -d
```

### 5. Verify the deployment

```bash
docker compose -f compose.prod.yaml ps
docker compose -f compose.prod.yaml logs -f api
```

## Project structure

```
sandbox-api-go/
├── auth/               # JWT
├── config/             # Environment variables
├── database/           # PostgreSQL init + migrations
│   └── migrations/     # SQL files
├── errors/             # Centralized error types
├── handlers/           # HTTP handlers
├── logger/             # Structured JSON logs
├── metrics/            # Prometheus
├── middleware/         # Auth, logging, panic recovery
├── models/             # Business entities
├── storage/            # MinIO client
├── validation/         # Input validation
├── websocket/          # WebSocket manager
├── Dockerfile
├── docker-compose.yml  # Dev
├── compose.prod.yaml   # Production
└── main.go
```

## Endpoints

### Public

```
POST   /auth/register
POST   /auth/login
POST   /auth/logout
GET    /metrics
GET    /ws
```

### Authenticated (JWT required)

```
GET|PUT /profile
GET     /auth/user

GET|POST|PUT|DELETE /users/{id}
PATCH   /users/{id}/status

GET     /tasks/board
GET|POST|PUT|DELETE /tasks/{id}
PATCH   /tasks/{id}/move
PATCH   /tasks/reorder

GET|POST|PUT|DELETE /columns/{id}
PATCH   /columns/reorder

GET|POST|DELETE /time-entries/{id}

GET     /notifications
PATCH   /notifications/read
PATCH   /notifications/read-all
DELETE  /notifications/{id}

POST    /media/upload
POST    /media/confirm
GET     /media
GET|DELETE /media/{id}
GET     /media/{id}/download
```

## Monitoring

The monitoring stack includes **Prometheus**, **Grafana**, **Loki** and **Promtail**.

- Prometheus scrapes `/metrics` every 10s + preconfigured alerts (error rate, latency, API down)
- Grafana with auto-provisioned datasources and dashboards
- Loki + Promtail for log aggregation across all containers

### When forking

Update the regex in `monitoring/promtail/promtail.yml` with your project name:

```yaml
# Replace sandbox-api-go with your project name
- source_labels: ['__meta_docker_container_name']
  regex: '/sandbox-api-go-(.*)-.*'
```

## Security

Scan the image for vulnerabilities:

```bash
trivy image --ignore-unfixed sandbox-api-go
```
