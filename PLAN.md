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
3. File-based storage with configurable location (persists across CLI restarts)

---

## Analysis: What Can Be Extracted

### Extractable Components (Low Coupling)

| Package | Purpose | Dependencies | Effort |
|---------|---------|--------------|--------|
| `pkg/storage/file` | File-based email storage | `pkg/extension` (can stub), `pkg/stringutil` | Low |
| `pkg/storage` | Storage interface | Minimal | Low |
| `pkg/message` | Message parsing/metadata | `pkg/storage`, `pkg/policy` | Low |
| `pkg/policy` | Address validation | Minimal | Low |
| `pkg/stringutil` | Mailbox name hashing | Minimal | Low |

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
│       ├── file_store.go   # File-based storage implementation
│       ├── smtp.go         # Minimal SMTP server
│       ├── message.go      # Message types
│       ├── hash.go         # Mailbox name hashing (from stringutil)
│       └── server.go       # Embeddable server entry point
```

### Target API

```go
package inbucket

// Config for the embedded email server
type Config struct {
    SMTPAddr    string // e.g., "127.0.0.1:54325"
    StoragePath string // e.g., "/path/to/emails" or from env var
    MaxMessages int    // Per-mailbox cap (default: 500)
}

// Environment variable for storage path configuration
// SUPABASE_INBUCKET_STORAGE_PATH=/path/to/emails

// Server is an embeddable email testing server
type Server struct {
    config  Config
    store   *FileStore
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
   - `pkg/storage/file/fstore.go` → File-based storage implementation
   - `pkg/storage/file/fmessage.go` → File message struct
   - `pkg/storage/file/mbox.go` → Mailbox directory handling
   - `pkg/stringutil/stringutil.go` → Mailbox name hashing

2. **Remove extension dependency**
   - The `extension.Host` is used for event emission
   - For embedded use, either stub it or remove events entirely

3. **Simplify configuration**
   - Replace `config.Storage` with simple struct
   - Add environment variable support for storage path:
     ```go
     // StoragePath resolution order:
     // 1. Explicit config value
     // 2. SUPABASE_INBUCKET_STORAGE_PATH env var
     // 3. Default: ~/.supabase/inbucket or $SUPABASE_WORKDIR/inbucket
     ```

4. **File storage structure**
   ```
   $STORAGE_PATH/
   └── mail/
       └── {hash[0:3]}/
           └── {hash[0:6]}/
               └── {full_hash}/
                   ├── index.gob          # Message metadata index
                   ├── 20240115T143022-0001.raw  # Raw email
                   └── 20240115T143055-0002.raw  # Raw email
   ```

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
| `pkg/storage/file/fstore.go` | `file_store.go` | Remove extension events, add env var config |
| `pkg/storage/file/fmessage.go` | `message.go` | Keep as-is |
| `pkg/storage/file/mbox.go` | `mbox.go` | Keep as-is |
| `pkg/stringutil/stringutil.go` | `hash.go` | Extract `HashMailboxName` only |
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
  volumes:
    - ./volumes/inbucket:/storage
```

### After (Embedded)
```go
// In supabase start command
emailServer := inbucket.New(inbucket.Config{
    SMTPAddr:    "127.0.0.1:54325",
    StoragePath: os.Getenv("SUPABASE_INBUCKET_STORAGE_PATH"), // or default
    MaxMessages: 500,
})
if err := emailServer.Start(ctx); err != nil {
    return err
}
defer emailServer.Stop()
```

### Environment Variable Configuration
```bash
# Set custom storage location
export SUPABASE_INBUCKET_STORAGE_PATH=/path/to/email/storage

# Or use default locations:
# - Linux/macOS: ~/.supabase/inbucket
# - Or relative to project: .supabase/inbucket
```

---

## Estimated Effort

| Phase | Description | Effort |
|-------|-------------|--------|
| Phase 1 | Extract storage layer | 2-4 hours |
| Phase 2 | SMTP server (using go-smtp) | 3-5 hours |
| Phase 3 | Message parsing | 1-2 hours |
| Phase 4 | Integration API | 1-2 hours |
| Testing | Unit + integration tests | 2-4 hours |
| **Total** | | **9-17 hours** |

*Note: Reduced from original estimate due to simplified scope (no REST API, no retention, no POP3).*

---

## Decisions (Resolved)

| Question | Decision | Impact |
|----------|----------|--------|
| Web UI needed? | ❌ No | Skip `pkg/server/web` entirely |
| POP3 support? | ❌ No | Skip `pkg/server/pop3` entirely |
| Message persistence? | ✅ Yes | Use file-based storage |
| REST API compatibility? | ❌ No | Users read emails from disk directly |
| Retention policy? | ❌ No | Users delete emails manually |

### Simplified Scope

With these decisions, the implementation is minimal:
- **SMTP server** - Accept incoming emails
- **File storage** - Persist to disk
- **No HTTP/REST** - Not needed
- **No POP3** - Not needed
- **No retention scanner** - Not needed

---

## Next Steps

1. ~~Confirm requirements with Supabase CLI team~~ ✅ Done
2. Prototype Phase 1 (storage extraction)
3. Prototype Phase 2 (minimal SMTP with go-smtp)
4. Integrate into Supabase CLI
5. Test with existing Supabase email workflows
