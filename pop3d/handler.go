package pop3d

import (
	"bufio"
	"bytes"
	//"container/list"
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"io"
	"net"
	//"strconv"
	"strings"
	"time"
)

type State int

const (
	AUTHORIZATION State = iota // The client must now identify and authenticate
	TRANSACTION                // Mailbox open, client may now issue commands
	UPDATE                     // Purge deleted messages, cleanup
	QUIT
)

func (s State) String() string {
	switch s {
	case AUTHORIZATION:
		return "AUTHORIZATION"
	case TRANSACTION:
		return "TRANSACTION"
	case UPDATE:
		return "UPDATE"
	case QUIT:
		return "QUIT"
	}
	return "Unknown"
}

var commands = map[string]bool{
	"QUIT": true,
	"STAT": true,
	"LIST": true,
	"RETR": true,
	"DELE": true,
	"NOOP": true,
	"RSET": true,
	"TOP":  true,
	"UIDL": true,
	"USER": true,
	"PASS": true,
	"APOP": true,
}

type Session struct {
	server     *Server
	id         int
	conn       net.Conn
	remoteHost string
	sendError  error
	state      State
	reader     *bufio.Reader
	user string
}

func NewSession(server *Server, id int, conn net.Conn) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return &Session{server: server, id: id, conn: conn, state: AUTHORIZATION,
		reader: reader, remoteHost: host}
}

func (ses *Session) String() string {
	return fmt.Sprintf("Session{id: %v, state: %v}", ses.id, ses.state)
}

/* Session flow:
 *  1. Send initial greeting
 *  2. Receive cmd
 *  3. If good cmd, respond, optionally change state
 *  4. If bad cmd, respond error
 *  5. Goto 2
 */
func (s *Server) startSession(id int, conn net.Conn) {
	log.Info("Connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	//expConnectsCurrent.Add(1)
	defer func() {
		conn.Close()
		s.waitgroup.Done()
		//expConnectsCurrent.Add(-1)
	}()

	ses := NewSession(s, id, conn)
	ses.greet()

	// This is our command reading loop
	for ses.state != QUIT && ses.sendError == nil {
		line, err := ses.readLine()
		if err == nil {
			if cmd, arg, ok := ses.parseCmd(line); ok {
				// Check against valid SMTP commands
				if cmd == "" {
					ses.send("-ERR Speak up")
					continue
				}
				if !commands[cmd] {
					ses.send(fmt.Sprintf("-ERR Syntax error, %v command unrecognized", cmd))
					ses.warn("Unrecognized command: %v", cmd)
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "APOP", "TOP":
					// These commands are not implemented in any state
					ses.send(fmt.Sprintf("-ERR %v command not implemented", cmd))
					ses.warn("Command %v not implemented by Inbucket", cmd)
					continue
				case "NOOP":
					// TODO move to transaction state
					ses.send("+OK I have sucessfully done nothing")
					continue
				case "RSET":
					// TODO move to transaction state
					// Reset session
					ses.trace("Resetting session state on RSET request")
					ses.reset()
					ses.send("+OK Session reset")
					continue
				case "QUIT":
					// TODO should be handled differently by transaciton
					ses.send("+OK Goodnight and good luck")
					ses.enterState(QUIT)
					continue
				}

				// Send command to handler for current state
				switch ses.state {
				case AUTHORIZATION:
					ses.authorizationHandler(cmd, arg)
					continue
				case TRANSACTION:
					//ses.transactionHandler(cmd, arg)
					continue
				case UPDATE:
					//ses.updateHandler(cmd, arg)
					continue
				}
				ses.error("Session entered unexpected state %v", ses.state)
				break
			} else {
				ses.send("-ERR Syntax error, command garbled")
			}
		} else {
			// readLine() returned an error
			if err == io.EOF {
				switch ses.state {
				case AUTHORIZATION:
					// EOF is common here
					ses.info("Client closed connection (state %v)", ses.state)
				default:
					ses.warn("Got EOF while in state %v", ses.state)
				}
				break
			}
			// not an EOF
			ses.warn("Connection error: %v", err)
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ses.send("-ERR Idle timeout, bye bye")
					break
				}
			}
			ses.send("-ERR Connection error, sorry")
			break
		}
	}
	if ses.sendError != nil {
		ses.warn("Network send error: %v", ses.sendError)
	}
	ses.info("Closing connection")
}

// AUTHORIZATION state
func (ses *Session) authorizationHandler(cmd string, arg []string) {
	switch cmd {
	case "HELO":
		ses.send("250 Great, let's get this show on the road")
		//ses.enterState(READY)
	case "EHLO":
		ses.send("250-Great, let's get this show on the road")
		ses.send("250-8BITMIME")
		//ses.enterState(READY)
	default:
		ses.ooSeq(cmd)
	}
}

func (ses *Session) enterState(state State) {
	ses.state = state
	ses.trace("Entering state %v", state)
}

func (ses *Session) greet() {
	ses.send("+OK Inbucket POP3 server ready")
}

// Calculate the next read or write deadline based on maxIdleSeconds
func (ses *Session) nextDeadline() time.Time {
	return time.Now().Add(time.Duration(ses.server.maxIdleSeconds) * time.Second)
}

// Send requested message, store errors in Session.sendError
func (ses *Session) send(msg string) {
	if err := ses.conn.SetWriteDeadline(ses.nextDeadline()); err != nil {
		ses.sendError = err
		return
	}
	if _, err := fmt.Fprint(ses.conn, msg+"\r\n"); err != nil {
		ses.sendError = err
		ses.warn("Failed to send: '%v'", msg)
		return
	}
	ses.trace(">> %v >>", msg)
}

// readByteLine reads a line of input into the provided buffer. Does
// not reset the Buffer - please do so prior to calling.
func (ses *Session) readByteLine(buf *bytes.Buffer) error {
	if err := ses.conn.SetReadDeadline(ses.nextDeadline()); err != nil {
		return err
	}
	for {
		line, err := ses.reader.ReadBytes('\r')
		if err != nil {
			return err
		}
		buf.Write(line)
		// Read the next byte looking for '\n'
		c, err := ses.reader.ReadByte()
		if err != nil {
			return err
		}
		buf.WriteByte(c)
		if c == '\n' {
			// We've reached the end of the line, return
			return nil
		}
		// Else, keep looking
	}
	// Should be unreachable
	return nil
}

// Reads a line of input
func (ses *Session) readLine() (line string, err error) {
	if err = ses.conn.SetReadDeadline(ses.nextDeadline()); err != nil {
		return "", err
	}
	line, err = ses.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	ses.trace("<< %v <<", strings.TrimRight(line, "\r\n"))
	return line, nil
}

func (ses *Session) parseCmd(line string) (cmd string, args []string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", nil, true
	}

	words := strings.Split(line, " ")
	return strings.ToUpper(words[0]), words[1:], true
}

func (ses *Session) reset() {
	//ses.enterState(READY)
}

func (ses *Session) ooSeq(cmd string) {
	ses.send(fmt.Sprintf("-ERR Command %v is out of sequence", cmd))
	ses.warn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (ses *Session) trace(msg string, args ...interface{}) {
	log.Trace("%v<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) info(msg string, args ...interface{}) {
	log.Info("%v<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) warn(msg string, args ...interface{}) {
	// Update metrics
	//expWarnsTotal.Add(1)
	log.Warn("%v<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) error(msg string, args ...interface{}) {
	// Update metrics
	//expErrorsTotal.Add(1)
	log.Error("%v<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}
