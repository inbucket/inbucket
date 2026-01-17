# Plan: Embedded Email Testing for Supabase CLI

## Goal

Implement a minimal embedded email testing capability directly in the Supabase CLI, allowing developers to:
1. Receive emails via SMTP (for auth confirmations, password resets, etc.)
2. Read received emails from disk
3. No Docker container required

## What Inbucket Uses

### Core Packages (from go.mod)

| Package | Purpose | Used For |
|---------|---------|----------|
| `github.com/jhillyerd/enmime/v2` | MIME parsing | Parsing email headers, body, attachments |
| `github.com/rs/zerolog` | Logging | Structured logging |
| `github.com/kelseyhightower/envconfig` | Config | Environment variable parsing |
| `github.com/gorilla/mux` | HTTP routing | Web UI (not needed) |
| `github.com/gorilla/websocket` | WebSocket | Real-time updates (not needed) |
| `github.com/yuin/gopher-lua` | Lua scripting | Extensions (not needed) |

### Custom Implementations

| Component | Implementation |
|-----------|---------------|
| SMTP Server | Custom, using `net` + `net/textproto` (~700 lines) |
| File Storage | Custom, using `encoding/gob` for index (~400 lines) |
| Message Hub | Custom pub/sub for WebSocket (not needed) |

---

## Recommended Packages for Supabase CLI

### Option 1: Minimal Dependencies (Recommended)

Use well-maintained, focused libraries instead of extracting from inbucket.

#### SMTP Server

**`github.com/emersion/go-smtp`** ⭐ Recommended

```go
import "github.com/emersion/go-smtp"
```

| Aspect | Details |
|--------|---------|
| Stars | 1.5k+ |
| Maintenance | Active, well-maintained |
| API | Clean backend interface |
| Features | AUTH, STARTTLS, size limits |
| Lines to implement | ~100-150 for basic handler |

Example minimal implementation:
```go
type Backend struct {
    storePath string
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
    return &Session{storePath: b.storePath}, nil
}

type Session struct {
    storePath string
    from      string
    to        []string
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
    s.from = from
    return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
    s.to = append(s.to, to)
    return nil
}

func (s *Session) Data(r io.Reader) error {
    // Read email and save to disk
    data, _ := io.ReadAll(r)
    return saveEmail(s.storePath, s.from, s.to, data)
}

func (s *Session) Reset() { s.from = ""; s.to = nil }
func (s *Session) Logout() error { return nil }
```

**Alternative: `github.com/mhale/smtpd`**
- Simpler API
- Less features (no AUTH/TLS built-in)
- Good for very basic needs

#### Email Parsing

**`github.com/jhillyerd/enmime`** ⭐ Recommended

Same library inbucket uses. Excellent MIME parsing.

```go
import "github.com/jhillyerd/enmime/v2"

env, err := enmime.ReadEnvelope(reader)
// env.GetHeader("Subject")
// env.Text (plain text body)
// env.HTML (HTML body)
// env.Attachments
```

**Alternative: `github.com/DusanKasan/parsemail`**
- Simpler API
- Fewer features
- Good for basic email reading

**Alternative: Standard library `net/mail`**
- No external dependency
- Basic header parsing only
- Would need manual body handling

#### Storage

**Option A: Simple .eml files** ⭐ Recommended for simplicity

```
$STORAGE_PATH/
└── {mailbox}/
    ├── 1705312822-0001.eml    # Unix timestamp + counter
    ├── 1705312855-0002.eml
    └── 1705312890-0003.eml
```

Pros:
- Human-readable (can open in email client)
- No index file needed
- Simple to implement (~50 lines)

Cons:
- Need to parse each file to get metadata (slightly slower listing)

**Option B: .eml files + JSON index**

```
$STORAGE_PATH/
└── {mailbox}/
    ├── index.json             # Metadata cache
    ├── 1705312822-0001.eml
    └── 1705312855-0002.eml
```

index.json:
```json
[
  {"id": "1705312822-0001", "from": "noreply@example.com", "subject": "Confirm", "date": "..."},
  {"id": "1705312855-0002", "from": "noreply@example.com", "subject": "Reset", "date": "..."}
]
```

Pros:
- Fast listing without parsing each file
- Still human-readable emails

