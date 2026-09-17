package pop3

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/textproto"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/storage/mem"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptStep describes one command/response exchange in a scripted POP3
// session. expect holds the exact response lines; multi-line responses include
// every line up to and including the terminating ".".
type scriptStep struct {
	send   string
	expect []string
}

// playPOP3Session starts a fresh session against server, consumes the greeting,
// and plays script against it.
func playPOP3Session(t *testing.T, server *Server, script []scriptStep) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	startPOP3Session(t, server, &mockConn{serverConn})
	t.Cleanup(func() { _ = clientConn.Close() })

	c := textproto.NewConn(clientConn)
	readPOP3Greeting(t, c)
	playPOP3Script(t, c, script)
}

// startPOP3Session runs startSession on serverConn, arranging cleanup so the
// server is drained before the test completes.
func startPOP3Session(t *testing.T, server *Server, serverConn net.Conn) {
	t.Helper()

	server.wg.Add(1)
	sessionNum++
	go server.startSession(context.Background(), sessionNum, serverConn)
	t.Cleanup(server.Drain)
}

func readPOP3Greeting(t *testing.T, c *textproto.Conn) {
	t.Helper()

	greeting, err := c.ReadLine()
	require.NoError(t, err, "reading server greeting")
	require.True(t, strings.HasPrefix(greeting, "+OK"), "expected +OK greeting, got %q", greeting)
}

// playPOP3Script sends each step and compares the exact response lines.
func playPOP3Script(t *testing.T, c *textproto.Conn, script []scriptStep) {
	t.Helper()

	for i, step := range script {
		require.NoErrorf(t, c.PrintfLine("%s", step.send), "step %d: sending %q", i, step.send)
		for j, want := range step.expect {
			got, err := c.ReadLine()
			require.NoErrorf(t, err, "step %d (%q): reading response line %d", i, step.send, j)
			require.Equalf(t, want, got, "step %d (%q): response line %d", i, step.send, j)
		}
	}
}

// loginSteps returns script steps authenticating as "mailbox"; count is the
// number of messages the greeting is expected to report.
func loginSteps(count int) []scriptStep {
	return []scriptStep{
		{"USER mailbox", []string{"+OK Hello mailbox, welcome to Inbucket"}},
		{"PASS anything", []string{fmt.Sprintf("+OK Found %v messages for mailbox", count)}},
	}
}

// newMemStore returns an empty memory store; it assigns deterministic
// sequential message IDs ("1", "2", ...).
func newMemStore(t *testing.T) storage.Store {
	t.Helper()

	store, err := mem.New(config.Storage{}, extension.NewHost())
	require.NoError(t, err)
	return store
}

// deliverRaw stores a message with the exact raw content provided, returning
// its size.
func deliverRaw(t *testing.T, store storage.Store, mailbox, raw string) int64 {
	t.Helper()

	delivery := &message.Delivery{
		Meta: event.MessageMetadata{
			Mailbox: mailbox,
			Subject: "raw",
		},
		Reader: io.NopCloser(strings.NewReader(raw)),
	}
	_, err := store.AddMessage(delivery)
	require.NoError(t, err)
	return int64(len(raw))
}

// fakeMsg is a storage.Message with canned metadata, used to exercise RETR/TOP
// source error paths.
type fakeMsg struct {
	mailbox string
	id      string
	size    int64
	source  func() (io.ReadCloser, error)
}

func (m *fakeMsg) Mailbox() string                { return m.mailbox }
func (m *fakeMsg) ID() string                     { return m.id }
func (m *fakeMsg) From() *mail.Address            { return nil }
func (m *fakeMsg) To() []*mail.Address            { return nil }
func (m *fakeMsg) Date() time.Time                { return time.Time{} }
func (m *fakeMsg) Subject() string                { return "fake" }
func (m *fakeMsg) Size() int64                    { return m.size }
func (m *fakeMsg) Seen() bool                     { return false }
func (m *fakeMsg) Source() (io.ReadCloser, error) { return m.source() }

// readDeadlineErrorConn fails SetReadDeadline, to exercise the read error
// paths of the session read loop.
type readDeadlineErrorConn struct {
	net.Conn
	err error
}

func (c *readDeadlineErrorConn) SetReadDeadline(time.Time) error { return c.err }

