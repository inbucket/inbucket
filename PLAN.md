# Plan: Extract Inbucket for Supabase CLI Integration

## Overview

This document outlines a plan to extract minimal components from Inbucket so they can be used directly within the Supabase CLI, eliminating the need for a separate Docker container.

## Current State

### What Inbucket Is
- An email testing service (SMTP/POP3/Web UI)
- Written in Go (`github.com/inbucket/inbucket/v3`)
- Designed as a standalone daemon, not an embeddable library

### Current Supabase Usage (Assumed)
- Runs Inbucket as a Docker container
- Uses it for local email testing during `supabase start`
- Communicates via SMTP (sending) and REST API (reading)

### Problem
- Docker container adds overhead and complexity
- Users need Docker running to use email testing features
- Slower startup times

## Goal

Embed a minimal email testing capability directly in the Supabase CLI binary, providing:
1. SMTP server to receive emails
2. API/interface to read received emails
3. In-memory storage (no persistence needed for local dev)

---

## Analysis: What Can Be Extracted

### Extractable Components (Low Coupling)

| Package | Purpose | Dependencies | Effort |
|---------|---------|--------------|--------|
| `pkg/storage/mem` | In-memory email storage | `pkg/extension` (can stub) | Low |
| `pkg/storage` | Storage interface | Minimal | Low |
| `pkg/message` | Message parsing/metadata | `pkg/storage`, `pkg/policy` | Low |
| `pkg/policy` | Address validation | Minimal | Low |
| `pkg/rest/client` | REST client | None | N/A (not needed) |

### Non-Extractable (Tight Coupling)

| Package | Issue |
|---------|-------|
| `pkg/server/smtp` | Depends on full server assembly, msghub, config |
| `pkg/server/web` | Gorilla mux, templates, full lifecycle |
| `pkg/server/pop3` | Full server assembly |

---

## Implementation Options

### Option A: Fork & Refactor Inbucket (Recommended)

Create a minimal embeddable version by extracting and refactoring core components.

**Pros:**
- Full control over the API
- Minimal binary size increase
- No external process management

**Cons:**
- Maintenance burden (tracking upstream changes)
- Initial development effort

### Option B: Bundle Inbucket Binary

Compile and bundle the inbucket binary, spawn as subprocess.

**Pros:**
- No code changes to inbucket
- Easy to update

**Cons:**
- Still managing an external process
- Larger binary size
- Platform-specific binaries needed

### Option C: Upstream Contribution

Contribute embeddable API to upstream inbucket project.

**Pros:**
- Community maintained
- Benefits other users

**Cons:**
- Slower timeline (requires upstream approval)
- May not align with upstream goals

---

## Recommended Approach: Option A (Fork & Extract)

### Architecture

```
supabase-cli
├── internal/
│   └── inbucket/           # Extracted/adapted inbucket code
│       ├── storage.go      # Storage interface
│       ├── mem_store.go    # In-memory implementation
│       ├── smtp.go         # Minimal SMTP server
│       ├── message.go      # Message types
│       └── server.go       # Embeddable server entry point
```

### Target API

```go
package inbucket

// Config for the embedded email server
type Config struct {
    SMTPAddr    string // e.g., "127.0.0.1:54325"
    MaxMessages int    // Per-mailbox cap
    MaxSizeKB   int64  // Total storage cap
}

// Server is an embeddable email testing server
type Server struct {
    config  Config
    store   *MemStore
    smtp    *SMTPServer
}

// New creates a new embedded inbucket server
func New(cfg Config) *Server

// Start begins listening for SMTP connections
func (s *Server) Start(ctx context.Context) error

// Stop gracefully shuts down the server
func (s *Server) Stop() error

// --- Message Access API ---

// ListMailboxes returns all mailboxes with messages
func (s *Server) ListMailboxes() []string

// GetMessages returns all messages in a mailbox
func (s *Server) GetMessages(mailbox string) ([]Message, error)

// GetMessage returns a specific message
func (s *Server) GetMessage(mailbox, id string) (*Message, error)

// DeleteMessage removes a message
func (s *Server) DeleteMessage(mailbox, id string) error

// PurgeMailbox deletes all messages in a mailbox
func (s *Server) PurgeMailbox(mailbox string) error

// --- Message Types ---

type Message struct {
    ID      string
    From    string
    To      []string
    Subject string
    Date    time.Time
    Size    int64
    Seen    bool
    Body    string      // Plain text body
    HTML    string      // HTML body
    Raw     []byte      // Raw RFC 5322 message
}
```

