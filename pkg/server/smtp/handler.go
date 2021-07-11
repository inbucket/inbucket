package smtp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// State tracks the current mode of our SMTP state machine.
type State int

const (
	// timeStampFormat to use in Received header.
	timeStampFormat = "Mon, 02 Jan 2006 15:04:05 -0700 (MST)"

	// GREET State: Waiting for HELO
	GREET State = iota
	// READY State: Got HELO, waiting for MAIL
	READY
	// LOGIN State: Got AUTH LOGIN command, expecting Username
	LOGIN
	// PASSWORD State: Got Username, expecting password
	PASSWORD
	// MAIL State: Got MAIL, accepting RCPTs
	MAIL
	// DATA State: Got DATA, waiting for "."
	DATA
	// QUIT State: Client requested end of session
	QUIT

	// Messages sent to user during LOGIN auth procedure
	// Can vary, but values are taken directly from spec
	// https://tools.ietf.org/html/draft-murchison-sasl-login-00

	//usernameChallenge sent when inviting user to provide username. Is base64 encoded string `User Name`
	usernameChallenge = "VXNlciBOYW1lAA=="
	//passwordChallenge sent when inviting user to provide password. Is base64 encoded string `Password`
	passwordChallenge = "UGFzc3dvcmQA"
)

// fromRegex captures the from address and optional BODY=8BITMIME clause.  Matches FROM, while
// accepting '>' as quoted pair and in double quoted strings (?i) makes the regex case insensitive,
// (?:) is non-grouping sub-match
var fromRegex = regexp.MustCompile(
	"(?i)^FROM:\\s*<((?:(?:\\\\>|[^>])+|\"[^\"]+\"@[^>])+)?>( [\\w= ]+)?$")

func (s State) String() string {
	switch s {
	case GREET:
		return "GREET"
	case READY:
		return "READY"
	case MAIL:
		return "MAIL"
	case DATA:
		return "DATA"
	case QUIT:
		return "QUIT"
	}
	return "Unknown"
}

var commands = map[string]bool{
	"HELO":     true,
	"EHLO":     true,
	"MAIL":     true,
	"RCPT":     true,
	"DATA":     true,
	"RSET":     true,
	"SEND":     true,
	"SOML":     true,
	"SAML":     true,
	"VRFY":     true,
	"EXPN":     true,
	"HELP":     true,
	"NOOP":     true,
	"QUIT":     true,
	"TURN":     true,
	"STARTTLS": true,
	"AUTH":     true,
}

// Session holds the state of an SMTP session
type Session struct {
	*Server                          // Server this session belongs to.
	id           int                 // Session ID.
	conn         net.Conn            // TCP connection.
	remoteDomain string              // Remote domain from HELO command.
	remoteHost   string              // Remote host.
	sendError    error               // Last network send error.
	state        State               // Session state machine.
	reader       *bufio.Reader       // Buffered reading for TCP conn.
	from         string              // Sender from MAIL command.
	recipients   []*policy.Recipient // Recipients from RCPT commands.
	logger       zerolog.Logger      // Session specific logger.
	debug        bool                // Print network traffic to stdout.
	tlsState     *tls.ConnectionState
	text         *textproto.Conn
}

