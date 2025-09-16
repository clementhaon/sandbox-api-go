# 📊 Stack d'Observabilité - Sandbox API

Cette documentation décrit la stack complète d'observabilité mise en place pour surveiller l'API Sandbox Go.

## 🚀 Services Déployés

### **Prometheus** (Port 9090)
- **Collecte de métriques** en temps réel
- **Rétention** : 200 heures de données
- **Alertes** configurées pour erreurs, latence, et disponibilité

**Métriques collectées :**
- `http_requests_total` - Nombre total de requêtes HTTP
- `http_request_duration_seconds` - Latence des requêtes
- `database_operations_total` - Opérations base de données
- `auth_attempts_total` - Tentatives d'authentification
- `errors_total` - Erreurs par type et code

**URLs :**
- Interface : http://localhost:9090
- Métriques API : http://localhost:8080/metrics

### **Grafana** (Port 3000)
- **Dashboards** visuels pour métriques et logs
- **Alertes** configurables avec notifications
- **Exploration** de données en temps réel

**Accès :**
- URL : http://localhost:3000
- Utilisateur : `admin`
- Mot de passe : `admin123`

**Dashboards disponibles :**
- **API Overview** : Vue d'ensemble des performances
- **Logs Overview** : Analyse des logs en temps réel

### **Loki** (Port 3100)
- **Agrégation de logs** centralisée
- **Recherche** et filtrage avancés
- **Rétention** configurable des logs

### **Promtail**
- **Agent de collecte** de logs
- **Collecte automatique** des logs Docker
- **Parsing** et labélisation des logs

### **Node Exporter** (Port 9100)
- **Métriques système** (CPU, RAM, disque)
- **Monitoring infrastructure** complet

### **Postgres Exporter** (Port 9187)
- **Métriques PostgreSQL** détaillées
- **Performance** et santé de la base

## 🚢 Démarrage

```bash
# Démarrer toute la stack
docker-compose up -d

# Vérifier le statut
docker-compose ps

# Voir les logs
docker-compose logs -f prometheus grafana loki
```

## 📈 Dashboards Grafana

### 1. **API Overview Dashboard**
- **Request Rate** : Requêtes par seconde
- **Response Time** : Latence (95e percentile)
- **Error Rate** : Pourcentage d'erreurs
- **Active Users** : Utilisateurs connectés
- **HTTP Status Codes** : Distribution par code
- **Database Operations** : Opérations par type

### 2. **Logs Dashboard**
- **Error Logs** : Logs d'erreur en temps réel
- **Recent API Logs** : Derniers logs de l'API
- **Log Levels** : Distribution par niveau
- **Error Rate** : Évolution du taux d'erreur

## ⚠️ Alertes Configurées

### **Critiques**
- **API Down** : API inaccessible (1 min)
- **Database Errors** : Erreurs DB > 0.1/sec (2 min)

### **Warnings**
- **High Error Rate** : Erreurs 5xx > 0.1/sec (2 min)
- **High Response Time** : 95e percentile > 1s (3 min)
- **High DB Latency** : DB latence > 0.5s (5 min)
- **Auth Failures** : Échecs auth > 0.5/sec (3 min)

## 🔍 Requêtes Prometheus Utiles

```promql
# Taux de requêtes par endpoint
sum(rate(http_requests_total[5m])) by (endpoint)

# Latence par percentile
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# Taux d'erreur global
sum(rate(http_requests_total{status_code=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100

# Opérations DB les plus lentes
histogram_quantile(0.95, sum(rate(database_operation_duration_seconds_bucket[5m])) by (operation, le))
```

## 📝 Recherche Logs avec Loki

```logql
# Logs d'erreur de l'API
{job="sandbox-api"} |= "ERROR"

# Logs d'authentification
{job="sandbox-api"} | json | level="WARN" | message=~".*auth.*"

# Logs avec latence élevée
{job="sandbox-api"} | json | duration > 1s

# Agrégation par endpoint
sum(rate({job="sandbox-api"} [5m])) by (endpoint)
```

## 📊 Métriques Métier

L'API expose des métriques spécifiques :

- **Authentification** : Succès/échecs par type
- **Tâches** : Création, modification, suppression
- **Validation** : Erreurs de validation par champ
- **Performance** : Latence par endpoint et opération

## 🔧 Configuration Avancée

### Personnaliser les Alertes
Modifier `monitoring/prometheus/alert_rules.yml` et recharger :
```bash
curl -X POST http://localhost:9090/-/reload
```

### Ajouter des Dashboards
1. Copier le fichier JSON dans `monitoring/grafana/dashboards/`
2. Redémarrer Grafana : `docker-compose restart grafana`

### Rétention des Logs
Modifier `monitoring/loki/loki.yml` :
```yaml
limits_config:
  retention_period: 168h  # 7 jours
```

## 🚨 Troubleshooting

### Prometheus ne collecte pas les métriques
```bash
# Vérifier la configuration
curl http://localhost:9090/api/v1/targets

# Tester l'endpoint metrics de l'API
curl http://localhost:8080/metrics
```

### Grafana n'affiche pas de données
1. Vérifier les datasources : Settings > Data Sources
2. Tester la connectivité Prometheus/Loki
3. Vérifier les requêtes dans les dashboards

### Logs manquants dans Loki
```bash
# Vérifier Promtail
docker-compose logs promtail

# Tester l'API Loki
curl http://localhost:3100/ready
```

## 📚 Ressources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Loki Documentation](https://grafana.com/docs/loki/)
- [PromQL Guide](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [LogQL Guide](https://grafana.com/docs/loki/latest/logql/)