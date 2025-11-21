# 🚀 API REST Go avec Authentification et PostgreSQL

Une API REST complète en Go avec authentification JWT et base de données PostgreSQL, entièrement dockerisée.

## ✨ Fonctionnalités

- 🔐 **Authentification JWT** : Inscription, connexion et protection des routes
- 📋 **Gestion des tâches** : CRUD complet avec association utilisateur
- 📁 **Gestion des médias** : Upload et accès sécurisés aux fichiers via MinIO avec URLs présignées
- 🗄️ **Base de données PostgreSQL** : Persistance des données
- 🐳 **Dockerisation complète** : API, base de données, MinIO et pgAdmin
- 🔒 **Sécurité** : Mots de passe hashés avec bcrypt et URLs présignées pour les fichiers
- 🚀 **Performance** : Optimisé avec des index de base de données

## 🛠️ Technologies utilisées

- **Backend** : Go 1.21
- **Base de données** : PostgreSQL 15
- **Stockage d'objets** : MinIO (S3-compatible)
- **Authentification** : JWT (JSON Web Tokens)
- **Hachage** : bcrypt
- **Conteneurisation** : Docker & Docker Compose
- **Interface DB** : pgAdmin 4

## 🚀 Démarrage rapide avec Docker

### Prérequis
- Docker
- Docker Compose

### 1. Cloner le projet
```bash
git clone <votre-repo>
cd sandbox-api-go
```

### 2. Lancer l'application
```bash
docker-compose up --build
```

### 3. Accéder aux services
- **API Go** : http://localhost:8080
- **PostgreSQL** : localhost:5432
- **MinIO** : http://localhost:9000
  - Console MinIO : http://localhost:9001
  - User : minioadmin
  - Password : minioadmin123
- **pgAdmin** : http://localhost:5050
  - Email : admin@example.com
  - Mot de passe : admin123

## 📋 Endpoints disponibles

### Authentification (publique)
- `POST /auth/register` - S'inscrire
- `POST /auth/login` - Se connecter

### Tâches (authentification requise)
- `GET /tasks` - Lister vos tâches
- `POST /tasks` - Créer une tâche
- `GET /tasks/{id}` - Obtenir une tâche
- `PUT /tasks/{id}` - Mettre à jour une tâche
- `DELETE /tasks/{id}` - Supprimer une tâche

### Profil utilisateur (authentification requise)
- `GET /profile` - Obtenir le profil utilisateur
- `PUT /profile` - Mettre à jour le profil utilisateur

### Médias (authentification requise)
- `POST /media/upload` - Obtenir une URL présignée pour uploader un fichier
- `POST /media/confirm` - Confirmer l'upload et enregistrer le média
- `GET /media` - Lister tous vos médias
- `GET /media/{id}` - Obtenir un média par ID
- `GET /media/{id}/download` - Obtenir une URL présignée pour télécharger un fichier
- `DELETE /media/{id}` - Supprimer un média

## 🔐 Utilisation de l'API

### 1. Inscription d'un utilisateur
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nouveau_user",
    "email": "user@example.com",
    "password": "motdepasse123"
  }'
```

### 2. Connexion
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nouveau_user",
    "password": "motdepasse123"
  }'
```

### 3. Utilisation du token
```bash
curl -X GET http://localhost:8080/tasks \
  -H "Authorization: Bearer <votre_token_jwt>"
```

### 4. Upload d'un fichier

#### Étape 1 : Obtenir une URL présignée pour l'upload
```bash
curl -X POST http://localhost:8080/media/upload \
  -H "Authorization: Bearer <votre_token_jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "mon-image.jpg",
    "mime_type": "image/jpeg"
  }'
```

Réponse :
```json
{
  "upload_url": "http://minio:9000/user-uploads/users/1/mon-image-abc123.jpg?...",
  "object_key": "users/1/mon-image-abc123.jpg",
  "expires_in": 3600
}
```

#### Étape 2 : Uploader le fichier vers MinIO
```bash
curl -X PUT "<upload_url>" \
  -H "Content-Type: image/jpeg" \
  --data-binary @mon-image.jpg
```

#### Étape 3 : Confirmer l'upload et enregistrer dans la base de données
```bash
curl -X POST http://localhost:8080/media/confirm \
  -H "Authorization: Bearer <votre_token_jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "object_key": "users/1/mon-image-abc123.jpg",
    "original_filename": "mon-image.jpg",
    "mime_type": "image/jpeg"
  }'
```

### 5. Télécharger un fichier