// NewSession creates a new Session for the given connection
func NewSession(server *Server, id int, conn net.Conn, logger zerolog.Logger) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	return &Session{
		Server:     server,
		id:         id,
		conn:       conn,
		state:      GREET,
		reader:     reader,
		remoteHost: host,
		recipients: make([]*policy.Recipient, 0),
		logger:     logger,
		debug:      server.config.Debug,
		text:       textproto.NewConn(conn),
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
	logger := log.Hook(logHook{}).With().
		Str("module", "smtp").
		Str("remote", conn.RemoteAddr().String()).
		Int("session", id).Logger()
	logger.Info().Msg("Starting SMTP session")
	expConnectsCurrent.Add(1)
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Warn().Err(err).Msg("Closing connection")
		}
		s.wg.Done()
		expConnectsCurrent.Add(-1)
	}()

	ssn := NewSession(s, id, conn, logger)
	ssn.greet()

	// This is our command reading loop
	for ssn.state != QUIT && ssn.sendError == nil {
		if ssn.state == DATA {
			// Special case, does not use SMTP command format
			ssn.dataHandler()
			continue
		}
		line, err := ssn.readLine()
		if err == nil {
			//Handle LOGIN/PASSWORD states here, because they don't expect a command
			switch ssn.state {
			case LOGIN:
				ssn.loginHandler(line)
				continue
			case PASSWORD:
				ssn.passwordHandler(line)
				continue
			}

			if cmd, arg, ok := ssn.parseCmd(line); ok {
				// Check against valid SMTP commands
				if cmd == "" {
					ssn.send("500 Speak up")
					continue
				}
				if !commands[cmd] {
					ssn.send(fmt.Sprintf("500 Syntax error, %v command unrecognized", cmd))
					ssn.logger.Warn().Msgf("Unrecognized command: %v", cmd)
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "SEND", "SOML", "SAML", "EXPN", "HELP", "TURN":
					// These commands are not implemented in any state
					ssn.send(fmt.Sprintf("502 %v command not implemented", cmd))
					ssn.logger.Warn().Msgf("Command %v not implemented by Inbucket", cmd)
					continue
				case "VRFY":
					ssn.send("252 Cannot VRFY user, but will accept message")
					continue
				case "NOOP":
					ssn.send("250 I have sucessfully done nothing")
					continue
				case "RSET":
					// Reset session
					ssn.logger.Debug().Msgf("Resetting session state on RSET request")
					ssn.reset()
					ssn.send("250 Session reset")
					continue
				case "QUIT":
					ssn.send("221 Goodnight and good luck")
					ssn.enterState(QUIT)
					continue
				}

				// Send command to handler for current state
				switch ssn.state {
				case GREET:
					ssn.greetHandler(cmd, arg)
					continue
				case READY:
					ssn.readyHandler(cmd, arg)
					continue
				case MAIL:
					ssn.mailHandler(cmd, arg)
					continue
				}
				ssn.logger.Error().Msgf("Session entered unexpected state %v", ssn.state)
				break
			} else {
				ssn.send("500 Syntax error, command garbled")
			}
		} else {
			// readLine() returned an error
			if err == io.EOF {
				switch ssn.state {
				case GREET, READY:
					// EOF is common here
					ssn.logger.Info().Msgf("Client closed connection (state %v)", ssn.state)
				default:
					ssn.logger.Warn().Msgf("Got EOF while in state %v", ssn.state)
				}
				break
			}
			// Not an EOF
			ssn.logger.Warn().Msgf("Connection error: %v", err)
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ssn.send("221 Idle timeout, bye bye")
					break
				}
			}
			ssn.send("221 Connection error, sorry")
			break
		}
	}
	if ssn.sendError != nil {
		ssn.logger.Warn().Msgf("Network send error: %v", ssn.sendError)
	}
	ssn.logger.Info().Msgf("Closing connection")
}

// GREET state -> waiting for HELO
func (s *Session) greetHandler(cmd string, arg string) {
	const readyBanner = "Great, let's get this show on the road"
	switch cmd {
	case "HELO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			s.send("501 Domain/address argument required for HELO")
			return
		}
		s.remoteDomain = domain
		s.send("250 " + readyBanner)
		s.enterState(READY)
	case "EHLO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			s.send("501 Domain/address argument required for EHLO")
			return
		}
		s.remoteDomain = domain
		// Features before SIZE per RFC
		s.send("250-" + readyBanner)
		s.send("250-8BITMIME")
		s.send("250-AUTH PLAIN LOGIN")
		if s.Server.config.TLSEnabled && s.Server.tlsConfig != nil && s.tlsState == nil {
			s.send("250-STARTTLS")
		}
		s.send(fmt.Sprintf("250 SIZE %v", s.config.MaxMessageBytes))
		s.enterState(READY)
	default:
		s.ooSeq(cmd)
	}
}

func parseHelloArgument(arg string) (string, error) {
	domain := arg
	if idx := strings.IndexRune(arg, ' '); idx >= 0 {
		domain = arg[:idx]
	}
	if domain == "" {
		return "", fmt.Errorf("Invalid domain")
	}
	return domain, nil
}

func (s *Session) loginHandler(line string) {
	if len(line) == 0 {
		s.send("500 invalid Username")
		s.enterState(READY)
		return
	}
	s.send(fmt.Sprintf("334 %v", passwordChallenge))
	s.enterState(PASSWORD)
}

func (s *Session) passwordHandler(line string) {
	if len(line) == 0 {
		s.send("500 invalid Password")
		s.enterState(READY)
		return
	}
	s.send("235 Authentication successful")
	s.enterState(READY)
}

