package pop3

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	"STLS": true,
}

// Session defines an active POP3 session
type Session struct {
	*Server                      // Reference to the server we belong to.
	id         int               // Session ID number.
	conn       net.Conn          // Our network connection.
	remoteHost string            // IP address of client.
	sendError  error             // Used to bail out of read loop on send error.
	state      State             // Current session state.
	reader     *bufio.Reader     // Buffered reader for our net conn.
	user       string            // Mailbox name.
	messages   []storage.Message // Slice of messages in mailbox.
	retain     []bool            // Messages to retain upon UPDATE (true=retain).
	msgCount   int               // Number of undeleted messages.
	logger     zerolog.Logger    // Session specific logger.
	debug      bool              // Print network traffic to stdout.
}

// NewSession creates a new POP3 session
func NewSession(server *Server, id int, conn net.Conn, logger zerolog.Logger) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return &Session{
		Server:     server,
		id:         id,
		conn:       conn,
		state:      AUTHORIZATION,
		reader:     reader,
		remoteHost: host,
		logger:     logger,
		debug:      server.config.Debug,
	}
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
	logger := log.With().Str("module", "pop3").Str("remote", conn.RemoteAddr().String()).
		Int("session", id).Logger()
	logger.Debug().Msgf("ForceTLS: %t", s.config.ForceTLS)
	connToClose := conn
	if s.config.ForceTLS {
		logger.Debug().Msg("Setting up TLS for ForceTLS")
		tlsConn := tls.Server(conn, s.tlsConfig)
		s.tlsState = new(tls.ConnectionState)
		*s.tlsState = tlsConn.ConnectionState()
		conn = tlsConn
	}

	logger.Info().Msg("Starting POP3 session")
	defer func() {
		logger.Debug().Msg("closing at end of session")
		// Closing the tlsConn hangs.
		if err := connToClose.Close(); err != nil {
			logger.Warn().Err(err).Msg("Closing connection")
		}
		logger.Debug().Msg("End of session")
		s.wg.Done()
	}()

	ssn := NewSession(s, id, conn, logger)
	ssn.send(fmt.Sprintf("+OK Inbucket POP3 server ready <%v.%v@%v>", os.Getpid(),
		time.Now().Unix(), s.config.Domain))

	// This is our command reading loop
	for ssn.state != QUIT && ssn.sendError == nil {
		line, err := ssn.readLine()
		ssn.logger.Debug().Msgf("read %s", line)
		if err == nil {
			cmd, arg := ssn.parseCmd(line)
			// Commands we handle in any state
			if cmd == "CAPA" {
				// List our capabilities per RFC2449
				ssn.send("+OK Capability list follows")
				ssn.send("TOP")
				ssn.send("USER")
				ssn.send("UIDL")
				ssn.send("IMPLEMENTATION Inbucket")
				if s.tlsConfig != nil && s.tlsState == nil && !s.config.ForceTLS {
					ssn.send("STLS")
				}
				ssn.send(".")
				continue
			}

			// Check against valid SMTP commands
			if cmd == "" {
				ssn.send("-ERR Speak up")
				continue
			}

			if !commands[cmd] {
				ssn.send(fmt.Sprintf("-ERR Syntax error, %v command unrecognized", cmd))
				ssn.logger.Warn().Msgf("Unrecognized command: %v", cmd)
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

			ssn.logger.Error().Msgf("Session entered unexpected state %v", ssn.state)
			break
		} else {
			// readLine() returned an error
			if err == io.EOF {
				switch ssn.state {
				case AUTHORIZATION:
					// EOF is common here
					ssn.logger.Info().Msgf("Client closed connection (state %v)", ssn.state)
				default:
					ssn.logger.Warn().Msgf("Got EOF while in state %v", ssn.state)
				}
				break
			}

			// not an EOF
			ssn.logger.Warn().Msgf("Connection error: %v", err)
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
		ssn.logger.Warn().Msgf("Network send error: %v", ssn.sendError)
	}
	ssn.logger.Info().Msgf("Closing connection")
}

// AUTHORIZATION state
func (s *Session) authorizationHandler(cmd string, args []string) {
	switch cmd {
	case "QUIT":
		s.send("+OK Goodnight and good luck")
		s.logger.Debug().Msg("Quitting.")
		s.enterState(QUIT)

	case "STLS":
		if !s.Server.config.TLSEnabled || s.Server.config.ForceTLS {
			// Invalid command since TLS unconfigured.
			s.logger.Debug().Msgf("-ERR TLS unavailable on the server")
			s.send("-ERR TLS unavailable on the server")
			return
		}
		if s.tlsState != nil {
			// TLS state previously valid.
			s.logger.Debug().Msg("-ERR A TLS session already agreed upon.")
			s.send("-ERR A TLS session already agreed upon.")
			return
		}
		s.logger.Debug().Msg("Initiating TLS context.")

		// Start TLS connection handshake.
		s.send("+OK Begin TLS Negotiation")
		tlsConn := tls.Server(s.conn, s.Server.tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			s.logger.Error().Msgf("-ERR TLS handshake failed %v", err)
			s.ooSeq(cmd)
		}
		s.conn = tlsConn
		s.reader = bufio.NewReader(tlsConn)
		s.tlsState = new(tls.ConnectionState)
		*s.tlsState = tlsConn.ConnectionState()
		s.logger.Debug().Msgf("TLS set %v", *s.tlsState)

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
			s.logger.Warn().Msgf("Expected two arguments for APOP")
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
			s.logger.Warn().Msgf("STAT got an unexpected argument")
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
			s.logger.Warn().Msgf("LIST command had more than 1 argument")
			s.send("-ERR LIST command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				s.logger.Warn().Msgf("LIST command argument was not an integer")
				s.send("-ERR LIST command requires an integer argument")
				return
			}
			if msgNum < 1 {
				s.logger.Warn().Msgf("LIST command argument was less than 1")
				s.send("-ERR LIST argument must be greater than 0")
				return
			}
			if int(msgNum) > len(s.messages) {
				s.logger.Warn().Msgf("LIST command argument was greater than number of messages")
				s.send("-ERR LIST argument must not exceed the number of messages")
				return
			}
			if !s.retain[msgNum-1] {
				s.logger.Warn().Msgf("Client tried to LIST a message it had deleted")
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
			s.logger.Warn().Msgf("UIDL command had more than 1 argument")
			s.send("-ERR UIDL command must have zero or one argument")
			return
		}
		if len(args) == 1 {
			msgNum, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				s.logger.Warn().Msgf("UIDL command argument was not an integer")
				s.send("-ERR UIDL command requires an integer argument")
				return
			}
			if msgNum < 1 {
				s.logger.Warn().Msgf("UIDL command argument was less than 1")
				s.send("-ERR UIDL argument must be greater than 0")
				return
			}
			if int(msgNum) > len(s.messages) {
				s.logger.Warn().Msgf("UIDL command argument was greater than number of messages")
				s.send("-ERR UIDL argument must not exceed the number of messages")
				return
			}
			if !s.retain[msgNum-1] {
				s.logger.Warn().Msgf("Client tried to UIDL a message it had deleted")
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
			s.logger.Warn().Msgf("DELE command had invalid number of arguments")
			s.send("-ERR DELE command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logger.Warn().Msgf("DELE command argument was not an integer")
			s.send("-ERR DELE command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logger.Warn().Msgf("DELE command argument was less than 1")
			s.send("-ERR DELE argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logger.Warn().Msgf("DELE command argument was greater than number of messages")
			s.send("-ERR DELE argument must not exceed the number of messages")
			return
		}
		if s.retain[msgNum-1] {
			s.retain[msgNum-1] = false
			s.msgCount--
			s.send(fmt.Sprintf("+OK Deleted message %v", msgNum))
		} else {
			s.logger.Warn().Msgf("Client tried to DELE an already deleted message")
			s.send(fmt.Sprintf("-ERR Message %v has already been deleted", msgNum))
		}
	case "RETR":
		if len(args) != 1 {
			s.logger.Warn().Msgf("RETR command had invalid number of arguments")
			s.send("-ERR RETR command requires a single argument")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logger.Warn().Msgf("RETR command argument was not an integer")
			s.send("-ERR RETR command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logger.Warn().Msgf("RETR command argument was less than 1")
			s.send("-ERR RETR argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logger.Warn().Msgf("RETR command argument was greater than number of messages")
			s.send("-ERR RETR argument must not exceed the number of messages")
			return
		}
		s.send(fmt.Sprintf("+OK %v bytes follows", s.messages[msgNum-1].Size()))
		s.sendMessage(s.messages[msgNum-1])
	case "TOP":
		if len(args) != 2 {
			s.logger.Warn().Msgf("TOP command had invalid number of arguments")
			s.send("-ERR TOP command requires two arguments")
			return
		}
		msgNum, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			s.logger.Warn().Msgf("TOP command first argument was not an integer")
			s.send("-ERR TOP command requires an integer argument")
			return
		}
		if msgNum < 1 {
			s.logger.Warn().Msgf("TOP command first argument was less than 1")
			s.send("-ERR TOP first argument must be greater than 0")
			return
		}
		if int(msgNum) > len(s.messages) {
			s.logger.Warn().Msgf("TOP command first argument was greater than number of messages")
			s.send("-ERR TOP first argument must not exceed the number of messages")
			return
		}

		var lines int64
		lines, err = strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			s.logger.Warn().Msgf("TOP command second argument was not an integer")
			s.send("-ERR TOP command requires an integer argument")
			return
		}
		if lines < 0 {
			s.logger.Warn().Msgf("TOP command second argument was negative")
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
		s.send("+OK I have successfully done nothing")
	case "RSET":
		// Reset session, don't actually delete anything I told you to
		s.logger.Debug().Msgf("Resetting session state on RSET request")
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
		s.logger.Error().Msgf("Failed to read message for RETR command")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			s.logger.Error().Msgf("Failed to close message: %v", err)
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
		s.logger.Error().Msgf("Failed to read message for RETR command")
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
		s.logger.Error().Msgf("Failed to read message for RETR command")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			s.logger.Error().Msgf("Failed to close message: %v", err)
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
		s.logger.Error().Msgf("Failed to read message for RETR command")
		s.send(".")
		s.send("-ERR Failed to RETR that message, internal error")
		return
	}
	s.send(".")
}

// Load the users mailbox
func (s *Session) loadMailbox() {
	s.logger = s.logger.With().Str("mailbox", s.user).Logger()
	m, err := s.store.GetMessages(s.user)
	if err != nil {
		s.logger.Error().Msgf("Failed to load messages for %v: %v", s.user, err)
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
	s.logger.Info().Msgf("Processing deletes")
	for i, msg := range s.messages {
		if !s.retain[i] {
			s.logger.Debug().Str("id", msg.ID()).Msg("Deleting message")
			if err := s.store.RemoveMessage(s.user, msg.ID()); err != nil {
				s.logger.Warn().Str("id", msg.ID()).Err(err).Msg("Error deleting message")
			}
		}
	}
}

func (s *Session) enterState(state State) {
	s.state = state
	s.logger.Debug().Msgf("Entering state %v", state)
}

// nextDeadline calculates the next read or write deadline based on configured timeout.
func (s *Session) nextDeadline() time.Time {
	return time.Now().Add(s.config.Timeout)
}

// Send requested message, store errors in Session.sendError
func (s *Session) send(msg string) {
	if err := s.conn.SetWriteDeadline(s.nextDeadline()); err != nil {
		s.sendError = err
		return
	}
	if _, err := fmt.Fprint(s.conn, msg+"\r\n"); err != nil {
		s.sendError = err
		s.logger.Warn().Msgf("Failed to send: %q", msg)
		return
	}
	if s.debug {
		fmt.Printf("%04d > %v\n", s.id, msg)
	}
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
	if s.debug {
		fmt.Printf("%04d   %v\n", s.id, strings.TrimRight(line, "\r\n"))
	}
	return line, nil
}

func (s *Session) parseCmd(line string) (cmd string, args []string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", nil
	}

	words := strings.Split(line, " ")
	return strings.ToUpper(words[0]), words[1:]
}

func (s *Session) reset() {
	s.retainAll()
}

func (s *Session) ooSeq(cmd string) {
	s.send(fmt.Sprintf("-ERR Command %v is out of sequence", cmd))
	s.logger.Warn().Msgf("Wasn't expecting %v here", cmd)
}