#### Obtenir une URL présignée pour télécharger
```bash
curl -X GET http://localhost:8080/media/1/download \
  -H "Authorization: Bearer <votre_token_jwt>"
```

Réponse :
```json
{
  "download_url": "http://minio:9000/user-uploads/users/1/mon-image-abc123.jpg?...",
  "expires_in": 3600
}
```

Ensuite, utilisez l'URL pour télécharger le fichier :
```bash
curl -o fichier-telecharge.jpg "<download_url>"
```

### 6. Lister vos médias
```bash
curl -X GET http://localhost:8080/media \
  -H "Authorization: Bearer <votre_token_jwt>"
```

### 7. Supprimer un média
```bash
curl -X DELETE http://localhost:8080/media/1 \
  -H "Authorization: Bearer <votre_token_jwt>"
```

## 🗄️ Base de données

### Structure des tables

#### Table `users`
- `id` : Identifiant unique (SERIAL)
- `username` : Nom d'utilisateur (UNIQUE)
- `email` : Adresse email (UNIQUE)
- `password` : Mot de passe hashé
- `created_at` : Date de création
- `updated_at` : Date de mise à jour

#### Table `tasks`
- `id` : Identifiant unique (SERIAL)
- `title` : Titre de la tâche
- `description` : Description de la tâche
- `completed` : Statut de complétion
- `user_id` : Référence vers l'utilisateur
- `created_at` : Date de création
- `updated_at` : Date de mise à jour

#### Table `media`
- `id` : Identifiant unique (SERIAL)
- `user_id` : Référence vers l'utilisateur
- `object_key` : Clé de l'objet dans MinIO
- `bucket_name` : Nom du bucket (par défaut: user-uploads)
- `original_filename` : Nom original du fichier
- `file_size` : Taille du fichier en octets
- `mime_type` : Type MIME du fichier
- `created_at` : Date de création
- `updated_at` : Date de mise à jour

### Utilisateur de test
- **Username** : admin
- **Email** : admin@example.com
- **Mot de passe** : password123

## 🐳 Commandes Docker utiles

### Démarrer l'application
```bash
docker-compose up -d
```

### Voir les logs
```bash
docker-compose logs -f api
```

### Arrêter l'application
```bash
docker-compose down
```

### Redémarrer un service
```bash
docker-compose restart api
```

### Supprimer les volumes (attention : supprime les données)
```bash
docker-compose down -v
```

## 🔧 Configuration

### Variables d'environnement
Les variables d'environnement peuvent être modifiées dans le `docker-compose.yml` :

```yaml
environment:
  - DB_HOST=postgres
  - DB_PORT=5432
  - DB_USER=postgres
  - DB_PASSWORD=postgres123
  - DB_NAME=sandbox_api
  - DB_SSLMODE=disable
```

### Ports
- **8080** : API Go
- **5432** : PostgreSQL
- **9000** : MinIO API
- **9001** : MinIO Console
- **5050** : pgAdmin

## 🚀 Déploiement en production

### 🖥️ Prérequis serveur

- **Système d'exploitation** : Ubuntu 20.04+ / CentOS 8+ / Debian 11+
- **RAM** : Minimum 2GB (recommandé 4GB+)
- **Stockage** : Minimum 20GB (recommandé 50GB+)
- **Accès** : SSH avec privilèges sudo

### 📦 Installation sur le serveur

#### 1. Connexion et mise à jour
```bash
# Se connecter au serveur
ssh utilisateur@votre-serveur.com

# Mettre à jour le système
sudo apt update && sudo apt upgrade -y

# Installer les packages essentiels
sudo apt install -y curl wget git unzip software-properties-common
```

#### 2. Installation de Docker
```bash
# Installer Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Ajouter l'utilisateur au groupe docker
sudo usermod -aG docker $USER

# Redémarrer la session SSH ou exécuter
newgrp docker

# Vérifier l'installation
docker --version
```

#### 3. Installation de Docker Compose
```bash
# Installer Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# Rendre le fichier exécutable
sudo chmod +x /usr/local/bin/docker-compose

# Vérifier l'installation
docker-compose --version
```

#### 4. Configuration du pare-feu
```bash
# Installer UFW si pas présent
sudo apt install ufw -y

# Autoriser SSH
sudo ufw allow ssh

# Autoriser les ports de l'application
sudo ufw allow 80/tcp    # HTTP (si vous utilisez un reverse proxy)
sudo ufw allow 443/tcp   # HTTPS (si vous utilisez un reverse proxy)
sudo ufw allow 8080/tcp  # API Go (optionnel, pour tests)

# Activer le pare-feu
sudo ufw enable

# Vérifier le statut
sudo ufw status
```

