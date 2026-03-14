#!/bin/bash

# Script de déploiement - Build et push vers le registry privé

set -e

# Charger les variables d'environnement de production
if [ -f .env.prod ]; then
    export $(cat .env.prod | grep -v '^#' | xargs)
else
    echo "Erreur: Fichier .env.prod introuvable"
    exit 1
fi

# Vérifier que les variables essentielles sont définies
if [ -z "$REGISTRY_URL" ] || [ -z "$IMAGE_NAME" ] || [ -z "$IMAGE_TAG" ]; then
    echo "Erreur: Variables REGISTRY_URL, IMAGE_NAME ou IMAGE_TAG non définies"
    exit 1
fi

# Nom complet de l'image
FULL_IMAGE_NAME="${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG}"

echo "============================================"
echo "Déploiement de l'application"
echo "============================================"
echo "Registry: ${REGISTRY_URL}"
echo "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo "============================================"

# 1. Login au registry
echo ""
echo "Étape 1/4: Connexion au registry..."
if [ -n "$REGISTRY_USER" ] && [ -n "$REGISTRY_PASSWORD" ]; then
    echo "$REGISTRY_PASSWORD" | docker login "${REGISTRY_URL}" -u "${REGISTRY_USER}" --password-stdin
    echo "✓ Connexion réussie"
else
    echo "⚠ REGISTRY_USER ou REGISTRY_PASSWORD non défini, tentative sans authentification..."
fi

# 2. Build de l'image (multi-architecture pour AMD64)
echo ""
echo "Étape 2/4: Construction de l'image Docker pour AMD64..."
docker buildx build --platform linux/amd64 -t "${FULL_IMAGE_NAME}" --load .
echo "✓ Image construite: ${FULL_IMAGE_NAME}"

# 3. Tag avec 'latest' et date
DATE_TAG="${REGISTRY_URL}/${IMAGE_NAME}:$(date +%Y%m%d-%H%M%S)"
docker tag "${FULL_IMAGE_NAME}" "${DATE_TAG}"
echo "✓ Tag additionnel créé: ${DATE_TAG}"

# 4. Push vers le registry
echo ""
echo "Étape 3/4: Push de l'image vers le registry..."
docker push "${FULL_IMAGE_NAME}"
docker push "${DATE_TAG}"
echo "✓ Images poussées vers le registry"

# 5. Instructions de déploiement
echo ""
echo "============================================"
echo "✓ Build et push réussis!"
echo "============================================"
echo ""
echo "Pour déployer sur votre VPS:"
echo ""
echo "1. Copiez les fichiers nécessaires sur le VPS:"
echo "   scp docker-compose.prod.yml .env.prod user@your-vps:/path/to/app/"
echo "   scp -r monitoring user@your-vps:/path/to/app/"
echo "   scp init.sql user@your-vps:/path/to/app/"
echo ""
echo "2. Connectez-vous au VPS et déployez:"
echo "   ssh user@your-vps"
echo "   cd /path/to/app"
echo "   /// DébutOptionnel////"
echo "   docker login ${REGISTRY_URL}"
echo "   /// Fin Optionnel////"
echo "   docker compose -f compose.prod.yaml --env-file .env.prod pull api"
echo "   docker compose -f compose.prod.yaml --env-file .env.prod up -d"
echo ""
echo "3. Vérifiez le déploiement:"
echo "   docker-compose -f docker-compose.prod.yml ps"
echo "   docker-compose -f docker-compose.prod.yml logs -f api"
echo ""
echo "Images disponibles:"
echo "  - ${FULL_IMAGE_NAME}"
echo "  - ${DATE_TAG}"
echo "============================================"
