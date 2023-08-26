package smtp

import (
	"fmt"
	"io"

	"net"
	"net/textproto"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/extension"
	"github.com/inbucket/inbucket/pkg/extension/event"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/test"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type scriptStep struct {
	send   string
	expect int
}

// Test valid commands in GREET state.
func TestGreetStateValidCommands(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	tests := []scriptStep{
		{"HELO mydomain", 250},
		{"HELO mydom.com", 250},
		{"HelO mydom.com", 250},
		{"helo 127.0.0.1", 250},
		{"HELO ABC", 250},
		{"EHLO mydomain", 250},
		{"EHLO mydom.com", 250},
		{"EhlO mydom.com", 250},
		{"ehlo 127.0.0.1", 250},
		{"EHLO a", 250},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			defer server.Drain() // Required to prevent test logging data race.
			script := []scriptStep{
				tc,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}
		})
	}
}

// Test invalid commands in GREET state.
func TestGreetState(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())
	defer server.Drain() // Required to prevent test logging data race.

	tests := []scriptStep{
		{"HELO", 501},
		{"EHLO", 501},
		{"HELLO", 500},
		{"HELL", 500},
		{"hello", 500},
		{"Outlook", 500},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			defer server.Drain() // Required to prevent test logging data race.
			script := []scriptStep{
				tc,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestEmptyEnvelope(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())
	defer server.Drain()

	// Test out some empty envelope without blanks
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<>", 501},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test out some empty envelope with blanks
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM: <>", 501},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}
}

// Test AUTH commands.
func TestAuth(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())
	defer server.Drain()

	// PLAIN AUTH
	script := []scriptStep{
		{"EHLO localhost", 250},
		{"AUTH PLAIN aW5idWNrZXQ6cGFzc3dvcmQK", 235},
		{"RSET", 250},
		{"AUTH GSSAPI aW5idWNrZXQ6cGFzc3dvcmQK", 500},
		{"RSET", 250},
		{"AUTH PLAIN", 500},
		{"RSET", 250},
		{"AUTH PLAIN aW5idWNrZXQ6cG Fzc3dvcmQK", 500},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// LOGIN AUTH
	script = []scriptStep{
		{"EHLO localhost", 250},
		{"AUTH LOGIN", 334}, // Test with user/pass present.
		{"username", 334},
		{"password", 235},
		{"RSET", 250},
		{"AUTH LOGIN", 334}, // Test with empty user/pass.
		{"", 334},
		{"", 235},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}
}

// Test TLS commands.
func TestTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())
	defer server.Drain()

	// Test Start TLS parsing.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"STARTTLS", 454}, // TLS unconfigured.
	}

	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}
}

// Test valid commands in READY state.
func TestReadyStateValidCommands(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	// Test out some valid MAIL commands
	tests := []scriptStep{
		{"MAIL FROM:<john@gmail.com>", 250},
		{"MAIL FROM: <john@gmail.com>", 250},
		{"MAIL FROM: <john@gmail.com> BODY=8BITMIME", 250},
		{"MAIL FROM:<john@gmail.com> SIZE=1024", 250},
		{"MAIL FROM:<john@gmail.com> SIZE=1024 BODY=8BITMIME", 250},
		{"MAIL FROM:<bounces@onmicrosoft.com> SIZE=4096 AUTH=<>", 250},
		{"MAIL FROM:<b@o.com> SIZE=4096 AUTH=<> BODY=7BIT", 250},
		{"MAIL FROM:<host!host!user/data@foo.com>", 250},
		{"MAIL FROM:<\"first last\"@space.com>", 250},
		{"MAIL FROM:<user\\@internal@external.com>", 250},
		{"MAIL FROM:<user\\>name@host.com>", 250},
		{"MAIL FROM:<\"user>name\"@host.com>", 250},
		{"MAIL FROM:<\"user@internal\"@external.com>", 250},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			defer server.Drain()
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}
		})
	}
}

// Test invalid domains in READY state.
func TestReadyStateRejectedDomains(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	tests := []scriptStep{
		{"MAIL FROM: <john@validdomain.com>", 250},
		{"MAIL FROM: <john@invalidomain.com>", 501},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			defer server.Drain()
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}
		})
	}

}