// READY state -> waiting for MAIL
// AUTH can change
func (s *Session) readyHandler(cmd string, arg string) {
	if cmd == "STARTTLS" {
		if !s.Server.config.TLSEnabled {
			// Invalid command since TLS unconfigured.
			s.logger.Debug().Msgf("454 TLS unavailable on the server")
			s.send("454 TLS unavailable on the server")
			return
		}
		if s.tlsState != nil {
			// TLS state previously valid.
			s.logger.Debug().Msg("454 A TLS session already agreed upon.")
			s.send("454 A TLS session already agreed upon.")
			return
		}
		s.logger.Debug().Msg("Initiating TLS context.")

		// Start TLS connection handshake.
		s.send("220 STARTTLS")
		tlsConn := tls.Server(s.conn, s.Server.tlsConfig)
		s.conn = tlsConn
		s.text = textproto.NewConn(s.conn)
		s.tlsState = new(tls.ConnectionState)
		*s.tlsState = tlsConn.ConnectionState()
		s.enterState(GREET)
	} else if cmd == "AUTH" {
		args := strings.SplitN(arg, " ", 3)
		authMethod := args[0]
		switch authMethod {
		case "PLAIN":
			{
				if len(args) != 2 {
					s.send("500 Bad auth arguments")
					s.logger.Warn().Msgf("Bad auth attempt: %q", arg)
					return
				}
				s.logger.Info().Msgf("Accepting credentials: %q", args[1])
				s.send("235 2.7.0 Authentication successful")
				return
			}
		case "LOGIN":
			{
				s.send(fmt.Sprintf("334 %v", usernameChallenge))
				s.enterState(LOGIN)
				return
			}
		default:
			{
				s.send(fmt.Sprintf("500 Unsupported AUTH method: %v", authMethod))
				return
			}
		}
	} else if cmd == "MAIL" {
		// Capture group 1: from address.  2: optional params.
		m := fromRegex.FindStringSubmatch(arg)
		if m == nil {
			s.send("501 Was expecting MAIL arg syntax of FROM:<address>")
			s.logger.Warn().Msgf("Bad MAIL argument: %q", arg)
			return
		}
		from := m[1]
		if _, _, err := policy.ParseEmailAddress(from); from != "" && err != nil {
			s.send("501 Bad sender address syntax")
			s.logger.Warn().Msgf("Bad address as MAIL arg: %q, %s", from, err)
			return
		}
		if from == "" {
			from = "unspecified"
		}

		// This is where the client may put BODY=8BITMIME, but we already
		// read the DATA as bytes, so it does not effect our processing.
		if m[2] != "" {
			args, ok := s.parseArgs(m[2])
			if !ok {
				s.send("501 Unable to parse MAIL ESMTP parameters")
				s.logger.Warn().Msgf("Bad MAIL argument: %q", arg)
				return
			}
			if args["SIZE"] != "" {
				size, err := strconv.ParseInt(args["SIZE"], 10, 32)
				if err != nil {
					s.send("501 Unable to parse SIZE as an integer")
					s.logger.Warn().Msgf("Unable to parse SIZE %q as an integer", args["SIZE"])
					return
				}
				if int(size) > s.config.MaxMessageBytes {
					s.send("552 Max message size exceeded")
					s.logger.Warn().Msgf("Client wanted to send oversized message: %v", args["SIZE"])
					return
				}
			}
		}
		s.from = from
		s.logger.Info().Msgf("Mail from: %v", from)
		s.send(fmt.Sprintf("250 Roger, accepting mail from <%v>", from))
		s.enterState(MAIL)
	} else if cmd == "EHLO" {
		// Reset session
		s.logger.Debug().Msgf("Resetting session state on EHLO request")
		s.reset()
		s.send("250 Session reset")
	} else {
		s.ooSeq(cmd)
	}
}

// MAIL state -> waiting for RCPTs followed by DATA
func (s *Session) mailHandler(cmd string, arg string) {
	switch cmd {
	case "RCPT":
		if (len(arg) < 4) || (strings.ToUpper(arg[0:3]) != "TO:") {
			s.send("501 Was expecting RCPT arg syntax of TO:<address>")
			s.logger.Warn().Msgf("Bad RCPT argument: %q", arg)
			return
		}
		addr := strings.Trim(arg[3:], "<> ")
		recip, err := s.addrPolicy.NewRecipient(addr)
		if err != nil {
			s.send("501 Bad recipient address syntax")
			s.logger.Warn().Str("to", addr).Err(err).Msg("Bad address as RCPT arg")
			return
		}
		if !recip.ShouldAccept() {
			s.logger.Warn().Str("to", addr).Msg("Rejecting recipient domain")
			s.send("550 Relay not permitted")
			return
		}
		if len(s.recipients) >= s.config.MaxRecipients {
			s.logger.Warn().Msgf("Limit of %v recipients exceeded", s.config.MaxRecipients)
			s.send(fmt.Sprintf("552 Limit of %v recipients exceeded", s.config.MaxRecipients))
			return
		}
		s.recipients = append(s.recipients, recip)
		s.logger.Debug().Str("to", addr).Msg("Recipient added")
		s.send(fmt.Sprintf("250 I'll make sure <%v> gets this", addr))
		return
	case "DATA":
		if arg != "" {
			s.send("501 DATA command should not have any arguments")
			s.logger.Warn().Msgf("Got unexpected args on DATA: %q", arg)
			return
		}
		if len(s.recipients) == 0 {
			// DATA out of sequence
			s.ooSeq(cmd)
			return
		}
		s.enterState(DATA)
		return
	case "EHLO":
		// Reset session
		s.logger.Debug().Msgf("Resetting session state on EHLO request")
		s.reset()
		s.send("250 Session reset")
		return
	}
	s.ooSeq(cmd)
}

