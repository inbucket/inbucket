package pop3d

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type State int

const (
	AUTHORIZATION State = iota // The client must now identify and authenticate
	TRANSACTION                // Mailbox open, client may now issue commands
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
}

type Session struct {
	server     *Server         // Reference to the server we belong to
	id         int             // Session ID number
	conn       net.Conn        // Our network connection
	remoteHost string          // IP address of client
	sendError  error           // Used to bail out of read loop on send error
	state      State           // Current session state
	reader     *bufio.Reader   // Buffered reader for our net conn
	user       string          // Mailbox name
	mailbox    smtpd.Mailbox   // Mailbox instance
	messages   []smtpd.Message // Slice of messages in mailbox
	retain     []bool          // Messages to retain upon UPDATE (true=retain)
	msgCount   int             // Number of undeleted messages
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
	log.Info("POP3 connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	//expConnectsCurrent.Add(1)
	defer func() {
		conn.Close()
		s.waitgroup.Done()
		//expConnectsCurrent.Add(-1)
	}()

	ses := NewSession(s, id, conn)
	ses.send("+OK Inbucket POP3 server ready")

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
				case "APOP":
					// These commands are not implemented in any state
					ses.send(fmt.Sprintf("-ERR %v command not implemented", cmd))
					ses.warn("Command %v not implemented by Inbucket", cmd)
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
				ses.error("Failed to open mailbox for %v", ses.user)
				ses.send(fmt.Sprintf("-ERR Failed to open mailbox for %v", ses.user))
				ses.enterState(QUIT)
				return
			}
			ses.loadMailbox()
			ses.send(fmt.Sprintf("+OK Found %v messages for %v", ses.msgCount, ses.user))
			ses.enterState(TRANSACTION)
		}
	default:
		ses.ooSeq(cmd)
	}
}

