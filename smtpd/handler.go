package smtpd

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"github.com/jhillyerd/inbucket/log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type State int

const (
	GREET State = iota // Waiting for HELO
	READY              // Got HELO, waiting for MAIL
	MAIL               // Got MAIL, accepting RCPTs
	DATA               // Got DATA, waiting for "."
	QUIT               // Close session
)

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

type Session struct {
	server     *Server
	id         int
	conn       net.Conn
	remoteHost string
	sendError  error
	state      State
	reader     *bufio.Reader
	from       string
	recipients *list.List
}

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
	log.Info("Connection from %v, starting session <%v>", conn.RemoteAddr(), id)
	expConnectsCurrent.Add(1)
	defer conn.Close()
	defer expConnectsCurrent.Add(-1)

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
					continue
				}

				// Commands we handle in any state
				switch cmd {
				case "SEND", "SOML", "SAML", "EXPN", "HELP", "TURN":
					// These commands are not implemented in any state
					ss.send(fmt.Sprintf("502 %v command not implemented", cmd))
					ss.warn("Command %v not implemented by Inbucket", cmd)
					continue
				case "VRFY":
					ss.send("252 Cannot VRFY user, but will accept message")
					continue
				case "NOOP":
					ss.send("250 I have sucessfully done nothing")
					continue
				case "RSET":
					// Reset session
					ss.trace("Resetting session state on RSET request")
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
				ss.error("Session entered unexpected state %v", ss.state)
				break
			} else {
				ss.send("500 Syntax error, command garbled")
			}
		} else {
			// readLine() returned an error
			ss.error("Connection error: %v", err)
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
		ss.error("Network send error: %v", ss.sendError)
	}
	ss.info("Closing connection")
}

// GREET state -> waiting for HELO
func (ss *Session) greetHandler(cmd string, arg string) {
	switch cmd {
	case "HELO":
		ss.send("250 Great, let's get this show on the road")
		ss.enterState(READY)
	case "EHLO":
		ss.send("250-Great, let's get this show on the road")
		ss.send("250-8BITMIME")
		ss.send(fmt.Sprintf("250 SIZE %v", ss.server.maxMessageBytes))
		ss.enterState(READY)
	default:
		ss.ooSeq(cmd)
	}
}

// READY state -> waiting for MAIL
func (ss *Session) readyHandler(cmd string, arg string) {
	if cmd == "MAIL" {
		// (?i) makes the regex case insensitive
		re := regexp.MustCompile("(?i)^FROM:<([^>]+)>( [\\w= ]+)?$")
		m := re.FindStringSubmatch(arg)
		if m == nil {
			ss.send("501 Was expecting MAIL arg syntax of FROM:<address>")
			ss.warn("Bad MAIL argument: \"%v\"", arg)
			return
		}
		from := m[1]
		// This is where the client may put BODY=8BITMIME, but we already
		// ready the DATA as bytes, so it does not effect our processing.
		if m[2] != "" {
			args, ok := ss.parseArgs(m[2])
			if !ok {
				ss.send("501 Unable to parse MAIL ESMTP parameters")
				ss.warn("Bad MAIL argument: \"%v\"", arg)
				return
			}
			if args["SIZE"] != "" {
				size, err := strconv.ParseInt(args["SIZE"], 10, 32)
				if err != nil {
					ss.send("501 Unable to parse SIZE as an integer")
					ss.error("Unable to parse SIZE '%v' as an integer", args["SIZE"])
					return
				}
				if int(size) > ss.server.maxMessageBytes {
					ss.send("552 Max message size exceeded")
					ss.warn("Client wanted to send oversized message: %v", args["SIZE"])
					return
				}
			}
		}
		ss.from = from
		ss.recipients = list.New()
		ss.info("Mail from: %v", from)
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
			ss.warn("Bad RCPT argument: \"%v\"", arg)
			return
		}
		// This trim is probably too forgiving
		recip := strings.Trim(arg[3:], "<> ")
		if ss.recipients.Len() >= ss.server.maxRecips {
			ss.warn("Maximum limit of %v recipients reached", ss.server.maxRecips)
			ss.send(fmt.Sprintf("552 Maximum limit of %v recipients reached", ss.server.maxRecips))
			return
		}
		ss.recipients.PushBack(recip)
		ss.info("Recipient: %v", recip)
		ss.send(fmt.Sprintf("250 I'll make sure <%v> gets this", recip))
		return
	case "DATA":
		if arg != "" {
			ss.send("501 DATA command should not have any arguments")
			ss.warn("Got unexpected args on DATA: \"%v\"", arg)
			return
		}
		if ss.recipients.Len() > 0 {
			// We have recipients, go to accept data
			ss.enterState(DATA)
			return
		} else {
			// DATA out of sequence
			ss.ooSeq(cmd)
			return
		}
	}
	ss.ooSeq(cmd)
}