Cons:
- Need to keep index in sync

**Option C: SQLite (pure Go)**

```go
import "modernc.org/sqlite"  // Pure Go, no CGO
```

Pros:
- Fast queries
- ACID transactions
- Single file database

Cons:
- Adds ~5MB to binary size
- Emails not directly readable on disk

---

### Option 2: Extract from Inbucket

If you prefer to stay closer to inbucket's implementation:

| Component | Source | Lines | Changes Needed |
|-----------|--------|-------|----------------|
| SMTP Server | `pkg/server/smtp/` | ~700 | Remove metrics, extensions, policy |
| File Storage | `pkg/storage/file/` | ~400 | Remove extension events |
| Message parsing | `pkg/message/` | ~300 | Remove policy |
| Hashing | `pkg/stringutil/` | ~50 | None |

**Total: ~1,450 lines to extract and adapt**

---

## Recommended Architecture for Supabase CLI

### Minimal Implementation (~300 lines total)

```go
package emailtesting

import (
    "github.com/emersion/go-smtp"
    "github.com/jhillyerd/enmime/v2"
)

// Config for email testing server
type Config struct {
    SMTPAddr    string // Default: "127.0.0.1:54325"
    StoragePath string // Default: from SUPABASE_MAIL_PATH or .supabase/mail
}

// Server is an embedded email testing server
type Server struct {
    config Config
    smtp   *smtp.Server
}

// Start the SMTP server
func (s *Server) Start(ctx context.Context) error

// Stop the SMTP server
func (s *Server) Stop() error

// ListMailboxes returns all mailboxes with emails
func (s *Server) ListMailboxes() ([]string, error)

// ListEmails returns email summaries for a mailbox
func (s *Server) ListEmails(mailbox string) ([]EmailSummary, error)

// GetEmail returns full email content
func (s *Server) GetEmail(mailbox, id string) (*Email, error)

// DeleteEmail removes an email
func (s *Server) DeleteEmail(mailbox, id string) error

// Types
type EmailSummary struct {
    ID      string
    From    string
    To      []string
    Subject string
    Date    time.Time
}

type Email struct {
    EmailSummary
    TextBody string
    HTMLBody string
    Raw      []byte
}
```

### File Structure

```
.supabase/mail/                          # Or custom path via env var
└── user@example.com/                    # Mailbox = recipient address
    ├── 1705312822000-a1b2.eml          # {unix_ms}-{random}.eml
    └── 1705312855000-c3d4.eml
```

### Dependencies

```go
require (
    github.com/emersion/go-smtp v0.21.0    // ~15KB
    github.com/jhillyerd/enmime/v2 v2.1.0  // ~50KB
)
```

**Total added to binary: ~65KB** (vs megabytes for Docker image)

---

## Implementation Steps

### Phase 1: SMTP Server (~2 hours)

```go
// internal/mail/smtp.go
package mail

import (
    "io"
    "github.com/emersion/go-smtp"
)

type smtpBackend struct {
    storePath string
}

func (b *smtpBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
    return &smtpSession{storePath: b.storePath}, nil
}

type smtpSession struct {
    storePath string
    from      string
    to        []string
}

func (s *smtpSession) AuthPlain(username, password string) error {
    return nil // Accept any auth for local testing
}

func (s *smtpSession) Mail(from string, opts *smtp.MailOptions) error {
    s.from = from
    return nil
}

func (s *smtpSession) Rcpt(to string, opts *smtp.RcptOptions) error {
    s.to = append(s.to, to)
    return nil
}

func (s *smtpSession) Data(r io.Reader) error {
    data, err := io.ReadAll(r)
    if err != nil {
        return err
    }

    // Save to each recipient's mailbox
    for _, recipient := range s.to {
        if err := saveEmail(s.storePath, recipient, data); err != nil {
            return err
        }
    }
    return nil
}

func (s *smtpSession) Reset() {
    s.from = ""
    s.to = nil
}

func (s *smtpSession) Logout() error {
    return nil
}
```

### Phase 2: File Storage (~1 hour)

