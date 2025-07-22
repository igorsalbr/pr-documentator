#!/bin/bash

# Script completo para testar o desenvolvimento local
# Inclui: setup, build, test, webhook simulation

set -e

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

echo -e "${BLUE}🚀 PR Documentator - Teste Local Completo${NC}"
echo "========================================"
echo

# Função para mostrar progresso
show_step() {
    echo -e "${PURPLE}📋 PASSO $1: $2${NC}"
    echo "----------------------------------------"
}

# 1. Verificar dependências
show_step "1" "Verificando dependências"
echo -n "Go: "
if command -v go >/dev/null 2>&1; then
    echo -e "${GREEN}✓ $(go version | cut -d' ' -f3)${NC}"
else
    echo -e "${RED}✗ Go não encontrado${NC}"
    exit 1
fi

echo -n "OpenSSL: "
if command -v openssl >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Disponível${NC}"
else
    echo -e "${RED}✗ OpenSSL não encontrado${NC}"
    exit 1
fi

echo -n "jq (opcional): "
if command -v jq >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Disponível${NC}"
    JQ_AVAILABLE=true
else
    echo -e "${YELLOW}⚠ jq não encontrado (JSON não será formatado)${NC}"
    JQ_AVAILABLE=false
fi

echo

# 2. Configurar environment
show_step "2" "Configurando environment"
if [[ ! -f ".env" ]]; then
    echo "📄 Criando arquivo .env a partir do exemplo..."
    cp .env.example .env
    echo -e "${YELLOW}⚠️  Configure suas API keys no arquivo .env antes de continuar!${NC}"
    echo "   CLAUDE_API_KEY=sua-chave-claude"
    echo "   POSTMAN_API_KEY=sua-chave-postman"
    echo "   POSTMAN_WORKSPACE_ID=seu-workspace"
    echo "   POSTMAN_COLLECTION_ID=sua-collection"
    echo
    read -p "Pressione Enter quando terminar de configurar o .env..."
else
    echo -e "${GREEN}✓ Arquivo .env já existe${NC}"
fi

# Verificar se as chaves principais estão configuradas
source .env
if [[ -z "$CLAUDE_API_KEY" || "$CLAUDE_API_KEY" == "sk-ant-api03-your-key-here" ]]; then
    echo -e "${YELLOW}⚠️  CLAUDE_API_KEY não configurada - testes com Claude não funcionarão${NC}"
fi

if [[ -z "$POSTMAN_API_KEY" || "$POSTMAN_API_KEY" == "PMAK-your-key-here" ]]; then
    echo -e "${YELLOW}⚠️  POSTMAN_API_KEY não configurada - updates no Postman não funcionarão${NC}"
fi

echo

# 3. Instalar dependências
show_step "3" "Instalando dependências Go"
go mod download
go mod tidy
echo -e "${GREEN}✓ Dependências instaladas${NC}"
echo

# 4. Gerar certificados
show_step "4" "Gerando certificados HTTPS"
if [[ ! -f "certs/server.crt" || ! -f "certs/server.key" ]]; then
    ./scripts/generate_certs.sh
    echo -e "${GREEN}✓ Certificados gerados${NC}"
else
    echo -e "${GREEN}✓ Certificados já existem${NC}"
fi
echo

# 5. Build do projeto
show_step "5" "Compilando projeto"
if go build -o bin/pr-documentator cmd/server/main.go; then
    echo -e "${GREEN}✓ Projeto compilado com sucesso${NC}"
else
    echo -e "${RED}✗ Falha na compilação${NC}"
    exit 1
fi
echo

# 6. Executar testes unitários
show_step "6" "Executando testes unitários"
if go test ./... -v; then
    echo -e "${GREEN}✓ Todos os testes passaram${NC}"
else
    echo -e "${YELLOW}⚠️  Alguns testes falharam (pode ser normal se as APIs não estiverem configuradas)${NC}"
fi
echo

# 7. Iniciar servidor em background
show_step "7" "Iniciando servidor"
echo "🌟 Iniciando servidor em background..."
./bin/pr-documentator &
SERVER_PID=$!

# Aguardar o servidor inicializar
echo "⏳ Aguardando servidor inicializar..."
sleep 3

# Verificar se o servidor está rodando
if kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${GREEN}✓ Servidor iniciado (PID: $SERVER_PID)${NC}"
else
    echo -e "${RED}✗ Falha ao iniciar servidor${NC}"
    exit 1
fi
echo

# 8. Testar health check
show_step "8" "Testando health check"
if curl -s -k https://localhost:8443/health >/dev/null; then
    HEALTH_RESPONSE=$(curl -s -k https://localhost:8443/health)
    echo -e "${GREEN}✓ Health check OK${NC}"
    if $JQ_AVAILABLE; then
        echo "$HEALTH_RESPONSE" | jq .
    else
        echo "$HEALTH_RESPONSE"
    fi
else
    echo -e "${RED}✗ Health check falhou${NC}"
    echo "Logs do servidor:"
    jobs -p | xargs -I{} kill {}
    exit 1
fi
echo

# 9. Testar webhook
show_step "9" "Testando webhook simulation"
echo "🎯 Enviando payload de teste do GitHub..."

# Definir secret para teste
TEST_SECRET="test-secret-for-local-development"
export GITHUB_WEBHOOK_SECRET="$TEST_SECRET"

# Executar o teste de webhook
if ./scripts/test_webhook.sh "$TEST_SECRET" "https://localhost:8443"; then
    echo -e "${GREEN}✓ Teste de webhook concluído${NC}"
else
    echo -e "${YELLOW}⚠️  Teste de webhook teve problemas (verifique configuração de APIs)${NC}"
fi
echo

# 10. Cleanup
show_step "10" "Finalizando"
echo "🛑 Parando servidor..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo -e "${GREEN}✓ Servidor parado${NC}"
echo

echo -e "${GREEN}🎉 Teste local completo finalizado!${NC}"
echo
echo -e "${BLUE}📚 Próximos passos:${NC}"
echo "1. Configure suas API keys no .env para funcionalidade completa"
echo "2. Execute 'make dev' para desenvolvimento com hot reload"
echo "3. Use 'make test' para executar testes"
echo "4. Configure webhook real no GitHub apontando para seu servidor"
echo
echo -e "${BLUE}🔗 URLs úteis:${NC}"
echo "• Health: https://localhost:8443/health"
echo "• Webhook: https://localhost:8443/analyze-pr"
echo "• Documentação: README.md"