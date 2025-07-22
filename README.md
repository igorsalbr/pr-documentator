# PR Documentator

Um serviço Go modular que analisa Pull Requests do GitHub usando Claude AI para detectar mudanças em APIs e atualizar automaticamente a documentação no Postman.

## 📋 Índice

- [Visão Geral](#-visão-geral)
- [Arquitetura](#️-arquitetura)
- [Pré-requisitos](#-pré-requisitos)
- [Instalação](#-instalação-passo-a-passo)
- [Configuração](#️-configuração-detalhada)
- [Executando o Projeto](#-executando-o-projeto)
- [API Endpoints](#-api-endpoints)
- [Estrutura do Projeto](#-estrutura-do-projeto)
- [Desenvolvimento](#️-desenvolvimento)
- [Testes](#-testes)
- [Troubleshooting](#-troubleshooting)
- [Recursos Adicionais](#-recursos-adicionais)

## 🎯 Visão Geral

O **PR Documentator** automatiza a documentação de APIs através da análise inteligente de Pull Requests. O sistema:

1. 🔗 **Recebe webhooks** de Pull Requests do GitHub via HTTPS
2. 🤖 **Analisa mudanças** usando Claude AI para detectar alterações em APIs
3. 📊 **Identifica rotas** novas, modificadas ou removidas com seus payloads
4. 📝 **Atualiza automaticamente** a documentação no Postman
5. ✅ **Retorna feedback** estruturado sobre as mudanças processadas

## 🏗️ Arquitetura

O sistema segue uma arquitetura limpa e modular:

```
┌─────────────┐    HTTPS     ┌──────────────┐    AI Analysis    ┌─────────────┐
│   GitHub    │──────────────▶│ PR Analyzer  │───────────────────▶│   Claude    │
│  Webhook    │   Webhook     │   Service    │    Diff + Prompt   │     API     │
└─────────────┘               └──────┬───────┘                   └─────────────┘
                                     │
                              Update │
                           Collection│
                                     ▼
                              ┌─────────────┐
                              │   Postman   │
                              │     API     │
                              └─────────────┘
```

### Componentes Principais

- **HTTPS Server**: Gorilla Mux com middlewares de segurança e recuperação
- **Claude Integration**: Cliente Resty com circuit breaker para análise via AI
- **Postman Integration**: Cliente Resty com retry automático para coleções
- **GitHub Webhook Validation**: Verificação HMAC com assinaturas SHA-256
- **Circuit Breaker**: Proteção contra falhas em cascata nas APIs externas
- **Prometheus Metrics**: Observabilidade e monitoramento de performance
- **Dependency Injection**: Arquitetura limpa com interfaces
- **Structured Logging**: Logs estruturados com zerolog

## 📦 Pré-requisitos

Antes de começar, certifique-se de ter:

- **Go 1.21+** instalado ([download](https://golang.org/dl/))
- **Git** para controle de versão
- **OpenSSL** para gerar certificados HTTPS (pré-instalado no macOS/Linux)
- **Conta Anthropic** com API key do Claude ([console.anthropic.com](https://console.anthropic.com))
- **Conta Postman** com API key ([postman.com](https://www.postman.com))
- **Docker** (opcional) para containerização

## 🚀 Instalação Passo a Passo

### 1. Clone e Configure o Projeto

```bash
# Clone o repositório
git clone https://github.com/igorsal/pr-documentator.git
cd pr-documentator

# Instale as dependências do Go
go mod download
go mod tidy

# Instale ferramentas de desenvolvimento (opcional)
make install-tools
```

### 2. Configure as Variáveis de Ambiente

```bash
# Copie o arquivo de exemplo
cp .env.example .env

# Edite o arquivo .env com suas credenciais
nano .env  # ou use seu editor preferido
```

**Exemplo de configuração `.env`:**

```env
# ========================================
# Server Configuration
# ========================================
SERVER_HOST=0.0.0.0
SERVER_PORT=8443
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# ========================================
# TLS Configuration (Certificados HTTPS)
# ========================================
TLS_CERT_FILE=./certs/server.crt
TLS_KEY_FILE=./certs/server.key

# ========================================
# Claude API Configuration
# ========================================
CLAUDE_API_KEY=sk-ant-api03-SUA-CHAVE-AQUI
CLAUDE_MODEL=claude-3-sonnet-20240229
CLAUDE_MAX_TOKENS=4096
CLAUDE_BASE_URL=https://api.anthropic.com
CLAUDE_TIMEOUT=30s

# ========================================
# Postman API Configuration
# ========================================
POSTMAN_API_KEY=PMAK-SUA-CHAVE-AQUI
POSTMAN_WORKSPACE_ID=seu-workspace-id
POSTMAN_COLLECTION_ID=seu-collection-id
POSTMAN_BASE_URL=https://api.postman.com
POSTMAN_TIMEOUT=30s

# ========================================
# GitHub Configuration
# ========================================
GITHUB_WEBHOOK_SECRET=seu-webhook-secret-seguro

# ========================================
# Logging Configuration
# ========================================
LOG_LEVEL=info
LOG_FORMAT=json
```

### 3. Gere Certificados HTTPS

```bash
# Torne o script executável (se necessário)
chmod +x scripts/generate_certs.sh

# Gere os certificados
./scripts/generate_certs.sh
# ou use o Makefile
make gen-certs
```

O script irá criar:
- `certs/server.crt` - Certificado SSL auto-assinado
- `certs/server.key` - Chave privada RSA 2048-bit

## ⚙️ Configuração Detalhada

### Configurando Claude API

1. **Crie uma conta** em [console.anthropic.com](https://console.anthropic.com)
2. **Gere uma API Key:**
   - Vá para "API Keys" no dashboard
   - Clique em "Create Key"
   - Copie a chave (começa com `sk-ant-api03-`)
3. **Adicione no `.env`:**
   ```env
   CLAUDE_API_KEY=sk-ant-api03-sua-chave-real-aqui
   ```

### Configurando Postman API

1. **Obtenha sua API Key:**
   - Acesse [postman.com](https://www.postman.com)
   - Vá em "Account Settings" → "API Keys"
   - Gere uma nova chave (começa com `PMAK-`)

2. **Encontre os IDs necessários:**
   ```bash
   # Workspace ID - na URL do Postman
   # https://app.postman.com/workspace/SEU-WORKSPACE-ID
   
   # Collection ID - clique na coleção, vá em Info
   # Ou use a API para listar:
   curl -X GET https://api.postman.com/collections \
     -H "X-API-Key: SUA-API-KEY"
   ```

3. **Configure no `.env`:**
   ```env
   POSTMAN_API_KEY=PMAK-sua-chave-aqui
   POSTMAN_WORKSPACE_ID=workspace-id-aqui
   POSTMAN_COLLECTION_ID=collection-id-aqui
   ```

### Configurando GitHub Webhook

1. **No seu repositório GitHub:**
   - Vá em `Settings` → `Webhooks`
   - Clique em "Add webhook"

2. **Configure o webhook:**
   ```
   Payload URL: https://seu-dominio.com:8443/analyze-pr
   Content type: application/json
   Secret: gere-um-secret-seguro-aqui
   Events: Selecione "Pull requests"
   Active: ✓
   ```

3. **Gere um secret seguro:**
   ```bash
   # Gere um secret aleatório
   openssl rand -hex 32
   
   # Adicione ao .env
   GITHUB_WEBHOOK_SECRET=seu-secret-gerado-aqui
   ```

## 🏃 Executando o Projeto

### Desenvolvimento (Recomendado)

```bash
# Com hot reload (requer air)
make dev

# Ou instale o air primeiro e execute
go install github.com/cosmtrek/air@latest
make dev
```

### Execução Direta

```bash
# Execute diretamente com Go
make run
# ou
go run cmd/server/main.go
```

### Build para Produção

```bash
# Build otimizado
make build

# Execute o binário
./bin/pr-documentator
```

### Docker (Opcional)

```bash
# Build da imagem
make docker-build

# Execute o container
make docker-run
```

### Verificação da Instalação

Teste se o servidor está funcionando:

```bash
# Health check
curl -k https://localhost:8443/health

# Resposta esperada:
{
  "status": "healthy",
  "timestamp": "2024-01-10T10:00:00Z",
  "version": "1.0.0"
}
```

## 📡 API Endpoints

### 🩺 Health Check

**Endpoint:** `GET /health`

**Descrição:** Verifica se o serviço está funcionando

**Exemplo:**
```bash
curl -k https://localhost:8443/health
```

**Resposta:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-10T10:00:00Z",
  "version": "1.0.0"
}
```

### 🔍 Analyze Pull Request

**Endpoint:** `POST /analyze-pr`

**Descrição:** Recebe webhooks do GitHub e analisa PRs para mudanças em APIs

**Headers Obrigatórios:**
- `Content-Type: application/json`
- `X-GitHub-Event: pull_request`
- `X-Hub-Signature-256: sha256=...` (se configurado)

**Payload de Exemplo:**
```json
{
  "action": "opened",
  "number": 123,
  "pull_request": {
    "id": 456,
    "number": 123,
    "title": "Add user management endpoints",
    "body": "This PR adds CRUD endpoints for user management",
    "diff_url": "https://github.com/owner/repo/pull/123.diff",
    "patch_url": "https://github.com/owner/repo/pull/123.patch",
    "html_url": "https://github.com/owner/repo/pull/123"
  },
  "repository": {
    "id": 789,
    "name": "my-api",
    "full_name": "owner/my-api",
    "html_url": "https://github.com/owner/my-api"
  }
}
```

**Resposta de Sucesso:**
```json
{
  "status": "success",
  "analysis": {
    "new_routes": [
      {
        "method": "POST",
        "path": "/api/v1/users",
        "description": "Create a new user",
        "parameters": [
          {
            "name": "name",
            "in": "body",
            "type": "string",
            "required": true,
            "description": "User's full name"
          }
        ],
        "request_body": {
          "name": "string",
          "email": "string",
          "role": "string"
        },
        "response": {
          "id": "string",
          "name": "string",
          "email": "string",
          "created_at": "string"
        },
        "headers": [
          {
            "name": "Content-Type",
            "required": true,
            "description": "Must be application/json"
          }
        ]
      }
    ],
    "modified_routes": [],
    "deleted_routes": [],
    "summary": "Added new user creation endpoint with validation",
    "confidence": 0.95,
    "postman_update": {
      "collection_id": "12345-abc-def",
      "status": "success",
      "items_added": 1,
      "items_modified": 0,
      "items_deleted": 0,
      "updated_at": "2024-01-10T10:00:00Z"
    }
  },
  "timestamp": "2024-01-10T10:00:00Z"
}
```

## 📁 Estrutura do Projeto

```
pr-documentator/
├── 📂 cmd/
│   └── 📂 server/
│       └── 📄 main.go              # Entry point - injeção de dependência
├── 📂 internal/                    # Código privado da aplicação
│   ├── 📂 config/
│   │   └── 📄 config.go            # Configuração e variáveis de ambiente
│   ├── 📂 handlers/                # HTTP handlers
│   │   ├── 📄 health.go            # Health check endpoint
│   │   ├── 📄 metrics.go           # Prometheus metrics endpoint
│   │   └── 📄 pr_analyzer.go       # Handler principal de análise
│   ├── 📂 interfaces/              # Contratos e interfaces
│   │   └── 📄 interfaces.go        # Definições de interface para DI
│   ├── 📂 models/                  # Estruturas de dados
│   │   ├── 📄 github.go            # Modelos do GitHub (PR, Repository, etc.)
│   │   ├── 📄 analysis.go          # Modelos de análise e resposta
│   │   └── 📄 postman.go           # Modelos do Postman (Collection, Item, etc.)
│   ├── 📂 services/                # Lógica de negócio
│   │   └── 📄 analyzer.go          # Orquestração da análise
│   └── 📂 middleware/              # Middlewares HTTP
│       ├── 📄 auth.go              # Autenticação e validação de webhook
│       ├── 📄 errors.go            # Tratamento centralizado de erros
│       ├── 📄 logging.go           # Logging de requisições
│       └── 📄 metrics.go           # Middleware de métricas
├── 📂 io/                          # Integrações externas
│   ├── 📂 claude/                  # Integração Claude AI
│   │   ├── 📄 client.go            # Cliente Resty com circuit breaker
│   │   └── 📄 types.go             # Tipos específicos do Claude
│   └── 📂 postman/                 # Integração Postman API
│       ├── 📄 client.go            # Cliente Resty com retry automático
│       └── 📄 types.go             # Tipos específicos do Postman
├── 📂 pkg/                         # Utilitários reutilizáveis
│   ├── 📂 errors/
│   │   └── 📄 errors.go            # Tipos de erro customizados
│   ├── 📂 logger/
│   │   └── 📄 logger.go            # Logger estruturado com zerolog
│   └── 📂 metrics/
│       └── 📄 prometheus.go        # Coletor de métricas Prometheus
├── 📂 scripts/                     # Scripts de automação
│   ├── 📄 generate_certs.sh        # Geração de certificados HTTPS
│   ├── 📄 test_webhook.sh          # Teste de webhook local
│   └── 📄 test_local_development.sh # Suite completa de testes
├── 📂 test/                        # Testes
│   ├── 📂 fixtures/                # Dados de teste (payloads JSON)
│   ├── 📂 mocks/                   # Mocks para testes
│   └── 📂 integration/             # Testes de integração
├── 📂 certs/                       # Certificados SSL (gerados)
├── 📄 .env.example                 # Exemplo de variáveis de ambiente
├── 📄 .air.toml                    # Configuração hot reload
├── 📄 .gitignore                   # Arquivos ignorados pelo Git
├── 📄 go.mod                       # Dependências Go
├── 📄 go.sum                       # Checksums das dependências
├── 📄 Makefile                     # Comandos de automação
├── 📄 CLAUDE.md                    # Documentação para Claude Code
├── 📄 REFACTORING.md               # Documentação da refatoração
└── 📄 README.md                    # Esta documentação
```

### Principais Diretórios

- **`cmd/`**: Pontos de entrada da aplicação com dependency injection
- **`internal/`**: Código específico da aplicação (não importável externamente)
  - **`interfaces/`**: Contratos para dependency injection
  - **`handlers/`**: HTTP handlers com tratamento de erros
  - **`middleware/`**: Stack completo de middleware (auth, metrics, logging)
- **`io/`**: Clientes para APIs externas com circuit breakers
- **`pkg/`**: Utilitários reutilizáveis (errors, logger, metrics)
- **`scripts/`**: Scripts de automação e testes
- **`test/fixtures/`**: Payloads de teste para webhooks

## 🛠️ Desenvolvimento

### Comandos Make Disponíveis

```bash
# 📋 Ver todos os comandos disponíveis
make help

# 🚀 Desenvolvimento
make dev              # Servidor com hot reload
make run              # Executar diretamente
make build            # Build para produção
make build-dev        # Build com símbolos de debug

# 🧪 Testes
make test             # Todos os testes
make test-unit        # Apenas testes unitários
make test-int         # Apenas testes de integração
make test-coverage    # Testes com relatório de cobertura

# 🔍 Qualidade de Código
make lint             # Executar linter
make fmt              # Formatar código

# 📦 Dependências
make deps             # Baixar dependências
make deps-upgrade     # Atualizar dependências

# 🛠️ Utilitários
make clean            # Limpar arquivos gerados
make gen-certs        # Gerar certificados SSL
make install-tools    # Instalar ferramentas de dev
```

### Adicionando Novos Recursos

#### 1. Novo Endpoint HTTP

```go
// 1. Crie o handler em internal/handlers/
package handlers

type NewHandler struct {
    logger *logger.Logger
}

func (h *NewHandler) Handle(w http.ResponseWriter, r *http.Request) {
    // Implementação
}

// 2. Registre no main.go
router.HandleFunc("/new-endpoint", newHandler.Handle)
```

#### 2. Nova Integração Externa

```go
// 1. Crie em io/service/
package service

type Client struct {
    httpClient *http.Client
    config     ServiceConfig
    logger     *logger.Logger
}

func NewClient(cfg ServiceConfig, logger *logger.Logger) *Client {
    return &Client{...}
}

// 2. Configure em internal/config/config.go
type ServiceConfig struct {
    APIKey  string
    BaseURL string
}
```

#### 3. Novo Modelo de Dados

```go
// Adicione em internal/models/
type NewModel struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### Debugging

#### Logs Estruturados

```go
// Diferentes níveis de log
logger.Debug("Mensagem debug", "key", value)
logger.Info("Informação", "user", userID)
logger.Warn("Aviso", "retry_count", 3)
logger.Error("Erro", err, "context", context)
```

#### Visualizando Logs

```bash
# Em desenvolvimento (formato console)
LOG_FORMAT=console make dev

# Em produção (formato JSON)
tail -f logs/app.log | jq .

# Buscar erros
grep ERROR logs/app.log
```

## 🧪 Testes

### Executando Testes

```bash
# Todos os testes
make test

# Apenas testes rápidos
make test-unit

# Testes de integração
make test-int

# Com relatório de cobertura
make test-coverage
```

### Estrutura de Teste

```go
// Exemplo: internal/handlers/health_test.go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/igorsal/pr-documentator/pkg/logger"
)

func TestHealthHandler_Handle(t *testing.T) {
    // Arrange
    logger := logger.New("debug", "console")
    handler := NewHealthHandler(logger)
    
    req := httptest.NewRequest("GET", "/health", nil)
    rec := httptest.NewRecorder()
    
    // Act
    handler.Handle(rec, req)
    
    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Body.String(), "healthy")
}
```

### 🧪 Testando Localmente

#### **Teste Rápido Completo**
```bash
# Execute todos os testes automaticamente
./scripts/test_local_development.sh

# Este script irá:
# ✅ Verificar dependências
# ✅ Configurar environment
# ✅ Compilar o projeto
# ✅ Gerar certificados
# ✅ Executar testes unitários
# ✅ Iniciar servidor
# ✅ Testar health check
# ✅ Simular webhook do GitHub
```

#### **Teste Manual do Webhook**

**1. Inicie o servidor:**
```bash
# Com hot reload
make dev

# Ou diretamente
make build && ./bin/pr-documentator
```

**2. Teste o health check:**
```bash
curl -k https://localhost:8443/health

# Resposta esperada:
# {
#   "status": "healthy",
#   "timestamp": "2024-01-15T10:00:00Z",
#   "version": "1.0.0"
# }
```

**3. Simule um webhook do GitHub:**
```bash
# Usando o script automatizado
./scripts/test_webhook.sh

# Ou usando curl manualmente:
# Calcule a assinatura HMAC
PAYLOAD='{"action":"opened","number":123,...}'
SECRET="test-secret-123"
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

# Envie a requisição
curl -X POST https://localhost:8443/analyze-pr \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d @test/fixtures/github_pr_opened.json \
  -k
```

#### **Exemplos de Payloads de Teste**

**📄 Payload de PR Aberto (`test/fixtures/github_pr_opened.json`):**
```json
{
  "action": "opened",
  "number": 123,
  "pull_request": {
    "id": 1234567890,
    "title": "Add user management API endpoints",
    "body": "This PR adds new REST API endpoints for user management",
    "diff_url": "https://github.com/owner/repo/pull/123.diff",
    "html_url": "https://github.com/owner/repo/pull/123"
  },
  "repository": {
    "full_name": "developer/my-api-project"
  }
}
```

**📄 Payload de PR Atualizado (`test/fixtures/github_pr_synchronize.json`):**
```json
{
  "action": "synchronize",
  "number": 123,
  "pull_request": {
    "title": "Add user management API endpoints",
    "body": "Updated: Added validation and rate limiting"
  }
}
```

#### **Simulando Diferentes Cenários**

**Cenário 1: PR com Novas Rotas**
```bash
# Use o payload padrão que simula adição de endpoints
./scripts/test_webhook.sh test-secret-123 https://localhost:8443
```

**Cenário 2: PR sem Mudanças de API**
```bash
# Crie um payload personalizado para testar PRs sem mudanças de API
cat > test/fixtures/no_api_changes.json << 'EOF'
{
  "action": "opened",
  "pull_request": {
    "title": "Fix typo in documentation",
    "body": "Just fixing a small typo in README",
    "diff_url": "https://github.com/repo/pull/124.diff"
  }
}
EOF

# Teste com este payload
curl -X POST https://localhost:8443/analyze-pr \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -d @test/fixtures/no_api_changes.json \
  -k
```

**Cenário 3: Teste sem Secret (opcional)**
```bash
# Configure sem secret no .env
GITHUB_WEBHOOK_SECRET=""

# Teste sem assinatura
curl -X POST https://localhost:8443/analyze-pr \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -d @test/fixtures/github_pr_opened.json \
  -k
```

#### **Debugging de Testes**

**Ver logs detalhados:**
```bash
# Configure log level para debug
LOG_LEVEL=debug LOG_FORMAT=console make dev

# Em outro terminal, execute o teste
./scripts/test_webhook.sh
```

**Testar componentes individualmente:**
```bash
# Testar apenas compilação
go build -o bin/test cmd/server/main.go

# Testar parsing de payload
go test -v ./internal/handlers -run TestPRAnalyzerHandler

# Testar integração Claude (precisa de API key)
go test -v ./io/claude -run TestAnalyzePR
```

#### **Testando com APIs Reais**

**⚠️ Importante:** Para testes completos, configure suas APIs reais no `.env`:

```env
# Claude API - necessário para análise funcionar
CLAUDE_API_KEY=sk-ant-api03-sua-chave-real

# Postman API - necessário para updates funcionarem  
POSTMAN_API_KEY=PMAK-sua-chave-real
POSTMAN_WORKSPACE_ID=seu-workspace-id
POSTMAN_COLLECTION_ID=sua-collection-id
```

**Teste com APIs configuradas:**
```bash
# Execute o teste completo
./scripts/test_local_development.sh

# Resposta esperada com APIs reais:
# {
#   "status": "success",
#   "analysis": {
#     "new_routes": [...],
#     "confidence": 0.95,
#     "postman_update": {
#       "status": "success",
#       "items_added": 2
#     }
#   }
# }
```

#### **🌐 Testando com Webhooks Reais do GitHub**

Para testar com webhooks reais do GitHub, use o **ngrok** para expor seu servidor local:

**1. Instale o ngrok:**
```bash
# macOS
brew install ngrok

# Linux/Windows - baixe de: https://ngrok.com/download
```

**2. Exponha seu servidor local:**
```bash
# Inicie seu servidor primeiro
make dev

# Em outro terminal, exponha a porta 8443
ngrok http 8443

# Ngrok irá mostrar uma URL pública:
# https://abc123.ngrok.io -> https://localhost:8443
```

**3. Configure o webhook no GitHub:**
```
Repositório → Settings → Webhooks → Add webhook

Payload URL: https://abc123.ngrok.io/analyze-pr
Content type: application/json
Secret: seu-github-webhook-secret (mesmo do .env)
Events: ☑️ Pull requests
Active: ☑️
```

**4. Teste criando um PR real:**
```bash
# No seu repositório de teste:
git checkout -b test-api-changes

# Simule mudanças de API
cat > api/users.js << 'EOF'
// New API endpoint
app.post('/api/v1/users', (req, res) => {
  // Create user logic
  res.json({ id: 1, name: req.body.name });
});
EOF

git add . && git commit -m "Add user creation API endpoint"
git push origin test-api-changes

# Crie um PR no GitHub - o webhook será enviado automaticamente!
```

**5. Monitorar em tempo real:**
```bash
# Terminal 1: Servidor com logs detalhados
LOG_LEVEL=debug LOG_FORMAT=console make dev

# Terminal 2: Interface do ngrok
# Vá para: http://127.0.0.1:4040
# Veja todas as requisições em tempo real

# Terminal 3: Logs específicos de webhook
tail -f logs/webhook.log | grep "PR analysis"
```

**6. Dicas para debugging com ngrok:**
```bash
# Ver histórico de requisições
curl http://127.0.0.1:4040/api/requests/http | jq

# Replay uma requisição
curl -X POST http://127.0.0.1:4040/api/requests/http/{request-id}/replay

# Verificar se webhook chegou
curl -X GET "https://api.github.com/repos/owner/repo/hooks" \
  -H "Authorization: token YOUR_GITHUB_TOKEN"
```

## 🐛 Troubleshooting

### Problemas Comuns

#### ❌ "certificate signed by unknown authority"

**Problema:** Certificado auto-assinado não é confiável

**Soluções:**
```bash
# Opção 1: Ignorar verificação SSL (apenas desenvolvimento)
curl -k https://localhost:8443/health

# Opção 2: Confiar no certificado (macOS)
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./certs/server.crt

# Opção 3: Configurar variável de ambiente
export NODE_TLS_REJECT_UNAUTHORIZED=0
```

#### ❌ "invalid webhook signature"

**Problema:** Assinatura do GitHub webhook não confere

**Verificações:**
1. **Secret correto no `.env`:**
   ```env
   GITHUB_WEBHOOK_SECRET=mesmo-secret-do-github
   ```

2. **Headers corretos:**
   ```bash
   X-GitHub-Event: pull_request
   X-Hub-Signature-256: sha256=...
   ```

3. **Debug da validação:**
   ```bash
   LOG_LEVEL=debug make dev
   # Procure logs de validação de signature
   ```

#### ❌ "rate limit exceeded"

**Problema:** Muitas requisições para APIs externas

**Limites:**
- **Claude API:** 50 req/min (tier gratuito), 1000 req/min (tier pago)
- **Postman API:** 300 req/min

**Soluções:**
- Implementar cache de respostas
- Configurar rate limiting interno
- Upgrade do tier da API

#### ❌ "connection refused" ou "timeout"

**Problema:** Não consegue conectar com APIs externas

**Verificações:**
```bash
# Teste conectividade
curl -I https://api.anthropic.com
curl -I https://api.postman.com

# Verifique proxy corporativo
echo $HTTP_PROXY
echo $HTTPS_PROXY

# Teste DNS
nslookup api.anthropic.com
```

#### ❌ Problemas de Certificados HTTPS

```bash
# Regenere os certificados
rm -rf certs/
make gen-certs

# Verifique permissões
ls -la certs/
# server.key deve ter permissão 600
# server.crt deve ter permissão 644
```

### Logs de Debug

#### Habilitar Logs Detalhados

```bash
# No .env
LOG_LEVEL=debug
LOG_FORMAT=console

# Ou temporariamente
LOG_LEVEL=debug make dev
```

#### Analisando Logs

```bash
# Logs em tempo real
tail -f logs/app.log

# Buscar padrões específicos
grep "PR analysis" logs/app.log
grep -i error logs/app.log

# Com Docker
docker logs pr-documentator --follow
```

### Validação de Configuração

#### Script de Validação

Crie um arquivo `scripts/validate_config.sh`:

```bash
#!/bin/bash
echo "🔍 Validando configuração..."

# Verifica variáveis obrigatórias
required_vars=(
    "CLAUDE_API_KEY"
    "POSTMAN_API_KEY"
    "POSTMAN_WORKSPACE_ID"
    "POSTMAN_COLLECTION_ID"
)

for var in "${required_vars[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "❌ $var não configurada"
        exit 1
    fi
done

# Testa APIs
echo "🔗 Testando Claude API..."
curl -s -H "x-api-key: $CLAUDE_API_KEY" \
     https://api.anthropic.com/v1/models >/dev/null
echo "✅ Claude API OK"

echo "🔗 Testando Postman API..."
curl -s -H "X-API-Key: $POSTMAN_API_KEY" \
     https://api.postman.com/me >/dev/null
echo "✅ Postman API OK"

echo "🎉 Configuração válida!"
```

## 📚 Recursos Adicionais

### Documentação das APIs

- **📘 [Claude API](https://docs.anthropic.com/claude/reference/getting-started-with-the-api)**: Documentação oficial da API do Claude
- **📗 [Postman API](https://www.postman.com/postman/workspace/postman-public-workspace/documentation/12959542-c8142d51-e97c-46b6-bd77-52bb66712c9a)**: Documentação da API do Postman
- **📙 [GitHub Webhooks](https://docs.github.com/en/developers/webhooks-and-events/webhooks)**: Guia de webhooks do GitHub

### Bibliotecas e Frameworks Utilizados

- **🌐 [Gorilla Mux](https://github.com/gorilla/mux)**: Router HTTP robusto
- **📝 [Zerolog](https://github.com/rs/zerolog)**: Logger estruturado de alta performance
- **🔄 [Resty](https://github.com/go-resty/resty)**: Cliente HTTP avançado com retries automáticos
- **⚡ [Circuit Breaker](https://github.com/sony/gobreaker)**: Proteção contra falhas em cascata
- **📊 [Prometheus](https://github.com/prometheus/client_golang)**: Métricas e observabilidade
- **🏗️ [Dependency Injection](https://github.com/igorsal/pr-documentator/tree/main/internal/interfaces)**: Interfaces para arquitetura limpa
- **⚡ [Air](https://github.com/cosmtrek/air)**: Hot reload para desenvolvimento Go
- **🧪 [Testify](https://github.com/stretchr/testify)**: Framework de testes

### Melhores Práticas

#### Segurança
- ✅ Validação de assinaturas de webhook
- ✅ HTTPS obrigatório em produção
- ✅ Logs estruturados sem secrets
- ✅ Rate limiting implementado

#### Performance
- ✅ Cliente HTTP com timeout configurável
- ✅ Context propagation para cancelamento
- ✅ Graceful shutdown
- ✅ Middleware de recovery para panics

#### Monitoramento
- ✅ Health check endpoint
- ✅ Logs estruturados para agregação
- ✅ Métricas de tempo de resposta
- ✅ Alertas baseados em status codes

### Contribuindo

1. **Fork** o repositório
2. **Crie** uma branch para sua feature (`git checkout -b feature/nova-funcionalidade`)
3. **Commit** suas mudanças (`git commit -am 'Adiciona nova funcionalidade'`)
4. **Push** para a branch (`git push origin feature/nova-funcionalidade`)
5. **Abra** um Pull Request

### Roadmap

#### 🚧 Próximas Funcionalidades

- [ ] **Interface Web**: Dashboard para visualizar análises
- [ ] **Múltiplas Coleções**: Suporte a várias coleções Postman
- [ ] **Templates Personalizados**: Templates customizáveis para documentação
- [ ] **Integração Slack**: Notificações em canais do Slack
- [ ] **Cache Redis**: Cache distribuído para melhor performance
- [ ] **Métricas Prometheus**: Integração com Prometheus/Grafana

#### 💡 Ideias Futuras

- Suporte a outros LLMs (OpenAI GPT, Google PaLM)
- Integração com Swagger/OpenAPI
- Análise de breaking changes
- Geração automática de testes

### Licença

Este projeto é licenciado sob a **MIT License**. Veja o arquivo [LICENSE](LICENSE) para detalhes.

---

## 🤝 Suporte

- **🐛 Issues**: [GitHub Issues](https://github.com/igorsal/pr-documentator/issues)
- **💬 Discussões**: [GitHub Discussions](https://github.com/igorsal/pr-documentator/discussions)
- **📧 Email**: suporte@pr-documentator.com

**Desenvolvido com ❤️ usando Go e Claude AI**