// DATA
func (ss *Session) dataHandler() {
	msgSize := 0

	// Get a Mailbox and a new Message for each recipient
	mailboxes := make([]*Mailbox, ss.recipients.Len())
	messages := make([]*Message, ss.recipients.Len())
	i := 0
	for e := ss.recipients.Front(); e != nil; e = e.Next() {
		recip := e.Value.(string)
		mb, err := ss.server.dataStore.MailboxFor(recip)
		if err != nil {
			ss.error("Failed to open mailbox for %v", recip)
			ss.send(fmt.Sprintf("451 Failed to open mailbox for %v", recip))
			ss.reset()
			return
		}
		mailboxes[i] = mb
		messages[i] = mb.NewMessage()
		i++
	}

	ss.send("354 Start mail input; end with <CRLF>.<CRLF>")
	var buf bytes.Buffer
	for {
		buf.Reset()
		err := ss.readByteLine(&buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					ss.send("221 Idle timeout, bye bye")
				}
			}
			ss.error("Error: %v while reading", err)
			ss.enterState(QUIT)
			return
		}
		line := buf.Bytes()
		if string(line) == ".\r\n" {
			// Mail data complete
			for _, m := range messages {
				m.Close()
				expDeliveredTotal.Add(1)
			}
			ss.send("250 Mail accepted for delivery")
			ss.info("Message size %v bytes", msgSize)
			ss.reset()
			return
		}
		// SMTP RFC says remove leading periods from input
		if len(line) > 0 && line[0] == '.' {
			line = line[1:]
		}
		msgSize += len(line)
		if msgSize > ss.server.maxMessageBytes {
			// Max message size exceeded
			ss.send("552 Maximum message size exceeded")
			ss.error("Max message size exceeded while in DATA")
			ss.reset()
			// TODO: Should really cleanup the crap on filesystem...
			return
		}
		// Append to message objects
		for i, m := range messages {
			if err := m.Append(line); err != nil {
				ss.error("Failed to append to mailbox %v: %v", mailboxes[i], err)
				ss.send("554 Something went wrong")
				ss.reset()
				// TODO: Should really cleanup the crap on filesystem...
				return
			}
		}
	}
}

func (ss *Session) enterState(state State) {
	ss.state = state
	ss.trace("Entering state %v", state)
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
		ss.error("Failed to send: \"%v\"", msg)
		return
	}
	ss.trace("Sent: \"%v\"", msg)
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
		buf.Write(line)
		// Read the next byte looking for '\n'
		c, err := ss.reader.ReadByte()
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
func (ss *Session) readLine() (line string, err error) {
	if err = ss.conn.SetReadDeadline(ss.nextDeadline()); err != nil {
		return "", err
	}
	line, err = ss.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	ss.trace("Read: \"%v\"", strings.TrimRight(line, "\r\n"))
	return line, nil
}

func (ss *Session) parseCmd(line string) (cmd string, arg string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	l := len(line)
	switch {
	case l == 0:
		return "", "", true
	case l < 4:
		ss.error("Command too short: \"%v\"", line)
		return "", "", false
	case l == 4:
		return strings.ToUpper(line), "", true
	case l == 5:
		// Too long to be only command, too short to have args
		ss.error("Mangled command: \"%v\"", line)
		return "", "", false
	}
	// If we made it here, command is long enough to have args
	if line[4] != ' ' {
		// There wasn't a space after the command?
		ss.error("Mangled command: \"%v\"", line)
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
		ss.error("Failed to parse arg string: '%v'")
		return nil, false
	}
	for _, m := range pm {
		args[strings.ToUpper(m[1])] = m[2]
	}
	ss.trace("ESMTP params: %v", args)
	return args, true
}

func (ss *Session) reset() {
	ss.enterState(READY)
	ss.from = ""
	ss.recipients = nil
}

func (ss *Session) ooSeq(cmd string) {
	ss.send(fmt.Sprintf("503 Command %v is out of sequence", cmd))
	ss.warn("Wasn't expecting %v here", cmd)
}

// Session specific logging methods
func (ss *Session) trace(msg string, args ...interface{}) {
	log.Trace("%v<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) info(msg string, args ...interface{}) {
	log.Info("%v<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) warn(msg string, args ...interface{}) {
	log.Warn("%v<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) error(msg string, args ...interface{}) {
	log.Error("%v<%v> %v", ss.remoteHost, ss.id, fmt.Sprintf(msg, args...))
}
