package pop3

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jhillyerd/inbucket/pkg/log"
	"github.com/jhillyerd/inbucket/pkg/storage"
)

// State tracks the current mode of our POP3 state machine
type State int

const (
	// AUTHORIZATION state: the client must now identify and authenticate
	AUTHORIZATION State = iota
	// TRANSACTION state: mailbox open, client may now issue commands
	TRANSACTION
	// QUIT state: client requests us to end session
	QUIT
)

func (s State) String() string {
	switch s {
	case AUTHORIZATION:
		return "AUTHORIZATION"
	case TRANSACTION:
		return "TRANSACTION"
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
	"CAPA": true,
}

// Session defines an active POP3 session
type Session struct {
	server     *Server           // Reference to the server we belong to
	id         int               // Session ID number
	conn       net.Conn          // Our network connection
	remoteHost string            // IP address of client
	sendError  error             // Used to bail out of read loop on send error
	state      State             // Current session state
	reader     *bufio.Reader     // Buffered reader for our net conn
	user       string            // Mailbox name
	messages   []storage.Message // Slice of messages in mailbox
	retain     []bool            // Messages to retain upon UPDATE (true=retain)
	msgCount   int               // Number of undeleted messages
}

// NewSession creates a new POP3 session
func NewSession(server *Server, id int, conn net.Conn) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return &Session{server: server, id: id, conn: conn, state: AUTHORIZATION,
		reader: reader, remoteHost: host}
}

func (s *Session) String() string {
	return fmt.Sprintf("Session{id: %v, state: %v}", s.id, s.state)
}

/* Session flow:
 *  1. Send initial greeting
 *  2. Receive cmd
 *  3. If good cmd, respond, optionally change state
 *  4. If bad cmd, respond error
 *  5. Goto 2
 */
func (s *Server) startSession(id int, conn net.Conn) {
	log.Infof("POP3 connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	//expConnectsCurrent.Add(1)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Error closing POP3 connection for <%v>: %v", id, err)
		}
		s.waitgroup.Done()
		//expConnectsCurrent.Add(-1)
	}()

	ssn := NewSession(s, id, conn)
	ssn.send(fmt.Sprintf("+OK Inbucket POP3 server ready <%v.%v@%v>", os.Getpid(),
		time.Now().Unix(), s.domain))

	// This is our command reading loop
	for ssn.state != QUIT && ssn.sendError == nil {
		line, err := ssn.readLine()
		if err == nil {
			if cmd, arg, ok := ssn.parseCmd(line); ok {
				// Check against valid SMTP commands
				if cmd == "" {
					ssn.send("-ERR Speak up")
					continue
				}
				if !commands[cmd] {
					ssn.send(fmt.Sprintf("-ERR Syntax error, %v command unrecognized", cmd))
					ssn.logWarn("Unrecognized command: %v", cmd)
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "CAPA":
					// List our capabilities per RFC2449
					ssn.send("+OK Capability list follows")
					ssn.send("TOP")
					ssn.send("USER")
					ssn.send("UIDL")
					ssn.send("IMPLEMENTATION Inbucket")
					ssn.send(".")
					continue
				}

				// Send command to handler for current state
				switch ssn.state {
				case AUTHORIZATION:
					ssn.authorizationHandler(cmd, arg)
					continue
				case TRANSACTION:
					ssn.transactionHandler(cmd, arg)
					continue
				}
				ssn.logError("Session entered unexpected state %v", ssn.state)
				break
			} else {
				ssn.send("-ERR Syntax error, command garbled")
			}
		} else {
			// readLine() returned an error
			if err == io.EOF {
				switch ssn.state {
				case AUTHORIZATION:
					// EOF is common here
					ssn.logInfo("Client closed connection (state %v)", ssn.state)
				default:
					ssn.logWarn("Got EOF while in state %v", ssn.state)
				}
				break
			}
			// not an EOF
			ssn.logWarn("Connection error: %v", err)
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ssn.send("-ERR Idle timeout, bye bye")
					break
				}
			}
			ssn.send("-ERR Connection error, sorry")
			break
		}
	}
	if ssn.sendError != nil {
		ssn.logWarn("Network send error: %v", ssn.sendError)
	}
	ssn.logInfo("Closing connection")
}

