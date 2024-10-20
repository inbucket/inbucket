package smtp

import (
	"fmt"
	"io"

	"net"
	"net/textproto"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			script := []scriptStep{
				tc,
				{"QUIT", 221}}
			playSession(t, server, script)
		})
	}
}

// Test invalid commands in GREET state.
func TestGreetState(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

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
			script := []scriptStep{
				tc,
				{"QUIT", 221}}
			playSession(t, server, script)
		})
	}
}

func TestEmptyEnvelope(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	// Test out some empty envelope without blanks
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<>", 501},
	}
	playSession(t, server, script)

	// Test out some empty envelope with blanks
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM: <>", 501},
	}
	playSession(t, server, script)
}

// Test AUTH commands.
func TestAuth(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

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
	playSession(t, server, script)

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
	playSession(t, server, script)
}

// Test TLS commands.
func TestTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupSMTPServer(ds, extension.NewHost())

	// Test Start TLS parsing.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"STARTTLS", 454}, // TLS unconfigured.
	}

	playSession(t, server, script)
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
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			playSession(t, server, script)
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
		{"MAIL FROM: <john@s1.otherinvaliddomain.com>", 501},
		{"MAIL FROM: <john@s2.otherinvaliddomain.com>", 501},
	}

	for _, tc := range tests {
		t.Run(tc.send, func(t *testing.T) {
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			playSession(t, server, script)
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
			script := []scriptStep{
				{"HELO localhost", 250},
				tc,
				{"QUIT", 221}}
			playSession(t, server, script)
		})
	}
}

// Test commands in MAIL state
func TestMailState(t *testing.T) {
	mds := test.NewStore()
	server := setupSMTPServer(mds, extension.NewHost())

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
	playSession(t, server, script)

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
	playSession(t, server, script)

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
	playSession(t, server, script)

	// Test DATA
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"DATA", 354},
		{".", 250},
	}
	playSession(t, server, script)

	// Test late EHLO, similar to RSET
	script = []scriptStep{
		{"EHLO localhost", 250},
		{"EHLO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"EHLO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
	}
	playSession(t, server, script)

	// Test RSET
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
	}
	playSession(t, server, script)

	// Test QUIT
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<u1@gmail.com>", 250},
		{"QUIT", 221},
	}
	playSession(t, server, script)
}

// Test commands in DATA state
func TestDataState(t *testing.T) {
	mds := test.NewStore()
	server := setupSMTPServer(mds, extension.NewHost())

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
	playScriptAgainst(t, c, script)

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
	playScriptAgainst(t, c, script)

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

// Tests "MAIL FROM" emits BeforeMailFromAccepted event.
func TestBeforeMailFromAcceptedEventEmitted(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var got *event.SMTPSession
	extHost.Events.BeforeMailFromAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			got = &session
			return &event.SMTPResponse{Action: event.ActionDefer}
		})

	// Play and verify SMTP session.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"QUIT", 221}}
	playSession(t, server, script)

	assert.NotNil(t, got, "BeforeMailListener did not receive Address")
	assert.Equal(t, "john@gmail.com", got.From.Address, "Address had wrong value")
	assert.Equal(t, "pipe", got.RemoteAddr, "RemoteAddr had wrong value")
}

// Test "MAIL FROM" acts on BeforeMailFromAccepted event result.
func TestBeforeMailFromAcceptedEventResponse(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var shouldReturn *event.SMTPResponse
	var gotEvent *event.SMTPSession

	extHost.Events.BeforeMailFromAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			gotEvent = &session
			return shouldReturn
		})

	tcs := map[string]struct {
		script   scriptStep         // Command to send and SMTP code expected.
		eventRes event.SMTPResponse // Response to send from event listener.
	}{
		"allow": {
			script:   scriptStep{"MAIL FROM:<john@gmail.com>", 250},
			eventRes: event.SMTPResponse{Action: event.ActionAllow},
		},
		"deny": {
			script: scriptStep{"MAIL FROM:<john@gmail.com>", 550},
			eventRes: event.SMTPResponse{
				Action:    event.ActionDeny,
				ErrorCode: 550,
				ErrorMsg:  "meh",
			},
		},
		"defer": {
			script:   scriptStep{"MAIL FROM:<john@gmail.com>", 250},
			eventRes: event.SMTPResponse{Action: event.ActionDefer},
		},
	}

	for name, tc := range tcs {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset event listener.
			shouldReturn = &tc.eventRes
			gotEvent = nil

			// Play and verify SMTP session.
			script := []scriptStep{
				{"HELO localhost", 250},
				tc.script, // error code is the significant part.
				{"QUIT", 221}}
			playSession(t, server, script)

			assert.NotNil(t, gotEvent, "BeforeMailFromAccepted did not receive event")
		})
	}
}

