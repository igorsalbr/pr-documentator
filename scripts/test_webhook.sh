#!/bin/bash

# Script para testar o webhook localmente
# Usage: ./scripts/test_webhook.sh [webhook_secret] [server_url]

set -e

WEBHOOK_SECRET="${1:-test-secret-123}"
SERVER_URL="${2:-https://localhost:8443}"
PAYLOAD_FILE="test/fixtures/github_pr_opened.json"

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Testando webhook do PR Documentator${NC}"
echo "📍 URL: $SERVER_URL/analyze-pr"
echo "🔐 Secret: $WEBHOOK_SECRET"
echo "📄 Payload: $PAYLOAD_FILE"
echo

# Verificar se o payload existe
if [[ ! -f "$PAYLOAD_FILE" ]]; then
    echo -e "${RED}❌ Payload file not found: $PAYLOAD_FILE${NC}"
    exit 1
fi

# Ler o payload
PAYLOAD=$(cat "$PAYLOAD_FILE")

# Calcular a assinatura HMAC-SHA256
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | sed 's/^.* //')

echo -e "${YELLOW}📋 Detalhes da requisição:${NC}"
echo "Content-Type: application/json"
echo "X-GitHub-Event: pull_request"
echo "X-Hub-Signature-256: sha256=$SIGNATURE"
echo

# Fazer a requisição
echo -e "${BLUE}🚀 Enviando requisição...${NC}"
echo

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SERVER_URL/analyze-pr" \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD" \
  -k)

# Separar response body do status code
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$RESPONSE" | head -n -1)

echo -e "${YELLOW}📊 Resposta do servidor:${NC}"
echo "Status Code: $HTTP_CODE"
echo

# Verificar o status code
case $HTTP_CODE in
    200)
        echo -e "${GREEN}✅ Sucesso! Webhook processado corretamente${NC}"
        echo -e "${BLUE}📝 Response Body:${NC}"
        echo "$RESPONSE_BODY" | jq . 2>/dev/null || echo "$RESPONSE_BODY"
        ;;
    400)
        echo -e "${RED}❌ Bad Request - Verifique o payload${NC}"
        echo -e "${YELLOW}Response:${NC} $RESPONSE_BODY"
        ;;
    401)
        echo -e "${RED}❌ Unauthorized - Verifique o webhook secret${NC}"
        echo -e "${YELLOW}Response:${NC} $RESPONSE_BODY"
        ;;
    500)
        echo -e "${RED}❌ Internal Server Error${NC}"
        echo -e "${YELLOW}Response:${NC} $RESPONSE_BODY"
        ;;
    *)
        echo -e "${RED}❌ Unexpected status code: $HTTP_CODE${NC}"
        echo -e "${YELLOW}Response:${NC} $RESPONSE_BODY"
        ;;
esac

echo
echo -e "${BLUE}📚 Para mais testes:${NC}"
echo "1. Health check: curl -k $SERVER_URL/health"
echo "2. Sem secret: GITHUB_WEBHOOK_SECRET=\"\" ./scripts/test_webhook.sh"
echo "3. Secret incorreto: ./scripts/test_webhook.sh wrong-secret"