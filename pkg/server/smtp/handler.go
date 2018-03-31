package smtp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jhillyerd/inbucket/pkg/policy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// State tracks the current mode of our SMTP state machine
type State int

const (
	// GREET State: Waiting for HELO
	GREET State = iota
	// READY State: Got HELO, waiting for MAIL
	READY
	// MAIL State: Got MAIL, accepting RCPTs
	MAIL
	// DATA State: Got DATA, waiting for "."
	DATA
	// QUIT State: Client requested end of session
	QUIT
)

const timeStampFormat = "Mon, 02 Jan 2006 15:04:05 -0700 (MST)"

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
	"HELO": true,
	"EHLO": true,
	"MAIL": true,
	"RCPT": true,
	"DATA": true,
	"RSET": true,
	"SEND": true,
	"SOML": true,
	"SAML": true,
	"VRFY": true,
	"EXPN": true,
	"HELP": true,
	"NOOP": true,
	"QUIT": true,
	"TURN": true,
}

// Session holds the state of an SMTP session
type Session struct {
	server       *Server
	id           int
	conn         net.Conn
	remoteDomain string
	remoteHost   string
	sendError    error
	state        State
	reader       *bufio.Reader
	from         string
	recipients   []*policy.Recipient
	logger       zerolog.Logger // Session specific logger.
	debug        bool           // Print network traffic to stdout.
}

// NewSession creates a new Session for the given connection
func NewSession(server *Server, id int, conn net.Conn, logger zerolog.Logger) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return &Session{
		server:     server,
		id:         id,
		conn:       conn,
		state:      GREET,
		reader:     reader,
		remoteHost: host,
		recipients: make([]*policy.Recipient, 0),
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
		s.waitgroup.Done()
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
			// not an EOF
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
	switch cmd {
	case "HELO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			s.send("501 Domain/address argument required for HELO")
			return
		}
		s.remoteDomain = domain
		s.send("250 Great, let's get this show on the road")
		s.enterState(READY)
	case "EHLO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			s.send("501 Domain/address argument required for EHLO")
			return
		}
		s.remoteDomain = domain
		s.send("250-Great, let's get this show on the road")
		s.send("250-8BITMIME")
		s.send(fmt.Sprintf("250 SIZE %v", s.server.maxMessageBytes))
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

