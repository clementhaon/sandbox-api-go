# API REST Go avec Authentification JWT

Une API REST moderne en Go avec authentification par JWT, architecture modulaire et protection des endpoints.

## 🚀 Fonctionnalités

- **Authentification JWT** - Tokens sécurisés avec expiration
- **Architecture modulaire** - Code organisé en packages séparés
- **Protection des endpoints** - Middleware d'authentification
- **Isolation des données** - Chaque utilisateur ne voit que ses tâches
- **API RESTful complète** - CRUD operations pour les tâches

## 📁 Structure du projet

```
sandbox-api-go/
├── main.go              # Point d'entrée principal
├── models/              # Structures de données
│   ├── task.go         # Modèle Task
│   └── user.go         # Modèle User + Auth types
├── handlers/            # Handlers HTTP
│   ├── auth.go         # Inscription/Connexion
│   └── tasks.go        # CRUD des tâches
├── middleware/          # Middlewares
│   └── auth.go         # Validation JWT
├── auth/               # Utilities JWT
│   └── jwt.go          # Génération/validation tokens
└── go.mod              # Dépendances Go
```

## 🔧 Installation

```bash
# Cloner le dépôt
git clone <url>
cd sandbox-api-go

# Installer les dépendances
go mod tidy

# Lancer le serveur
go run main.go
```

## 📚 Endpoints

### Authentification (publique)

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| POST | `/auth/register` | Inscription d'un nouvel utilisateur |
| POST | `/auth/login` | Connexion et récupération du token |

### Tâches (authentification requise)

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/api/tasks` | Lister toutes vos tâches |
| POST | `/api/tasks` | Créer une nouvelle tâche |
| GET | `/api/tasks/{id}` | Obtenir une tâche spécifique |
| PUT | `/api/tasks/{id}` | Mettre à jour une tâche |
| DELETE | `/api/tasks/{id}` | Supprimer une tâche |

## 🔑 Utilisation

### 1. Inscription
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"monuser","email":"user@example.com","password":"motdepasse"}'
```

### 2. Connexion
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password123"}'
```

### 3. Utiliser le token pour accéder aux tâches
```bash
# Remplacez YOUR_TOKEN par le token reçu
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/tasks
```

### 4. Créer une tâche
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"title":"Ma tâche","description":"Description","completed":false}'
```

## 🔒 Sécurité

- **JWT avec expiration** - Tokens valides 24h
- **Isolation utilisateur** - Chaque utilisateur ne peut accéder qu'à ses données
- **Middleware de protection** - Tous les endpoints tasks sont protégés
- **Validation des tokens** - Vérification automatique à chaque requête

## 👤 Utilisateur de test

- **Username**: `admin`
- **Password**: `password123`

## 🛠️ Technologies utilisées

- **Go 1.21+** - Langage principal
- **JWT (golang-jwt/jwt/v5)** - Authentification
- **HTTP standard library** - Serveur web
- **JSON** - Format d'échange de données

## 🔮 Améliorations possibles

- [ ] Base de données (PostgreSQL/MongoDB)
- [ ] Hashage des mots de passe (bcrypt)
- [ ] Refresh tokens
- [ ] Rate limiting
- [ ] Tests unitaires
- [ ] Docker
- [ ] CORS middleware
- [ ] Logging structuré