```go
// internal/mail/storage.go
package mail

import (
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

func saveEmail(basePath, mailbox string, data []byte) error {
    // Sanitize mailbox name (email address)
    safeMailbox := sanitizeMailbox(mailbox)

    // Create mailbox directory
    mailboxPath := filepath.Join(basePath, safeMailbox)
    if err := os.MkdirAll(mailboxPath, 0755); err != nil {
        return err
    }

    // Generate unique filename
    id := generateID()
    filename := filepath.Join(mailboxPath, id+".eml")

    return os.WriteFile(filename, data, 0644)
}

func generateID() string {
    b := make([]byte, 4)
    rand.Read(b)
    return fmt.Sprintf("%d-%s", time.Now().UnixMilli(), hex.EncodeToString(b))
}

func sanitizeMailbox(email string) string {
    // Replace characters that are problematic in filenames
    // Keep it simple and readable
    return strings.ReplaceAll(email, "/", "_")
}
```

### Phase 3: Email Reading (~1 hour)

```go
// internal/mail/reader.go
package mail

import (
    "os"
    "path/filepath"
    "github.com/jhillyerd/enmime/v2"
)

func (s *Server) ListMailboxes() ([]string, error) {
    entries, err := os.ReadDir(s.config.StoragePath)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }

    var mailboxes []string
    for _, e := range entries {
        if e.IsDir() {
            mailboxes = append(mailboxes, e.Name())
        }
    }
    return mailboxes, nil
}

func (s *Server) GetEmail(mailbox, id string) (*Email, error) {
    path := filepath.Join(s.config.StoragePath, mailbox, id+".eml")
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    env, err := enmime.ReadEnvelope(f)
    if err != nil {
        return nil, err
    }

    return &Email{
        EmailSummary: EmailSummary{
            ID:      id,
            From:    env.GetHeader("From"),
            Subject: env.GetHeader("Subject"),
            // ... etc
        },
        TextBody: env.Text,
        HTMLBody: env.HTML,
    }, nil
}
```

### Phase 4: Server Lifecycle (~30 min)

```go
// internal/mail/server.go
package mail

import (
    "context"
    "time"
    "github.com/emersion/go-smtp"
)

func New(cfg Config) *Server {
    if cfg.SMTPAddr == "" {
        cfg.SMTPAddr = "127.0.0.1:54325"
    }
    if cfg.StoragePath == "" {
        cfg.StoragePath = os.Getenv("SUPABASE_MAIL_PATH")
        if cfg.StoragePath == "" {
            cfg.StoragePath = ".supabase/mail"
        }
    }
    return &Server{config: cfg}
}

func (s *Server) Start(ctx context.Context) error {
    be := &smtpBackend{storePath: s.config.StoragePath}

    s.smtp = smtp.NewServer(be)
    s.smtp.Addr = s.config.SMTPAddr
    s.smtp.Domain = "localhost"
    s.smtp.ReadTimeout = 10 * time.Second
    s.smtp.WriteTimeout = 10 * time.Second
    s.smtp.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
    s.smtp.MaxRecipients = 50
    s.smtp.AllowInsecureAuth = true

    go func() {
        <-ctx.Done()
        s.smtp.Close()
    }()

    return s.smtp.ListenAndServe()
}

func (s *Server) Stop() error {
    return s.smtp.Close()
}
```

---

## Comparison Summary

| Approach | Lines of Code | Dependencies | Binary Impact |
|----------|---------------|--------------|---------------|
| **Option 1: go-smtp + enmime** | ~300 | 2 packages | ~65KB |
| Option 2: Extract from inbucket | ~1,450 | 0 new | 0 |
| Current: Docker inbucket | 0 | Docker image | ~50MB image |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SUPABASE_MAIL_PATH` | `.supabase/mail` | Email storage directory |
| `SUPABASE_SMTP_ADDR` | `127.0.0.1:54325` | SMTP listen address |

---

## Recommendation

**Use Option 1** (go-smtp + enmime) because:

1. **Less code** - ~300 lines vs ~1,450 lines to extract
2. **Well-maintained** - Both libraries are actively maintained
3. **Cleaner** - Purpose-built libraries vs adapted code
4. **Simpler storage** - Plain .eml files vs gob-encoded index
5. **Debugging** - Users can open .eml files directly in email clients

The only reason to use Option 2 (extract from inbucket) would be if you need exact compatibility with inbucket's storage format, which doesn't seem necessary based on the requirements.
