package smtp

import (
	"bytes"
	"fmt"
	"io"

	"log"
	"net"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/test"
)

type scriptStep struct {
	send   string
	expect int
}

// Test commands in GREET state
func TestGreetState(t *testing.T) {
	ds := test.NewStore()
	server, logbuf, teardown := setupSMTPServer(ds)
	defer teardown()

	// Test out some mangled HELOs
	script := []scriptStep{
		{"HELO", 501},
		{"EHLO", 501},
		{"HELLO", 500},
		{"HELL", 500},
		{"hello", 500},
		{"Outlook", 500},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Valid HELOs
	if err := playSession(t, server, []scriptStep{{"HELO mydomain", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"HELO mydom.com", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"HelO mydom.com", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"helo 127.0.0.1", 250}}); err != nil {
		t.Error(err)
	}

	// Valid EHLOs
	if err := playSession(t, server, []scriptStep{{"EHLO mydomain", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"EHLO mydom.com", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"EhlO mydom.com", 250}}); err != nil {
		t.Error(err)
	}
	if err := playSession(t, server, []scriptStep{{"ehlo 127.0.0.1", 250}}); err != nil {
		t.Error(err)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test commands in READY state
func TestReadyState(t *testing.T) {
	ds := test.NewStore()
	server, logbuf, teardown := setupSMTPServer(ds)
	defer teardown()

	// Test out some mangled READY commands
	script := []scriptStep{
		{"HELO localhost", 250},
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
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	// Test out some valid MAIL commands
	script = []scriptStep{
		{"HELO localhost", 250},
		{"MAIL FROM:<john@gmail.com>", 250},
		{"RSET", 250},
		{"MAIL FROM: <john@gmail.com>", 250},
		{"RSET", 250},
		{"MAIL FROM: <john@gmail.com> BODY=8BITMIME", 250},
		{"RSET", 250},
		{"MAIL FROM:<john@gmail.com> SIZE=1024", 250},
		{"RSET", 250},
		{"MAIL FROM:<host!host!user/data@foo.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<\"first last\"@space.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<user\\@internal@external.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<user\\>name@host.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<\"user>name\"@host.com>", 250},
		{"RSET", 250},
		{"MAIL FROM:<\"user@internal\"@external.com>", 250},
	}
	if err := playSession(t, server, script); err != nil {
		t.Error(err)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test commands in MAIL state
func TestMailState(t *testing.T) {
	mds := test.NewStore()
	server, logbuf, teardown := setupSMTPServer(mds)
	defer teardown()

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

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test commands in DATA state
func TestDataState(t *testing.T) {
	mds := test.NewStore()
	server, logbuf, teardown := setupSMTPServer(mds)
	defer teardown()

	var script []scriptStep
	pipe := setupSMTPSession(server)
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

	// Test with no useful headers.
	pipe = setupSMTPSession(server)
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

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// playSession creates a new session, reads the greeting and then plays the script
func playSession(t *testing.T, server *Server, script []scriptStep) error {
	pipe := setupSMTPSession(server)
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

// net.Pipe does not implement deadlines
type mockConn struct {
	net.Conn
}

func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func setupSMTPServer(ds storage.Store) (s *Server, buf *bytes.Buffer, teardown func()) {
	cfg := &config.Root{
		MailboxNaming: config.FullNaming,
		SMTP: config.SMTP{
			Addr:            "127.0.0.1:2500",
			Domain:          "inbucket.local",
			MaxRecipients:   5,
			MaxMessageBytes: 5000,
			DefaultAccept:   true,
			RejectDomains:   []string{"deny.com"},
			Timeout:         5,
		},
	}
	// Capture log output.
	buf = new(bytes.Buffer)
	log.SetOutput(buf)
	// Create a server, don't start it.
	shutdownChan := make(chan bool)
	teardown = func() {
		close(shutdownChan)
	}
	addrPolicy := &policy.Addressing{Config: cfg}
	manager := &message.StoreManager{Store: ds}
	s = NewServer(cfg.SMTP, shutdownChan, manager, addrPolicy)
	return s, buf, teardown
}

var sessionNum int

func setupSMTPSession(server *Server) net.Conn {
	// Pair of pipes to communicate.
	serverConn, clientConn := net.Pipe()
	// Start the session.
	server.wg.Add(1)
	sessionNum++
	go server.startSession(sessionNum, &mockConn{serverConn})
	return clientConn
}
