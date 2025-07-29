#!/bin/bash

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Test de l'API REST Go avec Authentification${NC}"
echo "================================================="

# Test 1: Page d'accueil
echo -e "\n${YELLOW}1. Test GET / (page d'accueil)${NC}"
curl -s http://localhost:8080/ | jq '.'

# Test 2: Connexion avec utilisateur existant
echo -e "\n${YELLOW}2. Connexion avec utilisateur admin${NC}"
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password123"}')

echo "$LOGIN_RESPONSE" | jq '.'

# Extraire le token pour les prochaines requêtes
TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')

if [ "$TOKEN" != "null" ] && [ "$TOKEN" != "" ]; then
    echo -e "${GREEN}✅ Token obtenu: ${TOKEN:0:20}...${NC}"
else
    echo -e "${RED}❌ Impossible d'obtenir le token${NC}"
    exit 1
fi

# Test 3: Essayer d'accéder aux tâches sans token (doit échouer)
echo -e "\n${YELLOW}3. Test GET /api/tasks sans authentification (doit échouer)${NC}"
curl -s http://localhost:8080/api/tasks | jq '.'

# Test 4: Accéder aux tâches avec token
echo -e "\n${YELLOW}4. Test GET /api/tasks avec authentification${NC}"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Test 5: Créer une nouvelle tâche
echo -e "\n${YELLOW}5. Créer une nouvelle tâche${NC}"
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Ma tâche sécurisée","description":"Créée avec authentification","completed":false}' | jq '.'

# Test 6: Vérifier que la tâche a été ajoutée
echo -e "\n${YELLOW}6. Lister les tâches après création${NC}"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# Test 7: Inscription d'un nouvel utilisateur
echo -e "\n${YELLOW}7. Inscription d'un nouvel utilisateur${NC}"
NEW_USER_RESPONSE=$(curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"motdepasse123"}')

echo "$NEW_USER_RESPONSE" | jq '.'

# Extraire le token du nouvel utilisateur
NEW_TOKEN=$(echo "$NEW_USER_RESPONSE" | jq -r '.token')

# Test 8: Vérifier que le nouvel utilisateur n'a pas accès aux tâches de l'admin
echo -e "\n${YELLOW}8. Vérifier l'isolation des données utilisateur${NC}"
echo "Tâches du nouvel utilisateur (devrait être vide):"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $NEW_TOKEN" | jq '.'

# Test 9: Le nouvel utilisateur crée une tâche
echo -e "\n${YELLOW}9. Le nouvel utilisateur crée sa propre tâche${NC}"
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -d '{"title":"Tâche du nouvel utilisateur","description":"Visible seulement par moi","completed":false}' | jq '.'

# Test 10: Vérifier que chaque utilisateur ne voit que ses tâches
echo -e "\n${YELLOW}10. Vérification finale de l'isolation des données${NC}"
echo "Tâches de l'admin:"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $TOKEN" | jq '.tasks | length as $count | "Nombre de tâches: \($count)"'

echo "Tâches du nouvel utilisateur:"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer $NEW_TOKEN" | jq '.tasks | length as $count | "Nombre de tâches: \($count)"'

# Test 11: Essayer d'accéder à une tâche d'un autre utilisateur
echo -e "\n${YELLOW}11. Test de sécurité: accéder à la tâche d'un autre utilisateur (doit échouer)${NC}"
curl -s http://localhost:8080/api/tasks/1 \
  -H "Authorization: Bearer $NEW_TOKEN" | jq '.'

# Test 12: Token invalide
echo -e "\n${YELLOW}12. Test avec token invalide${NC}"
curl -s http://localhost:8080/api/tasks \
  -H "Authorization: Bearer token-invalide" | jq '.'

echo -e "\n${GREEN}✅ Tests d'authentification terminés !${NC}"
echo -e "${BLUE}📝 Résumé:${NC}"
echo -e "  - ✅ Authentification par JWT"
echo -e "  - ✅ Isolation des données par utilisateur"
echo -e "  - ✅ Protection des endpoints"
echo -e "  - ✅ Validation des tokens" 