# AGENTS.md - Inbucket

Guidance for AI agents working in this codebase.

## Project Overview

Inbucket is an email testing service that accepts messages for any email address and makes them available via web, REST, and POP3 interfaces. It's a self-contained Go application with an Elm-based web UI.

**Tech Stack:**
- Backend: Go 1.25+
- Frontend: Elm 0.19.1 with Parcel bundler
- Logging: zerolog
- Testing: testify (assert/require/suite), goldiff for golden file tests
- HTTP Router: gorilla/mux
- Configuration: envconfig (environment variables)
- Optional: Lua scripting for extensions (gopher-lua)

## Essential Commands

### Build

```bash
# Build Go binaries (inbucket server + client CLI)
make build

# Or build directly
go build ./cmd/inbucket
go build ./cmd/client

# Build UI (required before running server)
cd ui && yarn install && yarn build
```

### Test

```bash
# Run all Go tests with race detection
make test
# or
go test -race ./...

# Run tests for a specific package
go test -race ./pkg/storage/...

# Run tests with coverage
go test -race -coverprofile=profile.cov ./...
```

### Lint

```bash
# CI uses golangci-lint
golangci-lint run

# Make's lint target (older, uses golint)
make lint
```

### Run Development Server

```bash
# Build everything first
make build
cd ui && yarn build && cd ..

# Run with dev config
./etc/dev-start.sh

# Or run directly with defaults
./inbucket
```

Default ports:
- Web UI: http://localhost:9000
- SMTP: localhost:2500
- POP3: localhost:1100

### UI Development

```bash
cd ui

# Install dependencies
yarn install

# Development server with HMR (proxies to Go backend)
yarn start

# Production build
yarn build

# Clean build artifacts
yarn clean
```

## Code Organization

```
cmd/
  inbucket/           # Main server binary
  client/             # CLI client for REST API

pkg/
  config/             # Environment-based configuration
  extension/          # Lua extension system
    luahost/          # Lua VM pool and bindings
    event/            # Extension event types
  message/            # Message manager (storage abstraction)
  metric/             # Expvar metrics
  msghub/             # Real-time message pub/sub
  policy/             # Email address/domain policies
  rest/               # REST API v1/v2 controllers
    client/           # Go client library for REST API
    model/            # JSON API models
  server/
    smtp/             # SMTP server
    pop3/             # POP3 server
    web/              # HTTP server, handlers, helpers
  storage/            # Storage interface and implementations
    file/             # File-based storage
    mem/              # In-memory storage
  stringutil/         # String utilities
  test/               # Test utilities and integration tests
  webui/              # Web UI controllers

ui/
  src/
    Main.elm          # Elm app entry point
    Api.elm           # API client
    Page/             # Page modules (Home, Mailbox, Monitor, Status)
    Data/             # Data models
  tests/              # Elm tests
```

## Configuration

Inbucket uses environment variables for all configuration. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `INBUCKET_LOGLEVEL` | `info` | debug, info, warn, error |
| `INBUCKET_MAILBOXNAMING` | `local` | local, full, or domain |
| `INBUCKET_SMTP_ADDR` | `0.0.0.0:2500` | SMTP listen address |
| `INBUCKET_WEB_ADDR` | `0.0.0.0:9000` | HTTP listen address |
| `INBUCKET_POP3_ADDR` | `0.0.0.0:1100` | POP3 listen address |
| `INBUCKET_STORAGE_TYPE` | `memory` | `memory` or `file` |
| `INBUCKET_WEB_UIDIR` | `ui/dist` | Path to built UI files |

Run `./inbucket -help` for complete list.

See `doc/config.md` for detailed documentation.

## Code Patterns

### Error Handling
- Use zerolog for structured logging
- Return errors up the call stack; log at the top level
- Use `github.com/pkg/errors` patterns for wrapping

### HTTP Handlers
Handlers follow this pattern in `pkg/server/web/`:
```go
func Handler(f func(http.ResponseWriter, *http.Request, *Context) error) http.Handler
```

Controllers return errors; the wrapper handles HTTP responses.

