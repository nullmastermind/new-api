# AGENTS.md

Guidelines for AI coding agents working in this repository.

## Build & Development Commands

### Backend (Go)

```bash
# Run backend
go run main.go

# Build with version
go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

# Run all tests
go test ./...

# Run single test file
go test -v ./common/url_validator_test.go ./common/url_validator.go

# Run specific test function
go test -v -run TestValidateRedirectURL ./common/...

# Run tests in a package
go test -v ./relay/common/...
```

### Frontend (React + Vite)

```bash
cd web

# Install dependencies
bun install

# Development server
bun run dev

# Production build
DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../VERSION) bun run build

# Linting
bun run lint          # Prettier check
bun run lint:fix      # Prettier fix
bun run eslint        # ESLint check
bun run eslint:fix    # ESLint fix

# i18n
bun run i18n:extract  # Extract translation keys
bun run i18n:sync     # Sync translations
```

### Docker

```bash
docker-compose up -d  # Start with PostgreSQL + Redis
```

## Code Style Guidelines

### Go

#### Imports
Group imports in this order, separated by blank lines:
1. Standard library
2. Internal packages (`github.com/QuantumNous/new-api/...`)
3. External packages

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/QuantumNous/new-api/common"
    "github.com/QuantumNous/new-api/dto"
    relaycommon "github.com/QuantumNous/new-api/relay/common"  // Use aliases when needed

    "github.com/gin-gonic/gin"
)
```

#### Naming Conventions
- **Packages**: lowercase, single word (`controller`, `model`, `service`)
- **Files**: snake_case (`channel_affinity.go`, `relay_handler.go`)
- **Structs/Interfaces**: PascalCase (`OpenAIModel`, `RelayInfo`, `Adaptor`)
- **Functions**: PascalCase (exported), camelCase (unexported)
- **Constants**: PascalCase (`ChannelStatusEnabled`, `APITypeOpenAI`)

#### Error Handling
- Use `fmt.Errorf` with `%w` for wrapping errors
- Use `errors.New` for simple error messages
- Return errors as the last return value
- Check errors immediately after function calls

```go
if err != nil {
    return nil, fmt.Errorf("failed to process request: %w", err)
}
```

#### Testing Patterns
- Use table-driven tests with `t.Run` for subtests
- Use `t.Helper()` in test helper functions
- Use `t.Cleanup()` for teardown
- Prefer `github.com/stretchr/testify/require` for assertions

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid case", "input", "expected", false},
        {"error case", "bad", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            require.Equal(t, tt.want, got)
        })
    }
}
```

### React/JSX

#### License Header (Required)
All `.js` and `.jsx` files must start with the AGPL-3.0 license header:

```javascript
/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
...
*/
```

#### Imports
Order imports as:
1. React and core libraries
2. Third-party libraries
3. Internal components and helpers
4. Lazy-loaded components

```jsx
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Table } from '@douyinfe/semi-ui';
import { API } from '../helpers';
const LazyComponent = lazy(() => import('./LazyComponent'));
```

#### Naming Conventions
- **Components**: PascalCase (`ChannelsTable`, `EditModal`)
- **Files**: PascalCase for components (`ChannelsTable.jsx`), camelCase for utilities
- **Hooks**: `use` prefix (`useUsersData`, `useTokensData`)
- **Context**: PascalCase with `Context` suffix (`UserContext`, `StatusContext`)

#### Formatting
- Single quotes for strings (configured in Prettier)
- Max 1 empty line between code blocks
- Use `@so1ve/prettier-config` preset

#### i18n
- Use `t('key')` from `useTranslation()` hook
- Fallback language: Chinese (`zh`)
- Supported: zh, en, fr, ru, ja, vi
- Translation files: `web/src/i18n/locales/*.json`

## Architecture Notes

### Relay Adaptor Pattern
Each AI provider implements the `channel.Adaptor` interface in `relay/channel/`:
- `Init`, `GetRequestURL`, `SetupRequestHeader`
- `ConvertOpenAIRequest`, `DoRequest`, `DoResponse`
- `GetModelList`, `GetChannelName`

### Key Directories
| Directory | Purpose |
|-----------|---------|
| `controller/` | HTTP handlers |
| `model/` | GORM database models |
| `service/` | Business logic |
| `relay/channel/` | Provider-specific adaptors |
| `dto/` | Data transfer objects |
| `web/src/pages/` | React page components |
| `web/src/components/` | Reusable React components |

### Database
- Supports SQLite (default), MySQL (≥5.7.8), PostgreSQL (≥9.6)
- Column escaping: PostgreSQL uses `"group"`, MySQL uses `` `group` ``