---

## Implementation Steps

### Phase 1: Extract Storage Layer

1. **Copy minimal storage code**
   - `pkg/storage/storage.go` → Interface definitions
   - `pkg/storage/mem/store.go` → In-memory implementation
   - `pkg/storage/mem/message.go` → Message struct

2. **Remove extension dependency**
   - The `extension.Host` is used for event emission
   - For embedded use, either stub it or remove events entirely

3. **Simplify configuration**
   - Replace `config.Storage` with simple struct
   - Remove environment variable parsing

### Phase 2: Extract/Adapt SMTP Server

The existing SMTP server in `pkg/server/smtp` uses:
- `github.com/inbucket/inbucket/v3/pkg/msghub` (message broadcasting)
- `github.com/inbucket/inbucket/v3/pkg/message` (delivery manager)
- Full server lifecycle management

**Options:**

#### Option 2A: Adapt Existing SMTP Code
- Extract `pkg/server/smtp/server.go` and `handler.go`
- Remove msghub dependency (not needed without WebSocket UI)
- Simplify to directly write to storage

#### Option 2B: Use a Simpler SMTP Library
- Use `github.com/emersion/go-smtp` directly
- Write a minimal handler that stores to our extracted storage
- Less code to maintain, cleaner integration

**Recommendation: Option 2B** - The go-smtp library is already a dependency of inbucket and provides a clean interface.

### Phase 3: Message Parsing

Extract from `pkg/message/`:
- `delivery.go` - Message delivery logic
- `metadata.go` - Parsing headers, extracting body

Simplify to remove policy/addressing complexity if not needed.

### Phase 4: Integration API

Create the high-level `Server` type that:
1. Initializes storage
2. Starts SMTP listener
3. Exposes message access methods
4. Handles graceful shutdown

---

## Files to Extract/Adapt

| Source File | Target | Changes Needed |
|-------------|--------|----------------|
| `pkg/storage/storage.go` | `storage.go` | Remove `FromConfig`, simplify |
| `pkg/storage/mem/store.go` | `mem_store.go` | Remove extension events, simplify config |
| `pkg/storage/mem/message.go` | `message.go` | Keep as-is |
| `pkg/message/delivery.go` | `delivery.go` | Simplify, remove policy |
| `pkg/server/smtp/` | `smtp.go` | Heavy refactor or rewrite with go-smtp |

## Dependencies to Include

```go
require (
    github.com/emersion/go-smtp    // SMTP server
    github.com/emersion/go-sasl    // SMTP auth (optional)
)
```

---

## Testing Strategy

1. **Unit tests** for storage operations
2. **Integration test** sending email via SMTP and reading it back
3. **Compatibility test** with existing Supabase email testing workflows

---

## Migration Path for Supabase CLI

### Before (Docker-based)
```yaml
# docker-compose.yml
inbucket:
  image: inbucket/inbucket
  ports:
    - "54324:9000"   # Web UI
    - "54325:2500"   # SMTP
```

### After (Embedded)
```go
// In supabase start command
emailServer := inbucket.New(inbucket.Config{
    SMTPAddr:    "127.0.0.1:54325",
    MaxMessages: 100,
})
if err := emailServer.Start(ctx); err != nil {
    return err
}
defer emailServer.Stop()
```

---

## Estimated Effort

| Phase | Description | Effort |
|-------|-------------|--------|
| Phase 1 | Extract storage layer | 2-4 hours |
| Phase 2 | SMTP server (using go-smtp) | 4-8 hours |
| Phase 3 | Message parsing | 2-4 hours |
| Phase 4 | Integration API | 2-4 hours |
| Testing | Unit + integration tests | 4-8 hours |
| **Total** | | **14-28 hours** |

---

## Open Questions

1. **Does Supabase CLI need the Web UI?**
   - If yes, this approach won't work (UI is tightly coupled)
   - If no, we only need SMTP + programmatic access

2. **POP3 support needed?**
   - Likely not for local dev testing
   - Can be omitted to simplify

3. **Message persistence across CLI restarts?**
   - Probably not needed for local dev
   - In-memory storage is sufficient

4. **REST API compatibility?**
   - Does Supabase have tooling that calls inbucket's REST API?
   - If yes, we may need to expose compatible endpoints

---

## Next Steps

1. Confirm requirements with Supabase CLI team
2. Prototype Phase 1 (storage extraction)
3. Prototype Phase 2 (minimal SMTP with go-smtp)
4. Evaluate binary size impact
5. Decide on final approach