### Storage Interface
New storage backends implement `storage.Store` interface (`pkg/storage/storage.go`):
```go
type Store interface {
    AddMessage(message Message) (id string, err error)
    GetMessage(mailbox, id string) (Message, error)
    GetMessages(mailbox string) ([]Message, error)
    MarkSeen(mailbox, id string) error
    PurgeMessages(mailbox string) error
    RemoveMessage(mailbox, id string) error
    VisitMailboxes(f func([]Message) (cont bool)) error
}
```

Register in `cmd/inbucket/main.go` init():
```go
storage.Constructors["mytype"] = mystore.New
```

### JSON Tag Convention
JSON fields use kebab-case (configured in `.golangci.yml` tagliatelle):
```go
type Example struct {
    FieldName string `json:"field-name"`
}
```

### Elm Architecture
The UI follows The Elm Architecture:
- `Main.elm` - App shell, routing
- `Page/*.elm` - Page modules with Model, Msg, init, update, view
- `Data/*.elm` - Data types and JSON decoders
- `Api.elm` - HTTP client for REST API

## Testing

### Test Structure
- Unit tests: alongside source files (`*_test.go`)
- Integration tests: `pkg/test/integration_test.go`
- Test utilities: `pkg/test/`

### Test Frameworks
- Standard `testing` package
- `github.com/stretchr/testify/assert` - assertions
- `github.com/stretchr/testify/require` - fatal assertions  
- `github.com/stretchr/testify/suite` - test suites
- `github.com/jhillyerd/goldiff` - golden file testing

### Test Utilities
Located in `pkg/test/`:
- `StoreStub`, `ManagerStub` - mock implementations
- `DeliverToStore()` - create test messages
- `StoreSuite()` - table-driven storage tests
- `NewLuaState()` - Lua testing helper

### Golden File Tests
Input in `pkg/test/testdata/*.txt`, expected output in `*.golden`:
```go
goldiff.File(t, got, "testdata", "basic.golden")
```

### Running Specific Tests
```bash
# Run tests matching pattern
go test -race -run TestIntegration ./pkg/test/

# Run with verbose output
go test -race -v ./pkg/storage/mem/

# Run storage suite for specific implementation
go test -race -run TestMemStore ./pkg/storage/mem/
```

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- `build-and-test.yml` - Build and test on Linux/Windows, coverage to coveralls
- `lint.yml` - golangci-lint
- `docker-build.yml` - Docker image builds
- `release.yml` - goreleaser for releases

## Important Gotchas

1. **UI must be built before running server** - The Go server serves static files from `ui/dist/`

2. **Storage type affects persistence** - `memory` storage loses all data on restart; use `file` for persistence

3. **Port conflicts** - Default ports (9000, 2500, 1100) may conflict with other services

4. **Lua scripting is optional** - If `inbucket.lua` is not present, the server runs without extensions

5. **Test coverage requires race detector** - CI always runs with `-race`

6. **golangci-lint v2 config** - Uses v2 format in `.golangci.yml`

7. **Windows paths in storage** - Use `$` instead of `:` in file storage paths (e.g., `D$/inbucket`)

## REST API

Base URL: `http://localhost:9000/api/`

### API v1 Endpoints
- `GET /v1/mailbox/{name}` - List messages
- `GET /v1/mailbox/{name}/{id}` - Get message
- `PATCH /v1/mailbox/{name}/{id}` - Mark as seen
- `DELETE /v1/mailbox/{name}` - Purge mailbox
- `DELETE /v1/mailbox/{name}/{id}` - Delete message
- `GET /v1/mailbox/{name}/{id}/source` - Get raw source

### API v2 Endpoints
- `GET /v2/monitor/messages` - WebSocket for real-time messages

Go client available: `github.com/inbucket/inbucket/v3/pkg/rest/client`

## Development Tips

1. **Quick iteration** - Use `make reflex` for auto-rebuild on Go file changes

2. **UI development** - Run `yarn start` in `ui/` for HMR; it proxies API requests to the Go server

3. **Debug network** - Run with `-netdebug` flag to dump SMTP/POP3 traffic

4. **Test email sending** - Use swaks or the test scripts in `etc/swaks-tests/`

5. **Check configuration** - Run `./inbucket -help` to see all env vars and defaults