// AUTHORIZATION state
func (s *Session) authorizationHandler(cmd string, args []string) {
	switch cmd {
	case "QUIT":
		s.send("+OK Goodnight and good luck")
		s.enterState(QUIT)
	case "USER":
		if len(args) > 0 {
			s.user = args[0]
			s.send(fmt.Sprintf("+OK Hello %v, welcome to Inbucket", s.user))
		} else {
			s.send("-ERR Missing username argument")
		}
	case "PASS":
		if s.user == "" {
			s.ooSeq(cmd)
		} else {
			s.loadMailbox()
			s.send(fmt.Sprintf("+OK Found %v messages for %v", s.msgCount, s.user))
			s.enterState(TRANSACTION)
		}
	case "APOP":
		if len(args) != 2 {
			s.logWarn("Expected two arguments for APOP")
			s.send("-ERR APOP requires two arguments")
			return
		}
		s.user = args[0]
		s.loadMailbox()
		s.send(fmt.Sprintf("+OK Found %v messages for %v", s.msgCount, s.user))
		s.enterState(TRANSACTION)
	default:
		s.ooSeq(cmd)
	}
}

// TRANSACTION state
func (s *Session) transactionHandler(cmd string, args []string) {
	switch cmd {
	case "STAT":
		if len(args) != 0 {
			s.logWarn("STAT got an unexpected argument")
			s.send("-ERR STAT command must have no arguments")
			return
		}
		var count int
		var size int64
		for i, msg := range s.messages {
			if s.retain[i] {
				count++
				size += msg.Size()
			}
		}
		s.send(fmt.Sprintf("+OK %v %v", count, size))
	case "LIST":
		if len(args) > 1 {
			s.logWarn("LIST command had more than 1 argument")
			s.send("-ERR LIST command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				s.logWarn("LIST command argument was not an integer")
				s.send("-ERR LIST command requires an integer argument")
				return
			}
			if msgNum < 1 {
				s.logWarn("LIST command argument was less than 1")
				s.send("-ERR LIST argument must be greater than 0")
				return
			}
			if int(msgNum) > len(s.messages) {
				s.logWarn("LIST command argument was greater than number of messages")
				s.send("-ERR LIST argument must not exceed the number of messages")
				return
			}
			if !s.retain[msgNum-1] {
				s.logWarn("Client tried to LIST a message it had deleted")
				s.send(fmt.Sprintf("-ERR You deleted message %v", msgNum))
				return
			}
			s.send(fmt.Sprintf("+OK %v %v", msgNum, s.messages[msgNum-1].Size()))
		} else {
			s.send(fmt.Sprintf("+OK Listing %v messages", s.msgCount))
			for i, msg := range s.messages {
				if s.retain[i] {
					s.send(fmt.Sprintf("%v %v", i+1, msg.Size()))
				}
			}
			s.send(".")
		}
	case "UIDL":
		if len(args) > 1 {
			s.logWarn("UIDL command had more than 1 argument")
			s.send("-ERR UIDL command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				s.logWarn("UIDL command argument was not an integer")
				s.send("-ERR UIDL command requires an integer argument")
				return
			}
			if msgNum < 1 {
				s.logWarn("UIDL command argument was less than 1")
				s.send("-ERR UIDL argument must be greater than 0")
				return
			}
			if int(msgNum) > len(s.messages) {
				s.logWarn("UIDL command argument was greater than number of messages")
				s.send("-ERR UIDL argument must not exceed the number of messages")
				return
			}
			if !s.retain[msgNum-1] {
				s.logWarn("Client tried to UIDL a message it had deleted")
				s.send(fmt.Sprintf("-ERR You deleted message %v", msgNum))
				return
			}
			s.send(fmt.Sprintf("+OK %v %v", msgNum, s.messages[msgNum-1].ID()))
		} else {
			s.send(fmt.Sprintf("+OK Listing %v messages", s.msgCount))
			for i, msg := range s.messages {
				if s.retain[i] {
					s.send(fmt.Sprintf("%v %v", i+1, msg.ID()))
				}
			}
			s.send(".")
		}
	case "DELE":
		if len(args) != 1 {
			s.logWarn("DELE command had invalid number of arguments")
			s.send("-ERR DELE command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logWarn("DELE command argument was not an integer")
			s.send("-ERR DELE command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logWarn("DELE command argument was less than 1")
			s.send("-ERR DELE argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logWarn("DELE command argument was greater than number of messages")
			s.send("-ERR DELE argument must not exceed the number of messages")
			return
		}
		if s.retain[msgNum-1] {
			s.retain[msgNum-1] = false
			s.msgCount--
			s.send(fmt.Sprintf("+OK Deleted message %v", msgNum))
		} else {
			s.logWarn("Client tried to DELE an already deleted message")
			s.send(fmt.Sprintf("-ERR Message %v has already been deleted", msgNum))
		}
	case "RETR":
		if len(args) != 1 {
			s.logWarn("RETR command had invalid number of arguments")
			s.send("-ERR RETR command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logWarn("RETR command argument was not an integer")
			s.send("-ERR RETR command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logWarn("RETR command argument was less than 1")
			s.send("-ERR RETR argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logWarn("RETR command argument was greater than number of messages")
			s.send("-ERR RETR argument must not exceed the number of messages")
			return
		}
		s.send(fmt.Sprintf("+OK %v bytes follows", s.messages[msgNum-1].Size()))
		s.sendMessage(s.messages[msgNum-1])
	case "TOP":
		if len(args) != 2 {
			s.logWarn("TOP command had invalid number of arguments")
			s.send("-ERR TOP command requires two arguments")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logWarn("TOP command first argument was not an integer")
			s.send("-ERR TOP command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logWarn("TOP command first argument was less than 1")
			s.send("-ERR TOP first argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logWarn("TOP command first argument was greater than number of messages")
			s.send("-ERR TOP first argument must not exceed the number of messages")
			return
		}

		var lines int64
		lines, err = strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			s.logWarn("TOP command second argument was not an integer")
			s.send("-ERR TOP command requires an integer argument")
			return
		}
		if lines < 0 {
			s.logWarn("TOP command second argument was negative")
			s.send("-ERR TOP second argument must be non-negative")
			return
		}
		s.send("+OK Top of message follows")
		s.sendMessageTop(s.messages[msgNum-1], int(lines))
	case "QUIT":
		s.send("+OK We will process your deletes")
		s.processDeletes()
		s.enterState(QUIT)
	case "NOOP":
		s.send("+OK I have sucessfully done nothing")
	case "RSET":
		// Reset session, don't actually delete anything I told you to
		s.logTrace("Resetting session state on RSET request")
		s.reset()
		s.send("+OK Session reset")
	default:
		s.ooSeq(cmd)
	}
}

// Send the contents of the message to the client
func (s *Session) sendMessage(msg storage.Message) {
	reader, err := msg.Source()
	if err != nil {
		s.logError("Failed to read message for RETR command")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			s.logError("Failed to close message: %v", err)
		}
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Lines starting with . must be prefixed with another .
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		s.send(line)
	}

	if err = scanner.Err(); err != nil {
		s.logError("Failed to read message for RETR command")
		s.send(".")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	s.send(".")
}

// Send the headers plus the top N lines to the client
func (s *Session) sendMessageTop(msg storage.Message, lineCount int) {
	reader, err := msg.Source()
	if err != nil {
		s.logError("Failed to read message for RETR command")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			s.logError("Failed to close message: %v", err)
		}
	}()

	scanner := bufio.NewScanner(reader)
	inBody := false
	for scanner.Scan() {
		line := scanner.Text()
		// Lines starting with . must be prefixed with another .
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		if inBody {
			// Check if we need to send anymore lines
			if lineCount < 1 {
				break
			} else {
				lineCount--
			}
		} else {
			if line == "" {
				// We've hit the end of the header
				inBody = true
			}
		}
		s.send(line)
	}

	if err = scanner.Err(); err != nil {
		s.logError("Failed to read message for RETR command")
		s.send(".")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	s.send(".")
}

// Load the users mailbox
func (s *Session) loadMailbox() {
	m, err := s.server.store.GetMessages(s.user)
	if err != nil {
		s.logError("Failed to load messages for %v: %v", s.user, err)
	}
	s.messages = m
	s.retainAll()
}

// Reset retain flag to true for all messages
func (s *Session) retainAll() {
	s.retain = make([]bool, len(s.messages))
	for i := range s.retain {
		s.retain[i] = true
	}
	s.msgCount = len(s.messages)
}

// This would be considered the "UPDATE" state in the RFC, but it does not fit
// with our state-machine design here, since no commands are accepted - it just
// indicates that the session was closed cleanly and that deletes should be
// processed.
func (s *Session) processDeletes() {
	s.logInfo("Processing deletes")
	for i, msg := range s.messages {
		if !s.retain[i] {
			s.logTrace("Deleting %v", msg)
			if err := s.server.store.RemoveMessage(s.user, msg.ID()); err != nil {
				s.logWarn("Error deleting %v: %v", msg, err)
			}
		}
	}
}

func (s *Session) enterState(state State) {
	s.state = state
	s.logTrace("Entering state %v", state)
}

// Calculate the next read or write deadline based on maxIdleSeconds
func (s *Session) nextDeadline() time.Time {
	return time.Now().Add(s.server.timeout)
}

// Send requested message, store errors in Session.sendError
func (s *Session) send(msg string) {
	if err := s.conn.SetWriteDeadline(s.nextDeadline()); err != nil {
		s.sendError = err
		return
	}
	if _, err := fmt.Fprint(s.conn, msg+"\r\n"); err != nil {
		s.sendError = err
		s.logWarn("Failed to send: '%v'", msg)
		return
	}
	s.logTrace(">> %v >>", msg)
}

// readByteLine reads a line of input into the provided buffer. Does
// not reset the Buffer - please do so prior to calling.
func (s *Session) readByteLine(buf *bytes.Buffer) error {
	if err := s.conn.SetReadDeadline(s.nextDeadline()); err != nil {
		return err
	}
	for {
		line, err := s.reader.ReadBytes('\r')
		if err != nil {
			return err
		}
		if _, err = buf.Write(line); err != nil {
			return err
		}
		// Read the next byte looking for '\n'
		c, err := s.reader.ReadByte()
		if err != nil {
			return err
		}
		if err := buf.WriteByte(c); err != nil {
			return err
		}
		if c == '\n' {
			// We've reached the end of the line, return
			return nil
		}
		// Else, keep looking
	}
	// Should be unreachable
}

// Reads a line of input
func (s *Session) readLine() (line string, err error) {
	if err = s.conn.SetReadDeadline(s.nextDeadline()); err != nil {
		return "", err
	}
	line, err = s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	s.logTrace("<< %v <<", strings.TrimRight(line, "\r\n"))
	return line, nil
}

func (s *Session) parseCmd(line string) (cmd string, args []string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", nil, true
	}

	words := strings.Split(line, " ")
	return strings.ToUpper(words[0]), words[1:], true
}

func (s *Session) reset() {
	s.retainAll()
}

func (s *Session) ooSeq(cmd string) {
	s.send(fmt.Sprintf("-ERR Command %v is out of sequence", cmd))
	s.logWarn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (s *Session) logTrace(msg string, args ...interface{}) {
	log.Tracef("POP3[%v]<%v> %v", s.remoteHost, s.id, fmt.Sprintf(msg, args...))
}

func (s *Session) logInfo(msg string, args ...interface{}) {
	log.Infof("POP3[%v]<%v> %v", s.remoteHost, s.id, fmt.Sprintf(msg, args...))
}

func (s *Session) logWarn(msg string, args ...interface{}) {
	// Update metrics
	//expWarnsTotal.Add(1)
	log.Warnf("POP3[%v]<%v> %v", s.remoteHost, s.id, fmt.Sprintf(msg, args...))
}

func (s *Session) logError(msg string, args ...interface{}) {
	// Update metrics
	//expErrorsTotal.Add(1)
	log.Errorf("POP3[%v]<%v> %v", s.remoteHost, s.id, fmt.Sprintf(msg, args...))
}
