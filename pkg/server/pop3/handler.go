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
	mailbox    storage.Mailbox   // Mailbox instance
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
	log.Infof("POP3 connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	//expConnectsCurrent.Add(1)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Error closing POP3 connection for <%v>: %v", id, err)
		}
		s.waitgroup.Done()
		//expConnectsCurrent.Add(-1)
	}()

	ses := NewSession(s, id, conn)
	ses.send(fmt.Sprintf("+OK Inbucket POP3 server ready <%v.%v@%v>", os.Getpid(),
		time.Now().Unix(), s.domain))

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
					ses.logWarn("Unrecognized command: %v", cmd)
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "CAPA":
					// List our capabilities per RFC2449
					ses.send("+OK Capability list follows")
					ses.send("TOP")
					ses.send("USER")
					ses.send("UIDL")
					ses.send("IMPLEMENTATION Inbucket")
					ses.send(".")
					continue
				}

				// Send command to handler for current state
				switch ses.state {
				case AUTHORIZATION:
					ses.authorizationHandler(cmd, arg)
					continue
				case TRANSACTION:
					ses.transactionHandler(cmd, arg)
					continue
				}
				ses.logError("Session entered unexpected state %v", ses.state)
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
					ses.logInfo("Client closed connection (state %v)", ses.state)
				default:
					ses.logWarn("Got EOF while in state %v", ses.state)
				}
				break
			}
			// not an EOF
			ses.logWarn("Connection error: %v", err)
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
		ses.logWarn("Network send error: %v", ses.sendError)
	}
	ses.logInfo("Closing connection")
}

// AUTHORIZATION state
func (ses *Session) authorizationHandler(cmd string, args []string) {
	switch cmd {
	case "QUIT":
		ses.send("+OK Goodnight and good luck")
		ses.enterState(QUIT)
	case "USER":
		if len(args) > 0 {
			ses.user = args[0]
			ses.send(fmt.Sprintf("+OK Hello %v, welcome to Inbucket", ses.user))
		} else {
			ses.send("-ERR Missing username argument")
		}
	case "PASS":
		if ses.user == "" {
			ses.ooSeq(cmd)
		} else {
			var err error
			ses.mailbox, err = ses.server.dataStore.MailboxFor(ses.user)
			if err != nil {
				ses.logError("Failed to open mailbox for %v", ses.user)
				ses.send(fmt.Sprintf("-ERR Failed to open mailbox for %v", ses.user))
				ses.enterState(QUIT)
				return
			}
			ses.loadMailbox()
			ses.send(fmt.Sprintf("+OK Found %v messages for %v", ses.msgCount, ses.user))
			ses.enterState(TRANSACTION)
		}
	case "APOP":
		if len(args) != 2 {
			ses.logWarn("Expected two arguments for APOP")
			ses.send("-ERR APOP requires two arguments")
			return
		}
		ses.user = args[0]
		var err error
		ses.mailbox, err = ses.server.dataStore.MailboxFor(ses.user)
		if err != nil {
			ses.logError("Failed to open mailbox for %v", ses.user)
			ses.send(fmt.Sprintf("-ERR Failed to open mailbox for %v", ses.user))
			ses.enterState(QUIT)
			return
		}
		ses.loadMailbox()
		ses.send(fmt.Sprintf("+OK Found %v messages for %v", ses.msgCount, ses.user))
		ses.enterState(TRANSACTION)
	default:
		ses.ooSeq(cmd)
	}
}