// Tests "RCPT TO" emits BeforeRcptToAccepted event.
func TestBeforeRcptToAcceptedSingleEventEmitted(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var got *event.SMTPSession
	extHost.Events.BeforeRcptToAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			got = &session
			return &event.SMTPResponse{Action: event.ActionDefer}
		})

	// Play and verify SMTP session.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<user@gmail.com>", 250},
		{"QUIT", 221}}
	playSession(t, server, script)

	require.NotNil(t, got, "BeforeRcptToListener did not receive SMTPSession")
	require.NotNil(t, got.From)
	require.NotNil(t, got.To)
	assert.Equal(t, "pipe", got.RemoteAddr, "RemoteAddr had wrong value")
	assert.Equal(t, "john@gmail.com", got.From.Address)
	assert.Len(t, got.To, 1)
	assert.Equal(t, "user@gmail.com", got.To[0].Address)
}

// Tests "RCPT TO" emits many BeforeRcptToAccepted events.
func TestBeforeRcptToAcceptedManyEventsEmitted(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var called int
	var got *event.SMTPSession
	extHost.Events.BeforeRcptToAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			called++
			got = &session
			return &event.SMTPResponse{Action: event.ActionDefer}
		})

	// Play and verify SMTP session.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<user@gmail.com>", 250},
		{"RCPT TO:<user2@gmail.com>", 250},
		{"QUIT", 221}}
	playSession(t, server, script)

	require.Equal(t, 2, called, "2 events should have been emitted")
	require.NotNil(t, got, "BeforeRcptToListener did not receive SMTPSession")
	require.NotNil(t, got.From)
	require.NotNil(t, got.To)
	assert.Equal(t, "john@gmail.com", got.From.Address)
	assert.Len(t, got.To, 2)
	assert.Equal(t, "user@gmail.com", got.To[0].Address)
	assert.Equal(t, "user2@gmail.com", got.To[1].Address)
}

// Tests we can continue after denying a "RCPT TO".
func TestBeforeRcptToAcceptedEventDeny(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var called int
	var got *event.SMTPSession
	extHost.Events.BeforeRcptToAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			called++

			// Reject bad address.
			action := event.ActionDefer
			for _, to := range session.To {
				if to.Address == "bad@apple.com" {
					action = event.ActionDeny
				}
			}

			got = &session
			return &event.SMTPResponse{Action: action, ErrorCode: 550, ErrorMsg: "rotten"}
		})

	// Play and verify SMTP session.
	script := []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RCPT TO:<user@gmail.com>", 250},
		{"RCPT TO:<bad@apple.com>", 550},
		{"RCPT TO:<user2@gmail.com>", 250},
		{"QUIT", 221}}
	playSession(t, server, script)

	require.Equal(t, 3, called, "3 events should have been emitted")
	require.NotNil(t, got, "BeforeRcptToListener did not receive SMTPSession")
	require.NotNil(t, got.From)
	require.NotNil(t, got.To)
	assert.Equal(t, "john@gmail.com", got.From.Address)

	// Verify bad apple dropped from final event.
	assert.Len(t, got.To, 2)
	assert.Equal(t, "user@gmail.com", got.To[0].Address)
	assert.Equal(t, "user2@gmail.com", got.To[1].Address)
}