### 🚀 Déploiement de l'application

#### 1. Cloner le projet
```bash
# Créer un répertoire pour l'application
mkdir -p /opt/apps
cd /opt/apps

# Cloner votre projet
git clone <votre-repo-git> sandbox-api-go
cd sandbox-api-go
```

#### 2. Configuration de production
```bash
# Créer un fichier d'environnement de production
cat > .env.prod << EOF
# Configuration de la base de données
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<mot_de_passe_fort_et_unique>
DB_NAME=sandbox_api
DB_SSLMODE=disable

# Configuration de l'API
API_PORT=8080
JWT_SECRET=<clé_jwt_secrète_et_longue>

# Configuration PostgreSQL
POSTGRES_DB=sandbox_api
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<mot_de_passe_fort_et_unique>
EOF

# Rendre le fichier sécurisé
chmod 600 .env.prod
```

#### 3. Créer docker-compose.prod.yml
```bash
# Créer une version de production du docker-compose
cat > docker-compose.prod.yml << 'EOF'
version: '3.8'

services:
  # API Go
  api:
    build: .
    restart: unless-stopped
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=${POSTGRES_PASSWORD}
      - DB_NAME=sandbox_api
      - DB_SSLMODE=disable
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - postgres
    networks:
      - app-network
    volumes:
      - ./logs:/app/logs
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  # Base de données PostgreSQL
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
      - ./backups:/backups
    networks:
      - app-network
    command: >
      postgres
      -c shared_preload_libraries=pg_stat_statements
      -c pg_stat_statements.track=all
      -c max_connections=100
      -c shared_buffers=256MB
      -c effective_cache_size=1GB

volumes:
  postgres_data:
    driver: local

networks:
  app-network:
    driver: bridge
EOF
```

#### 4. Lancer l'application
```bash
# Charger les variables d'environnement
source .env.prod

# Construire et démarrer les services
docker-compose -f docker-compose.prod.yml up -d --build

# Vérifier le statut
docker-compose -f docker-compose.prod.yml ps

# Voir les logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 🌐 Configuration du domaine et HTTPS

#### 1. Installation de Nginx (reverse proxy)
```bash
# Installer Nginx
sudo apt install nginx -y

# Créer la configuration du site
sudo nano /etc/nginx/sites-available/sandbox-api
```

#### 2. Configuration Nginx
```nginx
server {
    listen 80;
    server_name votre-domaine.com www.votre-domaine.com;

    # Redirection vers HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name votre-domaine.com www.votre-domaine.com;

    # Certificats SSL (à configurer avec Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/votre-domaine.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/votre-domaine.com/privkey.pem;

    # Configuration SSL sécurisée
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # Headers de sécurité
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";

    # Proxy vers l'API Go
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;
}
```

#### 3. Activer le site
```bash
# Créer un lien symbolique
sudo ln -s /etc/nginx/sites-available/sandbox-api /etc/nginx/sites-enabled/

# Tester la configuration
sudo nginx -t

# Redémarrer Nginx
sudo systemctl restart nginx

# Activer Nginx au démarrage
sudo systemctl enable nginx
```

#### 4. Installation de Let's Encrypt (HTTPS gratuit)
```bash
# Installer Certbot
sudo apt install certbot python3-certbot-nginx -y

# Obtenir un certificat SSL
sudo certbot --nginx -d votre-domaine.com -d www.votre-domaine.com

# Renouvellement automatique
sudo crontab -e
# Ajouter cette ligne :
# 0 12 * * * /usr/bin/certbot renew --quiet
```

### 🔒 Sécurisation de la production

#### 1. Mots de passe forts
```bash
# Générer des mots de passe sécurisés
openssl rand -base64 32
openssl rand -base64 32
```

#### 2. Configuration PostgreSQL sécurisée
```bash
# Modifier le docker-compose pour limiter l'accès
# Ajouter dans le service postgres :
    ports:
      - "127.0.0.1:5432:5432"  # Seulement localhost
```

#### 3. Sauvegardes automatiques
```bash
# Créer un script de sauvegarde
cat > backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/opt/apps/sandbox-api-go/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backup_$DATE.sql"

# Créer le répertoire de sauvegarde
mkdir -p $BACKUP_DIR

# Sauvegarder la base de données
docker exec sandbox-api-go_postgres_1 pg_dump -U postgres sandbox_api > $BACKUP_DIR/$BACKUP_FILE

