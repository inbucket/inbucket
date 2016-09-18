package smtpd

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jhillyerd/inbucket/log"
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
	recipients   *list.List
}

// NewSession creates a new Session for the given connection
func NewSession(server *Server, id int, conn net.Conn) *Session {
	reader := bufio.NewReader(conn)
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	return &Session{server: server, id: id, conn: conn, state: GREET, reader: reader, remoteHost: host}
}

func (ss *Session) String() string {
	return fmt.Sprintf("Session{id: %v, state: %v}", ss.id, ss.state)
}

/* Session flow:
 *  1. Send initial greeting
 *  2. Receive cmd
 *  3. If good cmd, respond, optionally change state
 *  4. If bad cmd, respond error
 *  5. Goto 2
 */
func (s *Server) startSession(id int, conn net.Conn) {
	log.Infof("SMTP Connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	expConnectsCurrent.Add(1)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("Error closing connection for <%v>: %v", id, err)
		}
		s.waitgroup.Done()
		expConnectsCurrent.Add(-1)
	}()

	ss := NewSession(s, id, conn)
	ss.greet()

	// This is our command reading loop
	for ss.state != QUIT && ss.sendError == nil {
		if ss.state == DATA {
			// Special case, does not use SMTP command format
			ss.dataHandler()
			continue
		}
		line, err := ss.readLine()
		if err == nil {
			if cmd, arg, ok := ss.parseCmd(line); ok {
				// Check against valid SMTP commands
				if cmd == "" {
					ss.send("500 Speak up")
					continue
				}
				if !commands[cmd] {
					ss.send(fmt.Sprintf("500 Syntax error, %v command unrecognized", cmd))
					ss.logWarn("Unrecognized command: %v", cmd)
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "SEND", "SOML", "SAML", "EXPN", "HELP", "TURN":
					// These commands are not implemented in any state
					ss.send(fmt.Sprintf("502 %v command not implemented", cmd))
					ss.logWarn("Command %v not implemented by Inbucket", cmd)
					continue
				case "VRFY":
					ss.send("252 Cannot VRFY user, but will accept message")
					continue
				case "NOOP":
					ss.send("250 I have sucessfully done nothing")
					continue
				case "RSET":
					// Reset session
					ss.logTrace("Resetting session state on RSET request")
					ss.reset()
					ss.send("250 Session reset")
					continue
				case "QUIT":
					ss.send("221 Goodnight and good luck")
					ss.enterState(QUIT)
					continue
				}

				// Send command to handler for current state
				switch ss.state {
				case GREET:
					ss.greetHandler(cmd, arg)
					continue
				case READY:
					ss.readyHandler(cmd, arg)
					continue
				case MAIL:
					ss.mailHandler(cmd, arg)
					continue
				}
				ss.logError("Session entered unexpected state %v", ss.state)
				break
			} else {
				ss.send("500 Syntax error, command garbled")
			}
		} else {
			// readLine() returned an error
			if err == io.EOF {
				switch ss.state {
				case GREET, READY:
					// EOF is common here
					ss.logInfo("Client closed connection (state %v)", ss.state)
				default:
					ss.logWarn("Got EOF while in state %v", ss.state)
				}
				break
			}
			// not an EOF
			ss.logWarn("Connection error: %v", err)
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ss.send("221 Idle timeout, bye bye")
					break
				}
			}
			ss.send("221 Connection error, sorry")
			break
		}
	}
	if ss.sendError != nil {
		ss.logWarn("Network send error: %v", ss.sendError)
	}
	ss.logInfo("Closing connection")
}