// Test "RCPT TO" acts on BeforeRcptToAccepted event result.
func TestBeforeRcptToAcceptedEventResponse(t *testing.T) {
	ds := test.NewStore()
	extHost := extension.NewHost()
	server := setupSMTPServer(ds, extHost)

	var shouldReturn *event.SMTPResponse
	var gotEvent *event.SMTPSession
	extHost.Events.BeforeRcptToAccepted.AddListener(
		"test",
		func(session event.SMTPSession) *event.SMTPResponse {
			gotEvent = &session
			return shouldReturn
		})

	tcs := map[string]struct {
		script   scriptStep         // Command to send and SMTP code expected.
		eventRes event.SMTPResponse // Response to send from event listener.
	}{
		"allow": {
			script:   scriptStep{"RCPT TO:<john@gmail.com>", 250},
			eventRes: event.SMTPResponse{Action: event.ActionAllow},
		},
		"deny": {
			script: scriptStep{"RCPT TO:<john@gmail.com>", 550},
			eventRes: event.SMTPResponse{
				Action:    event.ActionDeny,
				ErrorCode: 550,
				ErrorMsg:  "meh",
			},
		},
		"defer": {
			script:   scriptStep{"RCPT TO:<john@gmail.com>", 250},
			eventRes: event.SMTPResponse{Action: event.ActionDefer},
		},
	}

	for name, tc := range tcs {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Reset event listener.
			shouldReturn = &tc.eventRes
			gotEvent = nil

			// Play and verify SMTP session.
			script := []scriptStep{
				{"HELO localhost", 250},
				{"MAIL FROM:<user@gmail.com>", 250},
				tc.script, // error code is the significant part.
				{"QUIT", 221}}
			playSession(t, server, script)

			assert.NotNil(t, gotEvent, "BeforeRcptToListener did not receive SMTPSession")
		})
	}
}

// playSession creates a new session, reads the greeting and then plays the script
func playSession(t *testing.T, server *Server, script []scriptStep) {
	t.Helper()
	pipe := setupSMTPSession(t, server)
	c := textproto.NewConn(pipe)

	if code, _, err := c.ReadCodeLine(220); err != nil {
		t.Errorf("expected a 220 greeting, got %v", code)
	}

	playScriptAgainst(t, c, script)

	// Not all tests leave the session in a clean state, so the following two calls can fail
	_, _ = c.Cmd("QUIT")
	_, _, _ = c.ReadCodeLine(221)
}

// playScriptAgainst an existing connection, does not handle server greeting
func playScriptAgainst(t *testing.T, c *textproto.Conn, script []scriptStep) {
	t.Helper()

	for i, step := range script {
		id, err := c.Cmd("%s", step.send)
		if err != nil {
			t.Fatalf("Step %d, failed to send %q: %v", i, step.send, err)
		}

		c.StartResponse(id)
		code, msg, err := c.ReadResponse(step.expect)
		if err != nil {
			err = fmt.Errorf("Step %d, sent %q, expected %v, got %v: %q",
				i, step.send, step.expect, code, msg)
		}
		c.EndResponse(id)

		if err != nil {
			// Fail after c.EndResponse so we don't hang the connection
			t.Fatal(err)
		}
	}
}

// net.Pipe does not implement deadlines
type mockConn struct {
	net.Conn
}

func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// Creates an unstarted smtp.Server.
func setupSMTPServer(ds storage.Store, extHost *extension.Host) *Server {
	cfg := &config.Root{
		MailboxNaming: config.FullNaming,
		SMTP: config.SMTP{
			Addr:                "127.0.0.1:2500",
			Domain:              "inbucket.local",
			MaxRecipients:       5,
			MaxMessageBytes:     5000,
			DefaultAccept:       true,
			RejectDomains:       []string{"deny.com"},
			RejectOriginDomains: []string{"invalidomain.com", "*.otherinvaliddomain.com"},
			Timeout:             5,
		},
	}

	// Create a server, but don't start it.
	addrPolicy := &policy.Addressing{Config: cfg}
	manager := &message.StoreManager{Store: ds, ExtHost: extHost}

	return NewServer(cfg.SMTP, manager, addrPolicy, extHost)
}

var sessionNum int

func setupSMTPSession(t *testing.T, server *Server) net.Conn {
	t.Helper()
	logger := zerolog.New(zerolog.NewTestWriter(t))
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()

		// Drain is required to prevent a test-logging data race. If a (failing) test run is
		// hanging, this may be the culprit.
		server.Drain()
	})

	// Start the session.
	sessionNum++
	go server.startSession(sessionNum, &mockConn{serverConn}, logger)

	return clientConn
}