// fakeNetError is a net.Error with a configurable Timeout() result.
type fakeNetError struct {
	timeout bool
}

func (e fakeNetError) Error() string   { return "fake network error" }
func (e fakeNetError) Timeout() bool   { return e.timeout }
func (e fakeNetError) Temporary() bool { return false }

func TestAuthorizationCommandErrors(t *testing.T) {
	server := setupPOPServer(t, newMemStore(t), false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{
			name:   "USER without argument",
			script: []scriptStep{{"USER", []string{"-ERR Missing username argument"}}},
		},
		{
			name:   "PASS before USER",
			script: []scriptStep{{"PASS sekrit", []string{"-ERR Command PASS is out of sequence"}}},
		},
		{
			name:   "transaction command before login",
			script: []scriptStep{{"STAT", []string{"-ERR Command STAT is out of sequence"}}},
		},
		{
			name:   "APOP with one argument",
			script: []scriptStep{{"APOP mailbox", []string{"-ERR APOP requires two arguments"}}},
		},
		{
			name:   "unknown command",
			script: []scriptStep{{"FROBNICATE now", []string{"-ERR Syntax error, FROBNICATE command unrecognized"}}},
		},
		{
			name:   "empty command",
			script: []scriptStep{{"", []string{"-ERR Speak up"}}},
		},
		{
			name:   "QUIT",
			script: []scriptStep{{"QUIT", []string{"+OK Goodnight and good luck"}}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, tc.script)
		})
	}
}

