# sandbox-api-go

Boilerplate d'API REST en Go avec authentification JWT, PostgreSQL, MinIO et WebSocket.

## Stack

- **Go 1.26** — backend
- **PostgreSQL** — base de données avec migrations automatiques
- **MinIO** — stockage d'objets S3-compatible
- **JWT** — authentification
- **Prometheus** — métriques
- **Docker / Docker Compose** — conteneurisation

## Fonctionnalités incluses

- Authentification JWT (register, login, logout)
- Gestion des utilisateurs et profils
- Board Kanban (colonnes, tâches, réordonnancement)
- Time tracking
- Notifications
- Upload/download de médias via URLs présignées MinIO
- WebSocket pour les notifications temps réel
- Métriques Prometheus sur `/metrics`
- Logs JSON structurés
- Migrations automatiques au démarrage

## Démarrage local

### Prérequis

- Docker
- Docker Compose

### 1. Configurer l'environnement

```bash
cp .env.example .env
```

Modifier les valeurs dans `.env` si nécessaire.

### 2. Lancer les services

```bash
docker compose up -d --build
```

Services disponibles :

| Service | URL |
|---------|-----|
| API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |
| pgAdmin | http://localhost:5050 |

## Déploiement en production

Le déploiement repose sur un **registry Docker privé** et **nginxproxy/nginx-proxy** pour le routing.

### Prérequis serveur

- Docker + Docker Compose
- [nginxproxy/nginx-proxy](https://github.com/nginx-proxy/nginx-proxy) en cours d'exécution
- Accès à un registry Docker privé

### 1. Configurer les variables de production

Créer `.env.prod` sur le serveur à partir de `.env.example` :

```bash
cp .env.example .env.prod
chmod 600 .env.prod
```

Variables importantes à modifier :

```env
DB_PASSWORD=mot_de_passe_fort
JWT_SECRET=cle_jwt_longue_et_aleatoire
REGISTRY_URL=registry.example.com
REGISTRY_USER=user
REGISTRY_PASSWORD=password
IMAGE_NAME=sandbox-api-go
IMAGE_TAG=latest
```

Générer des secrets forts :

```bash
openssl rand -base64 32
```

### 2. Builder et pusher l'image

Depuis la machine de développement :

```bash
# Charger les variables
source .env.prod

# Build pour AMD64
docker buildx build --platform linux/amd64 \
  -t ${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG} --load .

# Login et push
echo "$REGISTRY_PASSWORD" | docker login "$REGISTRY_URL" \
  -u "$REGISTRY_USER" --password-stdin

docker push ${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG}
```

### 3. Configurer compose.prod.yaml

Le service doit exposer les variables `VIRTUAL_HOST` et `LETSENCRYPT_HOST` pour nginx-proxy :

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
    external: true   # réseau de nginx-proxy
  internal:
```

### 4. Déployer sur le serveur

```bash
# Copier les fichiers de config
scp compose.prod.yaml .env.prod user@server:/opt/app/

# Se connecter et déployer
ssh user@server
cd /opt/app

docker login $REGISTRY_URL -u $REGISTRY_USER
docker compose -f compose.prod.yaml --env-file .env.prod pull
docker compose -f compose.prod.yaml --env-file .env.prod up -d
```

### 5. Vérifier le déploiement

```bash
docker compose -f compose.prod.yaml ps
docker compose -f compose.prod.yaml logs -f api
```

## Structure du projet

```
sandbox-api-go/
├── auth/               # JWT
├── config/             # Variables d'environnement
├── database/           # Init PostgreSQL + migrations
│   └── migrations/     # Fichiers SQL
├── errors/             # Types d'erreurs centralisés
├── handlers/           # Handlers HTTP
├── logger/             # Logs JSON structurés
├── metrics/            # Prometheus
├── middleware/         # Auth, logging, panic recovery
├── models/             # Entités métier
├── storage/            # Client MinIO
├── validation/         # Validation des inputs
├── websocket/          # WebSocket manager
├── Dockerfile
├── docker-compose.yml  # Dev
├── compose.prod.yaml   # Production
└── main.go
```

## Endpoints

### Publics

```
POST   /auth/register
POST   /auth/login
POST   /auth/logout
GET    /metrics
GET    /ws
```

### Authentifiés (JWT requis)

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

## Sécurité

Scanner les vulnérabilités de l'image :

```bash
trivy image --ignore-unfixed sandbox-api-go
```