// GREET state -> waiting for HELO
func (ss *Session) greetHandler(cmd string, arg string) {
	switch cmd {
	case "HELO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			ss.send("501 Domain/address argument required for HELO")
			return
		}
		ss.remoteDomain = domain
		ss.send("250 Great, let's get this show on the road")
		ss.enterState(READY)
	case "EHLO":
		domain, err := parseHelloArgument(arg)
		if err != nil {
			ss.send("501 Domain/address argument required for EHLO")
			return
		}
		ss.remoteDomain = domain
		ss.send("250-Great, let's get this show on the road")
		ss.send("250-8BITMIME")
		ss.send(fmt.Sprintf("250 SIZE %v", ss.server.maxMessageBytes))
		ss.enterState(READY)
	default:
		ss.ooSeq(cmd)
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
func (ss *Session) readyHandler(cmd string, arg string) {
	if cmd == "MAIL" {
		// Match FROM, while accepting '>' as quoted pair and in double quoted strings
		// (?i) makes the regex case insensitive, (?:) is non-grouping sub-match
		re := regexp.MustCompile("(?i)^FROM:\\s*<((?:\\\\>|[^>])+|\"[^\"]+\"@[^>]+)>( [\\w= ]+)?$")
		m := re.FindStringSubmatch(arg)
		if m == nil {
			ss.send("501 Was expecting MAIL arg syntax of FROM:<address>")
			ss.logWarn("Bad MAIL argument: %q", arg)
			return
		}
		from := m[1]
		if _, _, err := ParseEmailAddress(from); err != nil {
			ss.send("501 Bad sender address syntax")
			ss.logWarn("Bad address as MAIL arg: %q, %s", from, err)
			return
		}
		// This is where the client may put BODY=8BITMIME, but we already
		// read the DATA as bytes, so it does not effect our processing.
		if m[2] != "" {
			args, ok := ss.parseArgs(m[2])
			if !ok {
				ss.send("501 Unable to parse MAIL ESMTP parameters")
				ss.logWarn("Bad MAIL argument: %q", arg)
				return
			}
			if args["SIZE"] != "" {
				size, err := strconv.ParseInt(args["SIZE"], 10, 32)
				if err != nil {
					ss.send("501 Unable to parse SIZE as an integer")
					ss.logWarn("Unable to parse SIZE %q as an integer", args["SIZE"])
					return
				}
				if int(size) > ss.server.maxMessageBytes {
					ss.send("552 Max message size exceeded")
					ss.logWarn("Client wanted to send oversized message: %v", args["SIZE"])
					return
				}
			}
		}
		ss.from = from
		ss.recipients = list.New()
		ss.logInfo("Mail from: %v", from)
		ss.send(fmt.Sprintf("250 Roger, accepting mail from <%v>", from))
		ss.enterState(MAIL)
	} else {
		ss.ooSeq(cmd)
	}
}

// MAIL state -> waiting for RCPTs followed by DATA
func (ss *Session) mailHandler(cmd string, arg string) {
	switch cmd {
	case "RCPT":
		if (len(arg) < 4) || (strings.ToUpper(arg[0:3]) != "TO:") {
			ss.send("501 Was expecting RCPT arg syntax of TO:<address>")
			ss.logWarn("Bad RCPT argument: %q", arg)
			return
		}
		// This trim is probably too forgiving
		recip := strings.Trim(arg[3:], "<> ")
		if _, _, err := ParseEmailAddress(recip); err != nil {
			ss.send("501 Bad recipient address syntax")
			ss.logWarn("Bad address as RCPT arg: %q, %s", recip, err)
			return
		}
		if ss.recipients.Len() >= ss.server.maxRecips {
			ss.logWarn("Maximum limit of %v recipients reached", ss.server.maxRecips)
			ss.send(fmt.Sprintf("552 Maximum limit of %v recipients reached", ss.server.maxRecips))
			return
		}
		ss.recipients.PushBack(recip)
		ss.logInfo("Recipient: %v", recip)
		ss.send(fmt.Sprintf("250 I'll make sure <%v> gets this", recip))
		return
	case "DATA":
		if arg != "" {
			ss.send("501 DATA command should not have any arguments")
			ss.logWarn("Got unexpected args on DATA: %q", arg)
			return
		}
		if ss.recipients.Len() > 0 {
			// We have recipients, go to accept data
			ss.enterState(DATA)
			return
		}
		// DATA out of sequence
		ss.ooSeq(cmd)
		return
	}
	ss.ooSeq(cmd)
}

// DATA
func (ss *Session) dataHandler() {
	type RecipientDetails struct {
		address, localPart, domainPart string
		mailbox                        Mailbox
	}
	recipients := make([]RecipientDetails, 0, ss.recipients.Len())
	// Timestamp for Received header
	stamp := time.Now().Format(timeStampFormat)
	// Get a Mailbox and a new Message for each recipient
	msgSize := 0
	if ss.server.storeMessages {
		for e := ss.recipients.Front(); e != nil; e = e.Next() {
			recip := e.Value.(string)
			local, domain, err := ParseEmailAddress(recip)
			if err != nil {
				ss.logError("Failed to parse address for %q", recip)
				ss.send(fmt.Sprintf("451 Failed to open mailbox for %v", recip))
				ss.reset()
				return
			}
			if strings.ToLower(domain) != ss.server.domainNoStore {
				// Not our "no store" domain, so store the message
				mb, err := ss.server.dataStore.MailboxFor(local)
				if err != nil {
					ss.logError("Failed to open mailbox for %q: %s", local, err)
					ss.send(fmt.Sprintf("451 Failed to open mailbox for %v", local))
					ss.reset()
					return
				}
				recipients = append(recipients, RecipientDetails{recip, local, domain, mb})
			} else {
				log.Tracef("Not storing message for %q", recip)
			}
		}
	}

	ss.send("354 Start mail input; end with <CRLF>.<CRLF>")
	var lineBuf bytes.Buffer
	msgBuf := make([][]byte, 0, 1024)
	for {
		lineBuf.Reset()
		err := ss.readByteLine(&lineBuf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ss.send("221 Idle timeout, bye bye")
				}
			}
			ss.logWarn("Error: %v while reading", err)
			ss.enterState(QUIT)
			return
		}
		line := lineBuf.Bytes()
		if string(line) == ".\r\n" {
			// Mail data complete
			if ss.server.storeMessages {
				// Create a message for each valid recipient
				for _, r := range recipients {
					msg, err := r.mailbox.NewMessage()
					if err != nil {
						ss.logError("Failed to create message for %q: %s", r.localPart, err)
						ss.send(fmt.Sprintf("451 Failed to create message for %v", r.localPart))
						ss.reset()
						return
					}

					// Generate Received header
					recd := fmt.Sprintf("Received: from %s ([%s]) by %s\r\n  for <%s>; %s\r\n",
						ss.remoteDomain, ss.remoteHost, ss.server.domain, r.address, stamp)
					if err := msg.Append([]byte(recd)); err != nil {
						ss.logError("Failed to write received header for %q: %s", r.localPart, err)
						ss.send(fmt.Sprintf("451 Failed to create message for %v", r.localPart))
						ss.reset()
						return
					}
					// Append lines from msgBuf
					for _, line = range msgBuf {
						if err := msg.Append(line); err != nil {
							ss.logError("Failed to append to mailbox %v: %v",
								r.mailbox, err)
							ss.send("554 Something went wrong")
							ss.reset()
							// Should really cleanup the crap on filesystem
							return
						}
					}
					if err := msg.Close(); err != nil {
						ss.logError("Error: %v while writing message", err)
					}
					expReceivedTotal.Add(1)
				} // end for
			} else {
				expReceivedTotal.Add(1)
			}
			ss.send("250 Mail accepted for delivery")
			ss.logInfo("Message size %v bytes", msgSize)
			ss.reset()
			return
		}
		// SMTP RFC says remove leading periods from input
		if len(line) > 0 && line[0] == '.' {
			line = line[1:]
		}
		// Second append copies line/lineBuf so we can reuse it
		msgBuf = append(msgBuf, append([]byte{}, line...))
		msgSize += len(line)
		if msgSize > ss.server.maxMessageBytes {
			// Max message size exceeded
			ss.send("552 Maximum message size exceeded")
			ss.logWarn("Max message size exceeded while in DATA")
			ss.reset()
			// Should really cleanup the crap on filesystem (after issue #23)
			return
		}
	} // end for
}