// TRANSACTION state
func (ses *Session) transactionHandler(cmd string, args []string) {
	switch cmd {
	case "LIST":
		if len(args) > 1 {
			ses.warn("LIST command had more than 1 argument")
			ses.send("-ERR LIST command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				ses.warn("LIST command argument was not an integer")
				ses.send("-ERR LIST command requires an integer argument")
				return
			}
			if msgNum < 1 {
				ses.warn("LIST command argument was less than 1")
				ses.send("-ERR LIST argument must be greater than 0")
				return
			}
			if int(msgNum) > len(ses.messages) {
				ses.warn("LIST command argument was greater than number of messages")
				ses.send("-ERR LIST argument must not exceed the number of messages")
				return
			}
			if !ses.retain[msgNum-1] {
				ses.warn("Client tried to LIST a message it had deleted")
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
			ses.warn("UIDL command had more than 1 argument")
			ses.send("-ERR UIDL command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				ses.warn("UIDL command argument was not an integer")
				ses.send("-ERR UIDL command requires an integer argument")
				return
			}
			if msgNum < 1 {
				ses.warn("UIDL command argument was less than 1")
				ses.send("-ERR UIDL argument must be greater than 0")
				return
			}
			if int(msgNum) > len(ses.messages) {
				ses.warn("UIDL command argument was greater than number of messages")
				ses.send("-ERR UIDL argument must not exceed the number of messages")
				return
			}
			if !ses.retain[msgNum-1] {
				ses.warn("Client tried to UIDL a message it had deleted")
				ses.send(fmt.Sprintf("-ERR You deleted message %v", msgNum))
				return
			}
			ses.send(fmt.Sprintf("+OK %v %v", msgNum, ses.messages[msgNum-1].Id()))
		} else {
			ses.send(fmt.Sprintf("+OK Listing %v messages", ses.msgCount))
			for i, msg := range ses.messages {
				if ses.retain[i] {
					ses.send(fmt.Sprintf("%v %v", i+1, msg.Id()))
				}
			}
			ses.send(".")
		}
	case "DELE":
		if len(args) != 1 {
			ses.warn("DELE command had invalid number of arguments")
			ses.send("-ERR DELE command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.warn("DELE command argument was not an integer")
			ses.send("-ERR DELE command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.warn("DELE command argument was less than 1")
			ses.send("-ERR DELE argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.warn("DELE command argument was greater than number of messages")
			ses.send("-ERR DELE argument must not exceed the number of messages")
			return
		}
		if ses.retain[msgNum-1] {
			ses.retain[msgNum-1] = false
			ses.msgCount -= 1
			ses.send(fmt.Sprintf("+OK Deleted message %v", msgNum))
		} else {
			ses.warn("Client tried to DELE an already deleted message")
			ses.send(fmt.Sprintf("-ERR Message %v has already been deleted", msgNum))
		}
	case "RETR":
		if len(args) != 1 {
			ses.warn("RETR command had invalid number of arguments")
			ses.send("-ERR RETR command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.warn("RETR command argument was not an integer")
			ses.send("-ERR RETR command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.warn("RETR command argument was less than 1")
			ses.send("-ERR RETR argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.warn("RETR command argument was greater than number of messages")
			ses.send("-ERR RETR argument must not exceed the number of messages")
			return
		}
		ses.sendMessage(ses.messages[msgNum-1])
	case "TOP":
		if len(args) != 2 {
			ses.warn("TOP command had invalid number of arguments")
			ses.send("-ERR TOP command requires two arguments")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			ses.warn("TOP command first argument was not an integer")
			ses.send("-ERR TOP command requires an integer argument")
			return
		}
		if msgNum < 1 {
			ses.warn("TOP command first argument was less than 1")
			ses.send("-ERR TOP first argument must be greater than 0")
			return
		}
		if int(msgNum) > len(ses.messages) {
			ses.warn("TOP command first argument was greater than number of messages")
			ses.send("-ERR TOP first argument must not exceed the number of messages")
			return
		}

		var lines int64
		lines, err = strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			ses.warn("TOP command second argument was not an integer")
			ses.send("-ERR TOP command requires an integer argument")
			return
		}
		if lines < 0 {
			ses.warn("TOP command second argument was negative")
			ses.send("-ERR TOP second argument must be non-negative")
			return
		}
		ses.sendMessageTop(ses.messages[msgNum-1], int(lines))
	case "QUIT":
		ses.send("+OK We will process your deletes")
		ses.processDeletes()
		ses.enterState(QUIT)
	case "NOOP":
		ses.send("+OK I have sucessfully done nothing")
	case "RSET":
		// Reset session, don't actually delete anything I told you to
		ses.trace("Resetting session state on RSET request")
		ses.reset()
		ses.send("+OK Session reset")
	default:
		ses.ooSeq(cmd)
	}
}

// Send the contents of the message to the client
func (ses *Session) sendMessage(msg smtpd.Message) {
	reader, err := msg.RawReader()
	defer reader.Close()
	if err != nil {
		ses.error("Failed to read message for RETR command")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
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
		ses.error("Failed to read message for RETR command")
		ses.send(".")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
	ses.send(".")
}

// Send the headers plus the top N lines to the client
func (ses *Session) sendMessageTop(msg smtpd.Message, lineCount int) {
	reader, err := msg.RawReader()
	defer reader.Close()
	if err != nil {
		ses.error("Failed to read message for RETR command")
		ses.send("-ERR Failed to RETR that message, internal error")
		return
	}
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
				lineCount -= 1
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
		ses.error("Failed to read message for RETR command")
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
		ses.error("Failed to load messages for %v", ses.user)
	}

	ses.retainAll()
}

// Reset retain flag to true for all messages
func (ses *Session) retainAll() {
	ses.retain = make([]bool, len(ses.messages))
	for i, _ := range ses.retain {
		ses.retain[i] = true
	}
	ses.msgCount = len(ses.messages)
}

// This would be considered the "UPDATE" state in the RFC, but it does not fit
// with our state-machine design here, since no commands are accepted - it just
// indicates that the session was closed cleanly and that deletes should be
// processed.
func (ses *Session) processDeletes() {
	ses.trace("Processing deletes")
}

func (ses *Session) enterState(state State) {
	ses.state = state
	ses.trace("Entering state %v", state)
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
	ses.warn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (ses *Session) trace(msg string, args ...interface{}) {
	log.Trace("POP3<%v> %v", ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) info(msg string, args ...interface{}) {
	log.Info("POP3<%v> %v", ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) warn(msg string, args ...interface{}) {
	// Update metrics
	//expWarnsTotal.Add(1)
	log.Warn("POP3<%v> %v", ses.id, fmt.Sprintf(msg, args...))
}

func (ses *Session) error(msg string, args ...interface{}) {
	// Update metrics
	//expErrorsTotal.Add(1)
	log.Error("POP3<%v> %v", ses.id, fmt.Sprintf(msg, args...))
}
