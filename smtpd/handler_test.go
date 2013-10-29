package smtpd

import (
	"github.com/jhillyerd/inbucket/config"
	"io/ioutil"
	"net"
	"net/textproto"
	"os"
	"testing"
	"time"
)

type scriptStep struct {
	send   string
	expect int
}

func TestHelo(t *testing.T) {
	server := setupSmtpServer()
	defer teardownSmtpServer(server)

	// Test out some manged HELOs
	playSession(t, server, []scriptStep{
		{"HELLO", 500},
	})
	playSession(t, server, []scriptStep{
		{"HELL", 500},
	})
	playSession(t, server, []scriptStep{
		{"hello", 500},
	})

	// Valid HELOs
	playSession(t, server, []scriptStep{
		{"HELO", 250},
	})
	playSession(t, server, []scriptStep{
		{"HELO mydomain", 250},
	})
	playSession(t, server, []scriptStep{
		{"HELO mydom.com", 250},
	})
	playSession(t, server, []scriptStep{
		{"HelO mydom.com", 250},
	})
}

// playSession creates a new session, reads the greeting and then plays the script
func playSession(t *testing.T, server *Server, script []scriptStep) {
	pipe := setupSmtpSession(server)
	c := textproto.NewConn(pipe)

	if code, _, err := c.ReadCodeLine(220); err != nil {
		t.Fatalf("Expected a 220 greeting, got %v", code)
		return
	}

	playScriptAgainst(t, c, script)

	c.Cmd("QUIT")
	c.ReadCodeLine(221)
}

// playScriptAgainst an existing connection, does not handle server greeting
func playScriptAgainst(t *testing.T, c *textproto.Conn, script []scriptStep) {
	for i, step := range script {
		id, err := c.Cmd(step.send)
		if err != nil {
			t.Fatalf("Step %d, failed to send %q: %v", i, step.send, err)
			return
		}

		c.StartResponse(id)
		if code, msg, err := c.ReadCodeLine(step.expect); err != nil {
			t.Errorf("Step %d, sent %q, expected %v, got %v: %q",
				i, step.send, step.expect, code, msg)
		}
		defer c.EndResponse(id)
	}
}

// net.Pipe does not implement deadlines
type mockConn struct {
	net.Conn
}

func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func setupSmtpServer() *Server {
	// Setup datastore
	path, err := ioutil.TempDir("", "inbucket")
	if err != nil {
		panic(err)
	}
	ds := NewFileDataStore(path)

	// Test Server Config
	cfg := config.SmtpConfig{
		Ip4address:      net.IPv4(127, 0, 0, 1),
		Ip4port:         2500,
		Domain:          "inbucket.local",
		DomainNoStore:   "bitbucket.local",
		MaxRecipients:   5,
		MaxIdleSeconds:  5,
		MaxMessageBytes: 5000,
		StoreMessages:   true,
	}

	// Create a server, don't start it
	return NewSmtpServer(cfg, ds)
}

var sessionNum int

func setupSmtpSession(server *Server) net.Conn {
	// Pair of pipes to communicate
	serverConn, clientConn := net.Pipe()
	// Start the session
	server.waitgroup.Add(1)
	sessionNum++
	go server.startSession(sessionNum, &mockConn{serverConn})

	return clientConn
}

func teardownSmtpServer(server *Server) {
	ds := server.dataStore.(*FileDataStore)
	if err := os.RemoveAll(ds.path); err != nil {
		panic(err)
	}
}