func (ss *Session) enterState(state State) {
	ss.state = state
	ss.logTrace("Entering state %v", state)
}

func (ss *Session) greet() {
	ss.send(fmt.Sprintf("220 %v Inbucket SMTP ready", ss.server.domain))
}

// Calculate the next read or write deadline based on maxIdleSeconds
func (ss *Session) nextDeadline() time.Time {
	return time.Now().Add(time.Duration(ss.server.maxIdleSeconds) * time.Second)
}

// Send requested message, store errors in Session.sendError
func (ss *Session) send(msg string) {
	if err := ss.conn.SetWriteDeadline(ss.nextDeadline()); err != nil {
		ss.sendError = err
		return
	}
	if _, err := fmt.Fprint(ss.conn, msg+"\r\n"); err != nil {
		ss.sendError = err
		ss.logWarn("Failed to send: %q", msg)
		return
	}
	ss.logTrace(">> %v >>", msg)
}

// readByteLine reads a line of input into the provided buffer. Does
// not reset the Buffer - please do so prior to calling.
func (ss *Session) readByteLine(buf *bytes.Buffer) error {
	if err := ss.conn.SetReadDeadline(ss.nextDeadline()); err != nil {
		return err
	}
	for {
		line, err := ss.reader.ReadBytes('\r')
		if err != nil {
			return err
		}
		if _, err = buf.Write(line); err != nil {
			return err
		}
		// Read the next byte looking for '\n'
		c, err := ss.reader.ReadByte()
		if err != nil {
			return err
		}
		if err = buf.WriteByte(c); err != nil {
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
func (ss *Session) readLine() (line string, err error) {
	if err = ss.conn.SetReadDeadline(ss.nextDeadline()); err != nil {
		return "", err
	}
	line, err = ss.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	ss.logTrace("<< %v <<", strings.TrimRight(line, "\r\n"))
	return line, nil
}

func (ss *Session) parseCmd(line string) (cmd string, arg string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	l := len(line)
	switch {
	case l == 0:
		return "", "", true
	case l < 4:
		ss.logWarn("Command too short: %q", line)
		return "", "", false
	case l == 4:
		return strings.ToUpper(line), "", true
	case l == 5:
		// Too long to be only command, too short to have args
		ss.logWarn("Mangled command: %q", line)
		return "", "", false
	}
	// If we made it here, command is long enough to have args
	if line[4] != ' ' {
		// There wasn't a space after the command?
		ss.logWarn("Mangled command: %q", line)
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
func (ss *Session) parseArgs(arg string) (args map[string]string, ok bool) {
	args = make(map[string]string)
	re := regexp.MustCompile(" (\\w+)=(\\w+)")
	pm := re.FindAllStringSubmatch(arg, -1)
	if pm == nil {
		ss.logWarn("Failed to parse arg string: %q")
		return nil, false
	}
	for _, m := range pm {
		args[strings.ToUpper(m[1])] = m[2]
	}
	ss.logTrace("ESMTP params: %v", args)
	return args, true
}

func (ss *Session) reset() {
	ss.enterState(READY)
	ss.from = ""
	ss.recipients = nil
}

func (ss *Session) ooSeq(cmd string) {
	ss.send(fmt.Sprintf("503 Command %v is out of sequence", cmd))
	ss.logWarn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (ss *Session) logTrace(msg string, args ...interface{}) {
	log.Tracef("SMTP[%v]<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) logInfo(msg string, args ...interface{}) {
	log.Infof("SMTP[%v]<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) logWarn(msg string, args ...interface{}) {
	// Update metrics
	expWarnsTotal.Add(1)
	log.Warnf("SMTP[%v]<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) logError(msg string, args ...interface{}) {
	// Update metrics
	expErrorsTotal.Add(1)
	log.Errorf("SMTP[%v]<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}