// READY state -> waiting for MAIL
func (s *Session) readyHandler(cmd string, arg string) {
	if cmd == "MAIL" {
		// Match FROM, while accepting '>' as quoted pair and in double quoted strings
		// (?i) makes the regex case insensitive, (?:) is non-grouping sub-match
		re := regexp.MustCompile("(?i)^FROM:\\s*<((?:\\\\>|[^>])+|\"[^\"]+\"@[^>]+)>( [\\w= ]+)?$")
		m := re.FindStringSubmatch(arg)
		if m == nil {
			s.send("501 Was expecting MAIL arg syntax of FROM:<address>")
			s.logger.Warn().Msgf("Bad MAIL argument: %q", arg)
			return
		}
		from := m[1]
		if _, _, err := policy.ParseEmailAddress(from); err != nil {
			s.send("501 Bad sender address syntax")
			s.logger.Warn().Msgf("Bad address as MAIL arg: %q, %s", from, err)
			return
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
				if int(size) > s.server.maxMessageBytes {
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
		// This trim is probably too forgiving
		addr := strings.Trim(arg[3:], "<> ")
		recip, err := s.server.apolicy.NewRecipient(addr)
		if err != nil {
			s.send("501 Bad recipient address syntax")
			s.logger.Warn().Msgf("Bad address as RCPT arg: %q, %s", addr, err)
			return
		}
		if len(s.recipients) >= s.server.maxRecips {
			s.logger.Warn().Msgf("Maximum limit of %v recipients reached", s.server.maxRecips)
			s.send(fmt.Sprintf("552 Maximum limit of %v recipients reached", s.server.maxRecips))
			return
		}
		s.recipients = append(s.recipients, recip)
		s.logger.Info().Msgf("Recipient: %v", addr)
		s.send(fmt.Sprintf("250 I'll make sure <%v> gets this", addr))
		return
	case "DATA":
		if arg != "" {
			s.send("501 DATA command should not have any arguments")
			s.logger.Warn().Msgf("Got unexpected args on DATA: %q", arg)
			return
		}
		if len(s.recipients) > 0 {
			// We have recipients, go to accept data
			s.enterState(DATA)
			return
		}
		// DATA out of sequence
		s.ooSeq(cmd)
		return
	}
	s.ooSeq(cmd)
}

// DATA
func (s *Session) dataHandler() {
	s.send("354 Start mail input; end with <CRLF>.<CRLF>")
	msgBuf := &bytes.Buffer{}
	for {
		lineBuf, err := s.readByteLine()
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
		if bytes.Equal(lineBuf, []byte(".\r\n")) || bytes.Equal(lineBuf, []byte(".\n")) {
			// Mail data complete.
			tstamp := time.Now().Format(timeStampFormat)
			for _, recip := range s.recipients {
				if recip.ShouldStore() {
					// Generate Received header.
					prefix := fmt.Sprintf("Received: from %s ([%s]) by %s\r\n  for <%s>; %s\r\n",
						s.remoteDomain, s.remoteHost, s.server.domain, recip.Address.Address,
						tstamp)
					// Deliver message.
					_, err := s.server.manager.Deliver(
						recip, s.from, s.recipients, prefix, msgBuf.Bytes())
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
			s.logger.Info().Msgf("Message size %v bytes", msgBuf.Len())
			s.reset()
			return
		}
		// RFC: remove leading periods from DATA.
		if len(lineBuf) > 0 && lineBuf[0] == '.' {
			lineBuf = lineBuf[1:]
		}
		msgBuf.Write(lineBuf)
		if msgBuf.Len() > s.server.maxMessageBytes {
			s.send("552 Maximum message size exceeded")
			s.logger.Warn().Msgf("Max message size exceeded while in DATA")
			s.reset()
			return
		}
	}
}

func (s *Session) enterState(state State) {
	s.state = state
	s.logger.Debug().Msgf("Entering state %v", state)
}

func (s *Session) greet() {
	s.send(fmt.Sprintf("220 %v Inbucket SMTP ready", s.server.domain))
}

// Calculate the next read or write deadline based on maxIdle
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
		s.logger.Warn().Msgf("Failed to send: %q", msg)
		return
	}
	if s.debug {
		fmt.Printf("%04d > %v\n", s.id, msg)
	}
}

// readByteLine reads a line of input, returns byte slice.
func (s *Session) readByteLine() ([]byte, error) {
	if err := s.conn.SetReadDeadline(s.nextDeadline()); err != nil {
		return nil, err
	}
	b, err := s.reader.ReadBytes('\n')
	if err == nil && s.debug {
		fmt.Printf("%04d   %s\n", s.id, bytes.TrimRight(b, "\r\n"))
	}
	return b, err
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

func (s *Session) parseCmd(line string) (cmd string, arg string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	l := len(line)
	switch {
	case l == 0:
		return "", "", true
	case l < 4:
		s.logger.Warn().Msgf("Command too short: %q", line)
		return "", "", false
	case l == 4:
		return strings.ToUpper(line), "", true
	case l == 5:
		// Too long to be only command, too short to have args
		s.logger.Warn().Msgf("Mangled command: %q", line)
		return "", "", false
	}
	// If we made it here, command is long enough to have args
	if line[4] != ' ' {
		// There wasn't a space after the command?
		s.logger.Warn().Msgf("Mangled command: %q", line)
		return "", "", false
	}
	// I'm not sure if we should trim the args or not, but we will for now
	return strings.ToUpper(line[0:4]), strings.Trim(line[5:], " "), true
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
		s.logger.Warn().Msgf("Failed to parse arg string: %q")
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