// TRANSACTION state
func (ses *Session) transactionHandler(cmd string, args []string) {
	switch cmd {
	case "STAT":
		if len(args) != 0 {
			ses.logWarn("STAT got an unexpected argument")
			ses.send("-ERR STAT command must have no arguments")
			return
		}
		var count int
		var size int64
		for i, msg := range ses.messages {
			if ses.retain[i] {
				count++
				size += msg.Size()
			}
		}
		ses.send(fmt.Sprintf("+OK %v %v", count, size))
	case "LIST":
		if len(args) > 1 {
			ses.logWarn("LIST command had more than 1 argument")
			ses.send("-ERR LIST command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				ses.logWarn("LIST command argument was not an integer")
				ses.send("-ERR LIST command requires an integer argument")
				return
			}
			if msgNum < 1 {
				ses.logWarn("LIST command argument was less than 1")
				ses.send("-ERR LIST argument must be greater than 0")
				return
			}
			if int(msgNum) > len(ses.messages) {
				ses.logWarn("LIST command argument was greater than number of messages")
				ses.send("-ERR LIST argument must not exceed the number of messages")
				return
			}
			if !ses.retain[msgNum-1] {
				ses.logWarn("Client tried to LIST a message it had deleted")
				ses.send(fmt.Sprintf("-ERR You deleted message %v", msgNum))
				return
			}
			ses.send(fmt.Sprintf("+OK %v %v", msgNum, ses.messages[msgNum-1].Size()))
		} else {
			ses.send(fmt.Sprintf("+OK Listing %v messages", ses.msgCount))
			for i, msg := range ses.messages {
				if ses.retain[i] {
					ses.send(fmt.Sprintf("%v %v", i+1, msg.Size()))
				}
			}
			ses.send(".")
		}
	case "UIDL":
		if len(args) > 1 {
			ses.logWarn("UIDL command had more than 1 argument")
			ses.send("-ERR UIDL command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				ses.logWarn("UIDL command argument was not an integer")
				ses.send("-ERR UIDL command requires an integer argument")
				return
			}
			if msgNum < 1 {
				ses.logWarn("UIDL command argument was less than 1")
				ses.send("-ERR UIDL argument must be greater than 0")
				return
			}
			if int(msgNum) > len(ses.messages) {
				ses.logWarn("UIDL command argument was greater than number of messages")
				ses.send("-ERR UIDL argument must not exceed the number of messages")
				return
			}
			if !ses.retain[msgNum-1] {
				ses.logWarn("Client tried to UIDL a message it had deleted")
				ses.send(fmt.Sprintf("-ERR You deleted message %v", msgNum))
				return
			}
			ses.send(fmt.Sprintf("+OK %v %v", msgNum, ses.messages[msgNum-1].ID()))
		} else {
			ses.send(fmt.Sprintf("+OK Listing %v messages", ses.msgCount))
			for i, msg := range ses.messages {
				if ses.retain[i] {
					ses.send(fmt.Sprintf("%v %v", i+1, msg.ID()))
				}
			}
			ses.send(".")
		}
	case "DELE":
		if len(args) != 1 {
			ses.logWarn("DELE command had invalid number of arguments")
			ses.send("-ERR DELE command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.logWarn("DELE command argument was not an integer")
			ses.send("-ERR DELE command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.logWarn("DELE command argument was less than 1")
			ses.send("-ERR DELE argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.logWarn("DELE command argument was greater than number of messages")
			ses.send("-ERR DELE argument must not exceed the number of messages")
			return
		}
		if ses.retain[msgNum-1] {
			ses.retain[msgNum-1] = false
			ses.msgCount--
			ses.send(fmt.Sprintf("+OK Deleted message %v", msgNum))
		} else {
			ses.logWarn("Client tried to DELE an already deleted message")
			ses.send(fmt.Sprintf("-ERR Message %v has already been deleted", msgNum))
		}
	case "RETR":
		if len(args) != 1 {
			ses.logWarn("RETR command had invalid number of arguments")
			ses.send("-ERR RETR command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.logWarn("RETR command argument was not an integer")
			ses.send("-ERR RETR command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.logWarn("RETR command argument was less than 1")
			ses.send("-ERR RETR argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.logWarn("RETR command argument was greater than number of messages")
			ses.send("-ERR RETR argument must not exceed the number of messages")
			return
		}
		ses.send(fmt.Sprintf("+OK %v bytes follows", ses.messages[msgNum-1].Size()))
		ses.sendMessage(ses.messages[msgNum-1])
	case "TOP":
		if len(args) != 2 {
			ses.logWarn("TOP command had invalid number of arguments")
			ses.send("-ERR TOP command requires two arguments")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.logWarn("TOP command first argument was not an integer")
			ses.send("-ERR TOP command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.logWarn("TOP command first argument was less than 1")
			ses.send("-ERR TOP first argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.logWarn("TOP command first argument was greater than number of messages")
			ses.send("-ERR TOP first argument must not exceed the number of messages")
			return
		}

		var lines int64
		lines, err = strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			ses.logWarn("TOP command second argument was not an integer")
			ses.send("-ERR TOP command requires an integer argument")
			return
		}
		if lines < 0 {
			ses.logWarn("TOP command second argument was negative")
			ses.send("-ERR TOP second argument must be non-negative")
			return
		}
		ses.send("+OK Top of message follows")
		ses.sendMessageTop(ses.messages[msgNum-1], int(lines))
	case "QUIT":
		ses.send("+OK We will process your deletes")
		ses.processDeletes()
		ses.enterState(QUIT)
	case "NOOP":
		ses.send("+OK I have sucessfully done nothing")
	case "RSET":
		// Reset session, don't actually delete anything I told you to
		ses.logTrace("Resetting session state on RSET request")
		ses.reset()
		ses.send("+OK Session reset")
	default:
		ses.ooSeq(cmd)
	}
}

// Send the contents of the message to the client
func (ses *Session) sendMessage(msg storage.Message) {
	reader, err := msg.RawReader()
	if err != nil {
		ses.logError("Failed to read message for RETR command")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			ses.logError("Failed to close message: %v", err)
		}
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Lines starting with . must be prefixed with another .
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		ses.send(line)
	}

	if err = scanner.Err(); err != nil {
		ses.logError("Failed to read message for RETR command")
		ses.send(".")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
	ses.send(".")
}

// Send the headers plus the top N lines to the client
func (ses *Session) sendMessageTop(msg storage.Message, lineCount int) {
	reader, err := msg.RawReader()
	if err != nil {
		ses.logError("Failed to read message for RETR command")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			ses.logError("Failed to close message: %v", err)
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
		ses.send(line)
	}

	if err = scanner.Err(); err != nil {
		ses.logError("Failed to read message for RETR command")
		ses.send(".")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
	ses.send(".")
}

// Load the users mailbox
func (ses *Session) loadMailbox() {
	var err error
	ses.messages, err = ses.mailbox.GetMessages()
	if err != nil {
		ses.logError("Failed to load messages for %v", ses.user)
	}

	ses.retainAll()
}

// Reset retain flag to true for all messages
func (ses *Session) retainAll() {
	ses.retain = make([]bool, len(ses.messages))
	for i := range ses.retain {
		ses.retain[i] = true
	}
	ses.msgCount = len(ses.messages)
}

// This would be considered the "UPDATE" state in the RFC, but it does not fit
// with our state-machine design here, since no commands are accepted - it just
// indicates that the session was closed cleanly and that deletes should be
// processed.
func (ses *Session) processDeletes() {
	ses.logInfo("Processing deletes")
	for i, msg := range ses.messages {
		if !ses.retain[i] {
			ses.logTrace("Deleting %v", msg)
			if err := msg.Delete(); err != nil {
				ses.logWarn("Error deleting %v: %v", msg, err)
			}
		}
	}
}

func (ses *Session) enterState(state State) {
	ses.state = state
	ses.logTrace("Entering state %v", state)
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
		ses.logWarn("Failed to send: '%v'", msg)
		return
	}
	ses.logTrace(">> %v >>", msg)
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
		if _, err = buf.Write(line); err != nil {
			return err
		}
		// Read the next byte looking for '\n'
		c, err := ses.reader.ReadByte()
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
func (ses *Session) readLine() (line string, err error) {
	if err = ses.conn.SetReadDeadline(ses.nextDeadline()); err != nil {
		return "", err
	}
	line, err = ses.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	ses.logTrace("<< %v <<", strings.TrimRight(line, "\r\n"))
	return line, nil
}

func (ses *Session) parseCmd(line string) (cmd string, args []string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", nil, true
	}

	words := strings.Split(line, " ")
	return strings.ToUpper(words[0]), words[1:], true
}

func (ses *Session) reset() {
	ses.retainAll()
}

func (ses *Session) ooSeq(cmd string) {
	ses.send(fmt.Sprintf("-ERR Command %v is out of sequence", cmd))
	ses.logWarn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (ses *Session) logTrace(msg string, args ...interface{}) {
	log.Tracef("POP3[%v]<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) logInfo(msg string, args ...interface{}) {
	log.Infof("POP3[%v]<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) logWarn(msg string, args ...interface{}) {
	// Update metrics
	//expWarnsTotal.Add(1)
	log.Warnf("POP3[%v]<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) logError(msg string, args ...interface{}) {
	// Update metrics
	//expErrorsTotal.Add(1)
	log.Errorf("POP3[%v]<%v> %v", ses.remoteHost, ses.id, fmt.Sprintf(msg, args...))
}