// DATA
func (s *Session) dataHandler() {
	s.send("354 Start mail input; end with <CRLF>.<CRLF>")
	msgBuf, err := s.readDataBlock()
	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				s.send("221 Idle timeout, bye bye")
			}
		}
		s.logger.Warn().Msgf("Error: %v while reading", err)
		s.enterState(QUIT)
		return
	}
	mailData := bytes.NewBuffer(msgBuf)

	// Mail data complete.
	tstamp := time.Now().Format(timeStampFormat)
	for _, recip := range s.recipients {
		if recip.ShouldStore() {
			// Generate Received header.
			prefix := fmt.Sprintf("Received: from %s ([%s]) by %s\r\n  for <%s>; %s\r\n",
				s.remoteDomain, s.remoteHost, s.config.Domain, recip.Address.Address,
				tstamp)

			// Deliver message.
			_, err := s.manager.Deliver(
				recip, s.from, s.recipients, prefix, mailData.Bytes())
			if err != nil {
				s.logger.Error().Msgf("delivery for %v: %v", recip.LocalPart, err)
				s.send(fmt.Sprintf("451 Failed to store message for %v", recip.LocalPart))
				s.reset()
				return
			}
		}
		expReceivedTotal.Add(1)
	}
	s.send("250 Mail accepted for delivery")
	s.logger.Info().Msgf("Message size %v bytes", mailData.Len())
	s.reset()
	return
}

func (s *Session) enterState(state State) {
	s.state = state
	s.logger.Debug().Msgf("Entering state %v", state)
}

func (s *Session) greet() {
	s.send(fmt.Sprintf("220 %v Inbucket SMTP ready", s.config.Domain))
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
	if err := s.text.PrintfLine("%s", msg); err != nil {
		s.sendError = err
		s.logger.Warn().Msgf("Failed to send: %q", msg)
		return
	}
	if s.debug {
		fmt.Printf("%04d > %v\n", s.id, msg)
	}
}

// readDataBlock reads message DATA until `.` using the textproto pkg.
func (s *Session) readDataBlock() ([]byte, error) {
	if err := s.conn.SetReadDeadline(s.nextDeadline()); err != nil {
		return nil, err
	}
	b, err := s.text.ReadDotBytes()
	if err != nil {
		return nil, err
	}
	if s.debug {
		fmt.Printf("%04d   Received %d bytes\n", s.id, len(b))
	}
	return b, err
}

// readLine reads a line of input respecting deadlines.
func (s *Session) readLine() (line string, err error) {
	if err = s.conn.SetReadDeadline(s.nextDeadline()); err != nil {
		return "", err
	}
	line, err = s.text.ReadLine()
	if err != nil {
		return "", err
	}
	if s.debug {
		fmt.Printf("%04d   %v\n", s.id, strings.TrimRight(line, "\r\n"))
	}
	return line, nil
}

func (s *Session) parseCmd(line string) (cmd string, arg string, ok bool) {
	line = strings.TrimRight(line, "\r\n")

	// Find length of command or entire line.
	hasArg := true
	l := strings.IndexByte(line, ' ')
	if l == -1 {
		hasArg = false
		l = len(line)
	}

	switch {
	case l == 0:
		return "", "", true
	case l < 4:
		s.logger.Warn().Msgf("Command too short: %q", line)
		return "", "", false
	}

	if hasArg {
		return strings.ToUpper(line[0:l]), strings.Trim(line[l+1:], " "), true
	}

	return strings.ToUpper(line), "", true
}

// parseArgs takes the arguments proceeding a command and files them
// into a map[string]string after uppercasing each key.  Sample arg
// string:
//		" BODY=8BITMIME SIZE=1024"
// The leading space is mandatory.
func (s *Session) parseArgs(arg string) (args map[string]string, ok bool) {
	args = make(map[string]string)
	re := regexp.MustCompile(` (\w+)=(\w+)`)
	pm := re.FindAllStringSubmatch(arg, -1)
	if pm == nil {
		s.logger.Warn().Msgf("Failed to parse arg string: %q", arg)
		return nil, false
	}
	for _, m := range pm {
		args[strings.ToUpper(m[1])] = m[2]
	}
	s.logger.Debug().Msgf("ESMTP params: %v", args)
	return args, true
}

func (s *Session) reset() {
	s.enterState(READY)
	s.from = ""
	s.recipients = nil
}

func (s *Session) ooSeq(cmd string) {
	s.send(fmt.Sprintf("503 Command %v is out of sequence", cmd))
	s.logger.Warn().Msgf("Wasn't expecting %v here", cmd)
}