func TestAuthorizationApopLogin(t *testing.T) {
	server := setupPOPServer(t, newMemStore(t), false, false)

	script := []scriptStep{
		{"APOP mailbox digest", []string{"+OK Found 0 messages for mailbox"}},
		{"STAT", []string{"+OK 0 0"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	}
	playPOP3Session(t, server, script)
}

func TestTransactionStat(t *testing.T) {
	ds := newMemStore(t)
	_, size1 := test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	_, size2 := test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"STAT", []string{fmt.Sprintf("+OK %v %v", 2, size1+size2)}},
		{"DELE 1", []string{"+OK Deleted message 1"}},
		{"STAT", []string{fmt.Sprintf("+OK %v %v", 1, size2)}},
		{"RSET", []string{"+OK Session reset"}},
		{"STAT", []string{fmt.Sprintf("+OK %v %v", 2, size1+size2)}},
		{"STAT 1", []string{"-ERR STAT command must have no arguments"}},
		{"NOOP", []string{"+OK I have successfully done nothing"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionList(t *testing.T) {
	ds := newMemStore(t)
	_, size1 := test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	_, size2 := test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"LIST", []string{"+OK Listing 2 messages",
			fmt.Sprintf("1 %v", size1), fmt.Sprintf("2 %v", size2), "."}},
		{"LIST 2", []string{fmt.Sprintf("+OK 2 %v", size2)}},
		{"DELE 1", []string{"+OK Deleted message 1"}},
		{"LIST", []string{"+OK Listing 1 messages", fmt.Sprintf("2 %v", size2), "."}},
		{"LIST 1", []string{"-ERR You deleted message 1"}},
		{"CAPA", []string{"+OK Capability list follows",
			"TOP", "USER", "UIDL", "IMPLEMENTATION Inbucket", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionListArgumentErrors(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{"two arguments", []scriptStep{{"LIST 1 2", []string{"-ERR LIST command must have zero or one argument"}}}},
		{"not an integer", []scriptStep{{"LIST x", []string{"-ERR LIST command requires an integer argument"}}}},
		{"zero", []scriptStep{{"LIST 0", []string{"-ERR LIST argument must be greater than 0"}}}},
		{"negative", []scriptStep{{"LIST -1", []string{"-ERR LIST argument must be greater than 0"}}}},
		{"out of range", []scriptStep{{"LIST 2", []string{"-ERR LIST argument must not exceed the number of messages"}}}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, slices.Concat(loginSteps(1), tc.script))
		})
	}
}

func TestTransactionUidl(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"UIDL", []string{"+OK Listing 2 messages", "1 1", "2 2", "."}},
		{"UIDL 2", []string{"+OK 2 2"}},
		{"DELE 2", []string{"+OK Deleted message 2"}},
		{"UIDL", []string{"+OK Listing 1 messages", "1 1", "."}},
		{"UIDL 2", []string{"-ERR You deleted message 2"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionUidlArgumentErrors(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{"two arguments", []scriptStep{{"UIDL 1 2", []string{"-ERR UIDL command must have zero or one argument"}}}},
		{"not an integer", []scriptStep{{"UIDL x", []string{"-ERR UIDL command requires an integer argument"}}}},
		{"zero", []scriptStep{{"UIDL 0", []string{"-ERR UIDL argument must be greater than 0"}}}},
		{"out of range", []scriptStep{{"UIDL 2", []string{"-ERR UIDL argument must not exceed the number of messages"}}}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, slices.Concat(loginSteps(1), tc.script))
		})
	}
}

func TestTransactionDeleAndRset(t *testing.T) {
	ds := newMemStore(t)
	_, size1 := test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"DELE 2", []string{"+OK Deleted message 2"}},
		{"DELE 2", []string{"-ERR Message 2 has already been deleted"}},
		{"STAT", []string{fmt.Sprintf("+OK 1 %v", size1)}},
		{"RSET", []string{"+OK Session reset"}},
		{"DELE 2", []string{"+OK Deleted message 2"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionDeleArgumentErrors(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{"no argument", []scriptStep{{"DELE", []string{"-ERR DELE command requires a single argument"}}}},
		{"two arguments", []scriptStep{{"DELE 1 2", []string{"-ERR DELE command requires a single argument"}}}},
		{"not an integer", []scriptStep{{"DELE x", []string{"-ERR DELE command requires an integer argument"}}}},
		{"zero", []scriptStep{{"DELE 0", []string{"-ERR DELE argument must be greater than 0"}}}},
		{"out of range", []scriptStep{{"DELE 2", []string{"-ERR DELE argument must not exceed the number of messages"}}}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, slices.Concat(loginSteps(1), tc.script))
		})
	}
}

func TestTransactionQuitProcessesDeletes(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	serverConn, clientConn := net.Pipe()
	startPOP3Session(t, server, &mockConn{serverConn})
	t.Cleanup(func() { _ = clientConn.Close() })

	c := textproto.NewConn(clientConn)
	readPOP3Greeting(t, c)
	playPOP3Script(t, c, slices.Concat(loginSteps(2), []scriptStep{
		{"DELE 1", []string{"+OK Deleted message 1"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	}))

	// The deletes are processed before the session goroutine completes.
	server.Drain()
	msgs := test.GetAndCountMessages(t, ds, "mailbox", 1)
	assert.Equal(t, "2", msgs[0].ID())
}

func TestTransactionQuitPreservesMessagesAfterRset(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	serverConn, clientConn := net.Pipe()
	startPOP3Session(t, server, &mockConn{serverConn})
	t.Cleanup(func() { _ = clientConn.Close() })

	c := textproto.NewConn(clientConn)
	readPOP3Greeting(t, c)
	playPOP3Script(t, c, slices.Concat(loginSteps(2), []scriptStep{
		{"DELE 1", []string{"+OK Deleted message 1"}},
		{"RSET", []string{"+OK Session reset"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	}))

	server.Drain()
	test.GetAndCountMessages(t, ds, "mailbox", 2)
}

func TestTransactionQuitWithoutDeletes(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	serverConn, clientConn := net.Pipe()
	startPOP3Session(t, server, &mockConn{serverConn})
	t.Cleanup(func() { _ = clientConn.Close() })

	c := textproto.NewConn(clientConn)
	readPOP3Greeting(t, c)
	playPOP3Script(t, c, slices.Concat(loginSteps(1), []scriptStep{
		{"QUIT", []string{"+OK We will process your deletes"}},
	}))

	server.Drain()
	test.GetAndCountMessages(t, ds, "mailbox", 1)
}

func TestTransactionRetr(t *testing.T) {
	ds := newMemStore(t)
	_, size1 := test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"RETR 1", []string{fmt.Sprintf("+OK %v bytes follows", size1),
			"To: somebody@host", "From: somebodyelse@host", "Subject: One", "", "Test Body", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionRetrArgumentErrors(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{"no argument", []scriptStep{{"RETR", []string{"-ERR RETR command requires a single argument"}}}},
		{"two arguments", []scriptStep{{"RETR 1 2", []string{"-ERR RETR command requires a single argument"}}}},
		{"not an integer", []scriptStep{{"RETR x", []string{"-ERR RETR command requires an integer argument"}}}},
		{"zero", []scriptStep{{"RETR 0", []string{"-ERR RETR argument must be greater than 0"}}}},
		{"out of range", []scriptStep{{"RETR 2", []string{"-ERR RETR argument must not exceed the number of messages"}}}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, slices.Concat(loginSteps(1), tc.script))
		})
	}
}

// Documents current behavior: RFC 1939 requires a negative status indicator
// when RETR refers to a message marked as deleted, but the handler does not
// check the retain flag. Flagged for a follow-up fix.
func TestTransactionRetrDeletedMessage(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	_, size2 := test.DeliverToStore(t, ds, "mailbox", "Two", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(2), []scriptStep{
		{"DELE 2", []string{"+OK Deleted message 2"}},
		{"RETR 2", []string{fmt.Sprintf("+OK %v bytes follows", size2),
			"To: somebody@host", "From: somebodyelse@host", "Subject: Two", "", "Test Body", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionTop(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	headers := []string{"To: somebody@host", "From: somebodyelse@host", "Subject: One"}

	script := slices.Concat(loginSteps(1), []scriptStep{
		// Zero body lines: headers plus the blank separator line only.
		{"TOP 1 0", slices.Concat([]string{"+OK Top of message follows"}, headers, []string{""}, []string{"."})},
		// One body line.
		{"TOP 1 1", slices.Concat([]string{"+OK Top of message follows"}, headers, []string{"", "Test Body", "."})},
		// More body lines than the message contains.
		{"TOP 1 99", slices.Concat([]string{"+OK Top of message follows"}, headers, []string{"", "Test Body", "."})},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionTopArgumentErrors(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	tcs := []struct {
		name   string
		script []scriptStep
	}{
		{"one argument", []scriptStep{{"TOP 1", []string{"-ERR TOP command requires two arguments"}}}},
		{"three arguments", []scriptStep{{"TOP 1 2 3", []string{"-ERR TOP command requires two arguments"}}}},
		{"message not an integer", []scriptStep{{"TOP x 1", []string{"-ERR TOP command requires an integer argument"}}}},
		{"message zero", []scriptStep{{"TOP 0 1", []string{"-ERR TOP first argument must be greater than 0"}}}},
		{"message out of range", []scriptStep{{"TOP 2 1", []string{"-ERR TOP first argument must not exceed the number of messages"}}}},
		{"lines not an integer", []scriptStep{{"TOP 1 x", []string{"-ERR TOP command requires an integer argument"}}}},
		{"lines negative", []scriptStep{{"TOP 1 -1", []string{"-ERR TOP second argument must be non-negative"}}}},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			playPOP3Session(t, server, slices.Concat(loginSteps(1), tc.script))
		})
	}
}

// Documents current behavior: RFC 1939 (TOP, section 7) requires a negative
// status indicator when TOP refers to a message marked as deleted, but the
// handler does not check the retain flag. Flagged for a follow-up fix.
func TestTransactionTopDeletedMessage(t *testing.T) {
	ds := newMemStore(t)
	test.DeliverToStore(t, ds, "mailbox", "One", time.Now())
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(1), []scriptStep{
		{"DELE 1", []string{"+OK Deleted message 1"}},
		{"TOP 1 0", []string{"+OK Top of message follows",
			"To: somebody@host", "From: somebodyelse@host", "Subject: One", "", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionRetrDotStuffing(t *testing.T) {
	ds := newMemStore(t)
	size := deliverRaw(t, ds, "mailbox", "Subject: dots\r\n\r\n.leading\r\n..double\r\n.\r\nplain\r\n")
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(1), []scriptStep{
		// Each leading period is doubled; a lone "." line becomes "..".
		{"RETR 1", []string{fmt.Sprintf("+OK %v bytes follows", size),
			"Subject: dots", "", "..leading", "...double", "..", "plain", "."}},
		{"TOP 1 0", []string{"+OK Top of message follows", "Subject: dots", "", "."}},
		{"TOP 1 2", []string{"+OK Top of message follows",
			"Subject: dots", "", "..leading", "...double", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionRetrSourceError(t *testing.T) {
	ds := test.NewStore()
	_, err := ds.AddMessage(&fakeMsg{
		mailbox: "mailbox", id: "m1", size: 100,
		source: func() (io.ReadCloser, error) { return nil, errors.New("source unavailable") },
	})
	require.NoError(t, err)
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(1), []scriptStep{
		{"RETR 1", []string{"+OK 100 bytes follows",
			"-ERR Failed to RETR that message, internal error"}},
		{"TOP 1 0", []string{"+OK Top of message follows",
			"-ERR Failed to RETR that message, internal error"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

func TestTransactionRetrReadError(t *testing.T) {
	ds := test.NewStore()
	_, err := ds.AddMessage(&fakeMsg{
		mailbox: "mailbox", id: "m1", size: 100,
		source: func() (io.ReadCloser, error) {
			return io.NopCloser(iotest.ErrReader(errors.New("short read"))), nil
		},
	})
	require.NoError(t, err)
	server := setupPOPServer(t, ds, false, false)

	script := slices.Concat(loginSteps(1), []scriptStep{
		// A read failure mid-message terminates the multi-line response, then
		// reports the error.
		{"RETR 1", []string{"+OK 100 bytes follows", ".",
			"-ERR Failed to RETR that message, internal error"}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}

// Documents current behavior: errors loading the mailbox are logged but not
// reported to the client, which sees an empty mailbox.
func TestTransactionLoadMailboxError(t *testing.T) {
	server := setupPOPServer(t, test.NewStore(), false, false)

	script := []scriptStep{
		{"USER messageserr", []string{"+OK Hello messageserr, welcome to Inbucket"}},
		{"PASS anything", []string{"+OK Found 0 messages for messageserr"}},
		{"STAT", []string{"+OK 0 0"}},
		{"LIST", []string{"+OK Listing 0 messages", "."}},
		{"UIDL", []string{"+OK Listing 0 messages", "."}},
		{"QUIT", []string{"+OK We will process your deletes"}},
	}
	playPOP3Session(t, server, script)
}

func TestSessionString(t *testing.T) {
	server := setupPOPServer(t, test.NewStore(), false, false)

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	ssn := NewSession(server, 7, &mockConn{serverConn}, zerolog.New(zerolog.NewTestWriter(t)))
	assert.Equal(t, "Session{id: 7, state: AUTHORIZATION}", ssn.String())
}

func TestReadDeadlineError(t *testing.T) {
	tcs := []struct {
		name string
		err  error
		want string
	}{
		{"timeout", fakeNetError{timeout: true}, "-ERR Idle timeout, bye bye"},
		{"other error", fakeNetError{timeout: false}, "-ERR Connection error, sorry"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			server := setupPOPServer(t, newMemStore(t), false, false)

			serverConn, clientConn := net.Pipe()
			startPOP3Session(t, server, &readDeadlineErrorConn{serverConn, tc.err})
			t.Cleanup(func() { _ = clientConn.Close() })

			c := textproto.NewConn(clientConn)
			readPOP3Greeting(t, c)

			line, err := c.ReadLine()
			require.NoError(t, err, "reading error response")
			assert.Equal(t, tc.want, line)

			// The session should be closed.
			_, err = c.ReadLine()
			assert.Error(t, err, "expected connection to be closed")
		})
	}
}

func TestServerStartServeAndQuit(t *testing.T) {
	server, err := NewServer(config.POP3{
		Addr:    "127.0.0.1:0",
		Domain:  "inbucket.local",
		Timeout: 5 * time.Second,
	}, test.NewStore())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	started := make(chan struct{})
	go func() {
		defer close(started)
		server.Start(ctx, func() { close(ready) })
	}()
	<-ready
	t.Cleanup(func() {
		cancel()
		<-started
		server.Drain()
	})

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", server.listener.Addr().String())
	require.NoError(t, err)
	c := textproto.NewConn(conn)
	readPOP3Greeting(t, c)
	require.NoError(t, c.PrintfLine("QUIT"))

	line, err := c.ReadLine()
	require.NoError(t, err)
	assert.Equal(t, "+OK Goodnight and good luck", line)
	require.NoError(t, conn.Close())
}

func TestServerStartAddressErrors(t *testing.T) {
	// Reserve a port to force a bind failure.
	reserved, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reserved.Close() })

	tcs := []struct {
		name string
		addr string
	}{
		{"invalid port", "127.0.0.1:notaport"},
		{"port in use", reserved.Addr().String()},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(config.POP3{
				Addr:    tc.addr,
				Domain:  "inbucket.local",
				Timeout: 5 * time.Second,
			}, test.NewStore())
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			go server.Start(ctx, func() { t.Error("readyFunc must not be called") })

			select {
			case err := <-server.Notify():
				require.Error(t, err)
			case <-time.After(10 * time.Second):
				t.Fatal("expected a fatal error notification")
			}
		})
	}
}

func TestServerAcceptErrorNotifies(t *testing.T) {
	server, err := NewServer(config.POP3{
		Addr:    "127.0.0.1:0",
		Domain:  "inbucket.local",
		Timeout: 5 * time.Second,
	}, test.NewStore())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	started := make(chan struct{})
	go func() {
		defer close(started)
		server.Start(ctx, func() { close(ready) })
	}()
	<-ready
	t.Cleanup(func() {
		cancel()
		<-started
		server.Drain()
	})

	// Closing the listener without cancelling the context makes serve()
	// report the accept error via Notify.
	require.NoError(t, server.listener.Close())

	select {
	case err := <-server.Notify():
		require.Error(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("expected a fatal error notification")
	}
}

func TestNewServerCertificateError(t *testing.T) {
	_, err := NewServer(config.POP3{
		Addr:       "127.0.0.1:1100",
		Domain:     "inbucket.local",
		Timeout:    5 * time.Second,
		TLSEnabled: true,
		TLSCert:    "/nonexistent/cert.pem",
		TLSPrivKey: "/nonexistent/key.pem",
	}, test.NewStore())
	require.Error(t, err)
}

// TestValidateMsgNum exercises the shared message-number validation helper
// directly.
func TestValidateMsgNum(t *testing.T) {
	server := setupPOPServer(t, newMemStore(t), false, false)

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close() })

	// Drain server->client output so sends do not block.
	output := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(clientConn)
		for sc.Scan() {
			output <- sc.Text()
		}
		close(output)
	}()

	ssn := NewSession(server, 1, &mockConn{serverConn}, zerolog.New(zerolog.NewTestWriter(t)))
	ssn.messages = make([]storage.Message, 3)
	ssn.retainAll()

	tcs := []struct {
		name     string
		arg      string
		wantNum  int
		wantOK   bool
		wantSend string
	}{
		{name: "first message", arg: "1", wantNum: 1, wantOK: true},
		{name: "last message", arg: "3", wantNum: 3, wantOK: true},
		{name: "not an integer", arg: "x", wantSend: "-ERR TEST command requires an integer argument"},
		{name: "integer overflow", arg: "9999999999999", wantSend: "-ERR TEST command requires an integer argument"},
		{name: "zero", arg: "0", wantSend: "-ERR TEST argument must be greater than 0"},
		{name: "negative", arg: "-1", wantSend: "-ERR TEST argument must be greater than 0"},
		{name: "exceeds message count", arg: "4", wantSend: "-ERR TEST argument must not exceed the number of messages"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			num, ok := ssn.validateMsgNum("TEST", "TEST", tc.arg)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantNum, num)

			if tc.wantSend != "" {
				select {
				case line := <-output:
					assert.Equal(t, tc.wantSend, line)
				case <-time.After(time.Second):
					t.Error("expected an error response")
				}
			}
		})
	}
}

// TestTopUnlimitedLinesMatchesRetr guards the shared implementation of RETR
// and TOP: a TOP line limit larger than the message must produce the same
// output as RETR.
func TestTopUnlimitedLinesMatchesRetr(t *testing.T) {
	ds := newMemStore(t)
	size := deliverRaw(t, ds, "mailbox", "Subject: equivalence\r\n\r\nline 1\r\nline 2\r\n.leading\r\n")
	server := setupPOPServer(t, ds, false, false)

	body := []string{"Subject: equivalence", "", "line 1", "line 2", "..leading", "."}
	retrExpect := slices.Concat([]string{fmt.Sprintf("+OK %v bytes follows", size)}, body)
	topExpect := slices.Concat([]string{"+OK Top of message follows"}, body)

	script := slices.Concat(loginSteps(1), []scriptStep{
		{"RETR 1", retrExpect},
		{"TOP 1 999999", topExpect},
		{"QUIT", []string{"+OK We will process your deletes"}},
	})
	playPOP3Session(t, server, script)
}