// Test invalid commands in READY state.
func TestReadyStateInvalidCommands(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	tests := []scriptStep{
		{"FOOB", 500},
		{"HELO", 503},
		{"DATA", 503},
		{"MAIL", 501},
		{"MAIL FROM john@gmail.com", 501},
		{"MAIL FROM:john@gmail.com", 501},
		{"MAIL FROM:<john@gmail.com> SIZE=147KB", 501},
		{"MAIL FROM: <john@gmail.com> SIZE147", 501},
		{"MAIL FROM:<first@last@gmail.com>", 501},
		{"MAIL FROM:<first last@gmail.com>", 501},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			defer server.Drain()
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}
		})
	}

}

// Test commands in MAIL state
func TestMailState(t *testing.T) {
	mds := test.NewStore()
	server := setupSMTPServer(mds, extension.NewHost())
	defer server.Drain()

	// Test out some mangled READY commands
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"FOOB", 500},
		{"HELO", 503},
		{"DATA", 503},
		{"MAIL", 503},
		{"RCPT", 501},
		{"RCPT TO", 501},
		{"RCPT TO james@gmail.com", 501},
		{"RCPT TO:<first last@host.com>", 501},
		{"RCPT TO:<fred@fish@host.com", 501},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test out some good RCPT commands
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"RCPT TO: <u2@gmail.com>", 250},
		{"RCPT TO:u3@gmail.com", 250},
		{"RCPT TO:u3@deny.com", 550},
		{"RCPT TO: u4@gmail.com", 250},
		{"RSET", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{`RCPT TO:<"first/last"@host.com`, 250},
		{"RCPT TO:<u1@[127.0.0.1]>", 250},
		{"RCPT TO:<u1@[IPv6:2001:db8:aaaa:1::100]>", 250},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test out recipient limit
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"RCPT TO:<u2@gmail.com>", 250},
		{"RCPT TO:<u3@gmail.com>", 250},
		{"RCPT TO:<u4@gmail.com>", 250},
		{"RCPT TO:<u5@gmail.com>", 250},
		{"RCPT TO:<u6@gmail.com>", 552},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test DATA
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"DATA", 354},
		{".", 250},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test late EHLO, similar to RSET
	script = []scriptStep{
		{"EHLO localhost", 250},
		{"EHLO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"EHLO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test RSET
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test QUIT
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"QUIT", 221},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}
}

// Test commands in DATA state
func TestDataState(t *testing.T) {
	mds := test.NewStore()
	server := setupSMTPServer(mds, extension.NewHost())
	defer server.Drain()

	var script []scriptStep
	pipe := setupSMTPSession(t, server)
	c := textproto.NewConn(pipe)

	if code, _, err := c.ReadCodeLine(220); err != nil {
		t.Errorf("Expected a 220 greeting, got %v", code)
	}
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"DATA", 354},
	}
	if err := playScriptAgainst(t, c, script); err != nil {
		t.Error(err)
	}

	// Send a message
	body := `To: u1@gmail.com
From: john@gmail.com
Subject: test

Hi!
`
	dw := c.DotWriter()
	_, _ = io.WriteString(dw, body)
	_ = dw.Close()
	if code, _, err := c.ReadCodeLine(250); err != nil {
		t.Errorf("Expected a 250 greeting, got %v", code)
	}
	_, _ = c.Cmd("QUIT")
	_, _, _ = c.ReadCodeLine(221)

	// Test with no useful headers.
	pipe = setupSMTPSession(t, server)
	c = textproto.NewConn(pipe)
	if code, _, err := c.ReadCodeLine(220); err != nil {
		t.Errorf("Expected a 220 greeting, got %v", code)
	}
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"DATA", 354},
	}
	if err := playScriptAgainst(t, c, script); err != nil {
		t.Error(err)
	}

	// Send a message
	body = `X-Useless-Header: true

	Hi! Can you still deliver this?
	`
	dw = c.DotWriter()
	_, _ = io.WriteString(dw, body)
	_ = dw.Close()
	if code, _, err := c.ReadCodeLine(250); err != nil {
		t.Errorf("Expected a 250 greeting, got %v", code)
	}
	_, _ = c.Cmd("QUIT")
	_, _, _ = c.ReadCodeLine(221)
}

