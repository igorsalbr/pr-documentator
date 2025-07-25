# PR Documentator

A Go service that automatically analyzes GitHub Pull Requests using Claude AI to detect API changes and updates Postman documentation.

## 🎯 How it Works

1. **GitHub Webhook** → Receives PR events via HTTPS
2. **Claude AI Analysis** → Identifies new, modified, or deleted API routes  
3. **Postman Update** → Automatically updates collection documentation
4. **Manual Analysis** → Direct diff analysis via `/manual-analyze` endpoint

```
┌─────────────┐    HTTPS     ┌──────────────┐    AI Analysis    ┌─────────────┐
│   GitHub    │──────────────▶│ PR Analyzer  │───────────────────▶│   Claude    │
│  Webhook    │               │   Service    │                   │     API     │
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

## 🚀 Quick Start

### 1. Setup

```bash
git clone https://github.com/igorsal/pr-documentator.git
cd pr-documentator

# Install dependencies
go mod download

# Generate certificates
./scripts/generate_certs.sh
```

### 2. Configuration

Create a `.env` file:

```env
# API Keys
CLAUDE_API_KEY=sk-ant-api03-your-key-here
POSTMAN_API_KEY=PMAK-your-key-here
POSTMAN_WORKSPACE_ID=your-workspace-id
POSTMAN_COLLECTION_ID=your-collection-id

# Optional
GITHUB_WEBHOOK_SECRET=your-webhook-secret
SERVER_PORT=8443
LOG_LEVEL=info
```

### 3. Run

```bash
# Development with hot reload
make dev

# Or build and run
make build && ./bin/server
```

### 4. Test

```bash
# Health check
curl -k https://localhost:8443/health

# Manual analysis
curl -X POST https://localhost:8443/manual-analyze \
  -H "Content-Type: application/json" \
  -d '{"diff": "your-git-diff-content-here"}' \
  -k
```

## 📡 API Endpoints

### Health Check
- **GET** `/health` - Service status
- **GET** `/metrics` - Prometheus metrics  

### Analysis
- **POST** `/analyze-pr` - GitHub webhook endpoint (requires webhook signature)
- **POST** `/manual-analyze` - Manual diff analysis (public)

**Manual Analysis Example:**
```bash
curl -X POST https://localhost:8443/manual-analyze \
  -H "Content-Type: application/json" \
  -d '{
    "diff": "+app.post(\"/api/v1/users\", (req, res) => {\n+  res.json({id: 1, name: req.body.name});\n+});"
  }' \
  -k
```

**Response:**
```json
{
  "new_routes": [
    {
      "method": "POST",
      "path": "{{baseUrl}}/api/v1/users",
      "description": "Create a new user",
      "request_body": {"name": "string"},
      "response": {"id": "number", "name": "string"}
    }
  ],
  "modified_routes": [],
  "deleted_routes": [],
  "summary": "Added user creation endpoint",
  "confidence": 0.95
}
```

## 🔧 GitHub Webhook Setup

1. Go to your repository **Settings** → **Webhooks**
2. Add webhook:
   - **URL**: `https://your-domain.com:8443/analyze-pr`
   - **Content type**: `application/json`
   - **Secret**: Your webhook secret from `.env`
   - **Events**: Select "Pull requests"

## 🏗️ Project Structure

```
pr-documentator/
├── api/                    # HTTP layer
│   ├── handlers/          # Request handlers
│   └── middleware/        # HTTP middleware
├── cmd/server/            # Application entry point
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── interfaces/       # Dependency injection contracts
│   ├── models/           # Data structures
│   └── services/         # Business logic
├── io/                   # External integrations
│   ├── claude/           # Claude AI client
│   └── postman/          # Postman API client
├── pkg/                  # Reusable utilities
│   ├── errors/           # Error handling
│   ├── logger/           # Structured logging
│   └── metrics/          # Prometheus metrics
└── .vscode/              # VS Code configuration
```

## 💻 Development

### VS Code Integration

The project includes VS Code configuration for easy development:

- **F5** to run with debugger
- **Ctrl+Shift+P** → "Tasks: Run Task" → "run-local"
- Automatic certificate generation
- Debug environment variables configured

### Available Commands

```bash
make dev              # Hot reload development
make build            # Production build
make test             # Run tests
make lint             # Code linting
make clean            # Clean generated files
```

### API Keys Setup

**Claude API:**
1. Sign up at [console.anthropic.com](https://console.anthropic.com)
2. Create API key (starts with `sk-ant-api03-`)

**Postman API:**
1. Go to [postman.com](https://postman.com) → Account Settings → API Keys
2. Generate key (starts with `PMAK-`)
3. Get Workspace ID from URL: `https://app.postman.com/workspace/YOUR-WORKSPACE-ID`
4. Get Collection ID from collection info panel

## 🔍 Key Features

- **Zero Dependencies**: Removed resty, godotenv, air - uses native `net/http`
- **Modern Go**: Uses `signal.NotifyContext`, `any` instead of `interface{}`
- **Circuit Breaker**: Protection against external API failures
- **Structured Logging**: JSON logs with zerolog
- **Metrics**: Prometheus metrics for observability
- **Security**: HMAC webhook validation, HTTPS-only
- **Clean Architecture**: Dependency injection with interfaces
- **Manual Analysis**: Analyze diffs without GitHub webhooks

## 🚨 Important Notes

- **HTTPS Required**: Service runs on HTTPS with self-signed certificates
- **Claude API**: Required for analysis functionality
- **Postman API**: Required for automatic documentation updates
- **Manual Endpoint**: Public endpoint for direct diff analysis (`/manual-analyze`)

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.