# Compresser la sauvegarde
gzip $BACKUP_DIR/$BACKUP_FILE

# Supprimer les sauvegardes de plus de 30 jours
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Sauvegarde créée: $BACKUP_FILE.gz"
EOF

# Rendre le script exécutable
chmod +x backup.sh

# Ajouter au crontab pour une sauvegarde quotidienne
crontab -e
# Ajouter cette ligne :
# 0 2 * * * /opt/apps/sandbox-api-go/backup.sh
```

### 📊 Monitoring et logs

#### 1. Logs structurés
```bash
# Voir les logs en temps réel
docker-compose -f docker-compose.prod.yml logs -f api

# Logs de la base de données
docker-compose -f docker-compose.prod.yml logs -f postgres
```

#### 2. Surveillance des ressources
```bash
# Installer htop pour surveiller les ressources
sudo apt install htop -y

# Surveiller les conteneurs Docker
docker stats

# Vérifier l'espace disque
df -h
```

### 🚀 Mise à jour de l'application

#### 1. Mise à jour automatique
```bash
# Créer un script de mise à jour
cat > update.sh << 'EOF'
#!/bin/bash
cd /opt/apps/sandbox-api-go

# Sauvegarder avant mise à jour
./backup.sh

# Récupérer les dernières modifications
git pull origin main

# Reconstruire et redémarrer
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d --build

echo "Application mise à jour avec succès!"
EOF

chmod +x update.sh
```

#### 2. Rollback en cas de problème
```bash
# Script de rollback
cat > rollback.sh << 'EOF'
#!/bin/bash
cd /opt/apps/sandbox-api-go

# Arrêter l'application
docker-compose -f docker-compose.prod.yml down

# Restaurer la dernière sauvegarde
LATEST_BACKUP=$(ls -t backups/*.sql.gz | head -1)
if [ -n "$LATEST_BACKUP" ]; then
    echo "Restauration de $LATEST_BACKUP..."
    gunzip -c $LATEST_BACKUP | docker exec -i sandbox-api-go_postgres_1 psql -U postgres -d sandbox_api
fi

# Redémarrer avec la version précédente
git checkout HEAD~1
docker-compose -f docker-compose.prod.yml up -d --build

echo "Rollback effectué!"
EOF

chmod +x rollback.sh
```

### 🔧 Variables d'environnement de production

```bash
# .env.prod
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=<mot_de_passe_très_fort>
DB_NAME=sandbox_api
DB_SSLMODE=disable

JWT_SECRET=<clé_jwt_très_longue_et_aléatoire>
API_PORT=8080

POSTGRES_DB=sandbox_api
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<même_mot_de_passe_que_DB_PASSWORD>
```

### 📋 Checklist de déploiement

- [ ] Serveur configuré avec Docker et Docker Compose
- [ ] Pare-feu configuré (ports 80, 443, SSH)
- [ ] Variables d'environnement sécurisées
- [ ] Base de données avec mot de passe fort
- [ ] Nginx configuré comme reverse proxy
- [ ] Certificat SSL installé (Let's Encrypt)
- [ ] Sauvegardes automatiques configurées
- [ ] Scripts de mise à jour et rollback créés
- [ ] Monitoring et logs configurés
- [ ] Tests de l'application effectués

## 🧪 Tests

### Tester l'API
```bash
# Test de l'endpoint racine
curl http://localhost:8080/

# Test de l'inscription
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@test.com","password":"test123"}'
```

## 📁 Structure du projet

```
sandbox-api-go/
├── auth/           # Gestion des JWT
├── database/       # Connexion PostgreSQL et migrations
│   └── migrations/ # Migrations de la base de données
├── handlers/       # Gestionnaires HTTP
├── middleware/     # Middleware d'authentification et erreurs
├── models/         # Modèles de données
├── storage/        # Service MinIO pour la gestion des fichiers
├── logger/         # Système de logs
├── errors/         # Gestion des erreurs
├── metrics/        # Métriques Prometheus
├── monitoring/     # Configuration de l'observabilité
├── Dockerfile      # Configuration Docker
├── docker-compose.yml  # Orchestration des services
├── init.sql        # Initialisation de la base de données
└── main.go         # Point d'entrée de l'application
```

## 🤝 Contribution

1. Fork le projet
2. Créer une branche feature
3. Commiter vos changements
4. Pousser vers la branche
5. Ouvrir une Pull Request

## 📄 Licence

Ce projet est sous licence MIT. Voir le fichier `LICENSE` pour plus de détails.

---

**Développé avec ❤️ en Go**