// playSession creates a new session, reads the greeting and then plays the script
func playSession(t *testing.T, server *Server, script []scriptStep) error {
	pipe := setupSMTPSession(t, server)
	c := textproto.NewConn(pipe)

	if code, _, err := c.ReadCodeLine(220); err != nil {
		return fmt.Errorf("Expected a 220 greeting, got %v", code)
	}

	err := playScriptAgainst(t, c, script)

	// Not all tests leave the session in a clean state, so the following two
	// calls can fail
	_, _ = c.Cmd("QUIT")
	_, _, _ = c.ReadCodeLine(221)

	return err
}

// playScriptAgainst an existing connection, does not handle server greeting
func playScriptAgainst(t *testing.T, c *textproto.Conn, script []scriptStep) error {
	for i, step := range script {
		id, err := c.Cmd(step.send)
		if err != nil {
			return fmt.Errorf("Step %d, failed to send %q: %v", i, step.send, err)
		}

		c.StartResponse(id)
		code, msg, err := c.ReadResponse(step.expect)
		if err != nil {
			err = fmt.Errorf("Step %d, sent %q, expected %v, got %v: %q",
				i, step.send, step.expect, code, msg)
		}
		c.EndResponse(id)

		if err != nil {
			// Return after c.EndResponse so we don't hang the connection
			return err
		}
	}
	return nil
}

// Tests "MAIL FROM" emits BeforeMailAccepted event.
func TestBeforeMailAcceptedEventEmitted(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)
	defer server.Drain()

	var got *event.AddressParts
	extHost.Events.BeforeMailAccepted.AddListener(
		"test",
		func(addr event.AddressParts) *bool {
			got = &addr
			return nil
		})

	// Play and verify SMTP session.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"QUIT", 221}}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	assert.NotNil(t, got, "BeforeMailListener did not receive Address")
	assert.Equal(t, "john", got.Local, "Address local part had wrong value")
	assert.Equal(t, "gmail.com", got.Domain, "Address domain part had wrong value")
}

// Test "MAIL FROM" acts on BeforeMailAccepted event result.
func TestBeforeMailAcceptedEventResponse(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)
	defer server.Drain()

	var shouldReturn *bool
	var gotEvent *event.AddressParts
	extHost.Events.BeforeMailAccepted.AddListener(
		"test",
		func(addr event.AddressParts) *bool {
			gotEvent = &addr
			return shouldReturn
		})

	allowRes := true
	denyRes := false
	tcs := map[string]struct {
		script   scriptStep // Command to send and SMTP code expected.
		eventRes *bool      // Response to send from event listener.
	}{
		"allow": {
			script:   scriptStep{"MAIL FROM:<john@gmail.com>", 250},
			eventRes: &allowRes,
		},
		"deny": {
			script:   scriptStep{"MAIL FROM:<john@gmail.com>", 550},
			eventRes: &denyRes,
		},
		"defer": {
			script:   scriptStep{"MAIL FROM:<john@gmail.com>", 250},
			eventRes: nil,
		},
	}

	for name, tc := range tcs {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset event listener.
			shouldReturn = tc.eventRes
			gotEvent = nil

			// Play and verify SMTP session.
			script := []scriptStep{
				{"HELO localhost", 250},
				tc.script,
				{"QUIT", 221}}
			if err := playSession(t, server, script); err != nil {
				t.Error(err)
			}

			assert.NotNil(t, gotEvent, "BeforeMailListener did not receive Address")
		})
	}

}

// net.Pipe does not implement deadlines
type mockConn struct {
	net.Conn
}

func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func setupSMTPServer(ds storage.Store, extHost *extension.Host) *Server {
	cfg := &config.Root{
		MailboxNaming: config.FullNaming,
		SMTP: config.SMTP{
			Addr:            "127.0.0.1:2500",
			Domain:          "inbucket.local",
			MaxRecipients:   5,
			MaxMessageBytes: 5000,
			DefaultAccept:   true,
			RejectDomains:   []string{"deny.com"},
			RejectOriginDomains: []string{"invalidomain.com"},
			Timeout:         5,
		},
	}

	// Create a server, don't start it.
	addrPolicy := &policy.Addressing{Config: cfg}
	manager := &message.StoreManager{Store: ds}

	return NewServer(cfg.SMTP, manager, addrPolicy, extHost)
}

var sessionNum int

func setupSMTPSession(t *testing.T, server *Server) net.Conn {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	serverConn, clientConn := net.Pipe()

	// Start the session.
	server.wg.Add(1)
	sessionNum++
	go server.startSession(sessionNum, &mockConn{serverConn}, logger)

	return clientConn
}
