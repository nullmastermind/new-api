# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

New API is a next-generation LLM gateway and AI asset management system, forked from [One API](https://github.com/songquanpeng/one-api). It provides a unified API interface for multiple AI providers (OpenAI, Claude, Gemini, etc.) with features like quota management, billing, rate limiting, and multi-language support.

## Build & Development Commands

### Backend (Go)
```bash
# Run backend directly
go run main.go

# Build backend
go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

# Run with flags
./new-api --port 3000 --log-dir ./logs
```

### Frontend (React + Vite)
```bash
cd web

# Install dependencies (uses Bun)
bun install

# Development server (proxies /api, /mj, /pg to localhost:3000)
bun run dev

# Production build
DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build

# Linting
bun run lint          # Check with Prettier
bun run lint:fix      # Fix with Prettier
bun run eslint        # ESLint check
bun run eslint:fix    # ESLint fix

# i18n commands
bun run i18n:extract  # Extract translation keys
bun run i18n:sync     # Sync translations
```

### Full Build (Makefile)
```bash
make all              # Build frontend + start backend
make build-frontend   # Build frontend only
make start-backend    # Start backend only
```

### Docker
```bash
docker-compose up -d  # Start with PostgreSQL + Redis
```

## Architecture

### Backend Structure (Go + Gin)
- **`main.go`**: Entry point, initializes resources, sets up routes and background tasks
- **`router/`**: Route definitions (API, relay, dashboard, web, video)
- **`controller/`**: HTTP handlers for all endpoints
- **`relay/`**: Core relay system for proxying requests to AI providers
  - **`relay/channel/`**: Provider-specific adaptors (openai, claude, gemini, ali, aws, etc.)
  - **`relay/helper/`**: Streaming, pricing, model mapping utilities
- **`model/`**: GORM database models and queries
- **`middleware/`**: Auth, rate limiting, CORS, distributor (channel selection)
- **`service/`**: Business logic (quota, channel selection, tokenization)
- **`setting/`**: Configuration management (ratio_setting, operation_setting, model_setting)
- **`constant/`**: Constants and environment variable definitions
- **`dto/`**: Data transfer objects for API requests/responses

### Frontend Structure (React + Semi UI)
- **`web/src/pages/`**: Page components (Channel, Token, Log, Playground, etc.)
- **`web/src/components/`**: Reusable components
- **`web/src/context/`**: React contexts (User, Status, Theme)
- **`web/src/i18n/`**: Internationalization (zh, en, fr, ru, ja, vi)
- **`web/src/helpers/`**: API utilities and helpers

### Relay Adaptor Pattern
Each AI provider has an adaptor implementing the `channel.Adaptor` interface:
```go
type Adaptor interface {
    Init(info *RelayInfo)
    GetRequestURL(info *RelayInfo) (string, error)
    SetupRequestHeader(c *gin.Context, req *http.Request, info *RelayInfo) error
    ConvertOpenAIRequest(c *gin.Context, info *RelayInfo, request *dto.GeneralOpenAIRequest) (any, error)
    DoRequest(c *gin.Context, info *RelayInfo, requestBody io.Reader) (any, error)
    DoResponse(c *gin.Context, resp *http.Response, info *RelayInfo) (usage any, err *types.NewAPIError)
    GetModelList() []string
    GetChannelName() string
}
```

## Special Patterns & Logic

### Model Name Suffix Handling
Models support reasoning effort suffixes that are parsed and stripped:
- `-high`, `-medium`, `-low`, `-minimal`, `-none`, `-xhigh` for OpenAI o-series/gpt-5
- `-thinking`, `-nothinking`, `-thinking-<budget>` for Gemini models
- `-search` suffix for xAI models enables web search

```go
// Example: "o3-mini-high" → effort="high", model="o3-mini"
// Example: "gemini-2.5-flash-thinking-128" → enables thinking with 128 token budget
```

### Token Authentication
Tokens use format `sk-<key>[-<channel_id>][-<group>]`:
- Key is validated against database
- Optional channel ID forces specific channel
- Optional group overrides user's default group
- WebSocket auth via `Sec-WebSocket-Protocol: openai-insecure-api-key.sk-xxx`

### Quota & Billing System
- Pre-consumption: Deducts estimated quota before request, refunds difference after
- Trust quota: Users with sufficient quota skip pre-consumption
- Batch updates: Optional async quota updates for performance (`BATCH_UPDATE_ENABLED=true`)
- Model pricing: Per-token ratios with completion/cache/audio multipliers

### Channel Selection & Retry
- Channels selected by group + model + priority + weight (weighted random)
- Multi-key channels support random or polling key selection
- Auto-retry on failure with configurable `RetryTimes`
- Auto-ban channels exceeding error threshold

### Streaming & SSE
- `StreamScannerHandler`: Handles SSE with configurable buffer (default 64MB)
- `STREAMING_TIMEOUT`: No-data timeout (default 300s)
- Ping keep-alive for long-running streams
- Custom `CustomEvent` renderer for SSE format

### Database Support
- SQLite (default), MySQL (≥5.7.8), PostgreSQL (≥9.6)
- Separate log database supported via `LOG_SQL_DSN`
- Column name escaping differs: PostgreSQL uses `"group"`, MySQL uses `` `group` ``

### Master/Slave Node Architecture
- `NODE_TYPE=master` (default): Runs all background tasks (sync, channel testing, batch updates)
- `NODE_TYPE=slave`: Only handles requests, no background tasks
- `SYNC_FREQUENCY`: Interval for syncing options from database (default 60s)

## Key Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SQL_DSN` | Database connection string | SQLite `./new-api.db` |
| `REDIS_CONN_STRING` | Redis connection (required for rate limiting) | - |
| `SESSION_SECRET` | Session encryption key | Random |
| `CRYPTO_SECRET` | Encryption key for sensitive data | - |
| `STREAMING_TIMEOUT` | SSE no-data timeout (seconds) | 300 |
| `RELAY_TIMEOUT` | Non-streaming request timeout (seconds) | 600 |
| `BATCH_UPDATE_ENABLED` | Enable async quota updates | false |
| `BATCH_UPDATE_INTERVAL` | Batch update interval (seconds) | 5 |
| `MEMORY_CACHE_ENABLED` | Enable in-memory caching | false |
| `TIKTOKEN_CACHE_DIR` | Directory for tiktoken cache | - |
| `NODE_TYPE` | master/slave node type | master |

## API Types (Provider Constants)

Key provider types in `relay/constant/api_type.go`:
- `APITypeOpenAI = 1`: OpenAI and compatible APIs
- `APITypeAnthropic = 14`: Claude/Anthropic
- `APITypeGemini = 24`: Google Gemini
- `APITypeAws = 33`: AWS Bedrock
- `APITypeAzure = 3`: Azure OpenAI
- `APITypeAli = 15`: Alibaba Qwen
- `APITypeZhipu = 18`: Zhipu GLM
- `APITypeBaidu = 17`: Baidu ERNIE

## License Headers

All source files require AGPL-3.0 license headers. Frontend uses ESLint plugin `eslint-plugin-header` to enforce this. The header format is defined in `web/.eslintrc.cjs`.

## i18n

- Fallback language: Chinese (`zh`)
- Supported: zh, en, fr, ru, ja, vi
- Translation files: `web/src/i18n/locales/*.json`
- Use `t('key')` from `useTranslation()` hook
- Extract new keys: `bun run i18n:extract`

