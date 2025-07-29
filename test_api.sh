#!/bin/bash

# Couleurs pour l'affichage
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Test de l'API REST Go${NC}"
echo "=============================="

# Test 1: Page d'accueil
echo -e "\n${YELLOW}1. Test GET / (page d'accueil)${NC}"
curl -s http://localhost:8080/ | jq '.'

# Test 2: Lister toutes les tâches
echo -e "\n${YELLOW}2. Test GET /api/tasks (lister toutes les tâches)${NC}"
curl -s http://localhost:8080/api/tasks | jq '.'

# Test 3: Créer une nouvelle tâche
echo -e "\n${YELLOW}3. Test POST /api/tasks (créer une nouvelle tâche)${NC}"
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Ma première tâche API","description":"Créée via curl","completed":false}' | jq '.'

# Test 4: Obtenir une tâche spécifique
echo -e "\n${YELLOW}4. Test GET /api/tasks/1 (obtenir tâche ID 1)${NC}"
curl -s http://localhost:8080/api/tasks/1 | jq '.'

# Test 5: Mettre à jour une tâche
echo -e "\n${YELLOW}5. Test PUT /api/tasks/2 (mettre à jour tâche ID 2)${NC}"
curl -s -X PUT http://localhost:8080/api/tasks/2 \
  -H "Content-Type: application/json" \
  -d '{"title":"Tâche modifiée","description":"Description mise à jour","completed":true}' | jq '.'

# Test 6: Vérifier les modifications
echo -e "\n${YELLOW}6. Test GET /api/tasks (vérifier les modifications)${NC}"
curl -s http://localhost:8080/api/tasks | jq '.'

# Test 7: Supprimer une tâche
echo -e "\n${YELLOW}7. Test DELETE /api/tasks/1 (supprimer tâche ID 1)${NC}"
curl -s -X DELETE http://localhost:8080/api/tasks/1 -w "Status: %{http_code}\n"

# Test 8: Vérifier la suppression
echo -e "\n${YELLOW}8. Test GET /api/tasks (après suppression)${NC}"
curl -s http://localhost:8080/api/tasks | jq '.'

# Test 9: Erreur 404
echo -e "\n${YELLOW}9. Test GET /api/tasks/999 (erreur 404)${NC}"
curl -s http://localhost:8080/api/tasks/999 | jq '.'

# Test 10: Erreur de validation
echo -e "\n${YELLOW}10. Test POST sans titre (erreur de validation)${NC}"
curl -s -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"description":"Sans titre","completed":false}' | jq '.'

echo -e "\n${GREEN}✅ Tests terminés !${NC}" 