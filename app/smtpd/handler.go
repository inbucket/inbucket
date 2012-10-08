package smtpd

import (
	"bufio"
	"container/list"
	"fmt"
	"net"
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
	sendError  error
	state      State
	reader     *bufio.Reader
	from       string
	recipients *list.List
}

func NewSession(server *Server, id int, conn net.Conn) *Session {
	reader := bufio.NewReader(conn)
	return &Session{server: server, id: id, conn: conn, state: GREET, reader: reader}
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
	s.trace("Starting session <%v>", id)
	defer conn.Close()

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
				case "SEND", "SOML", "SAML", "VRFY", "EXPN", "HELP", "TURN":
					// These commands are not implemented in any state
					ss.send(fmt.Sprintf("502 %v command not implemented", cmd))
					ss.warn("Command %v not implemented by Inbucket", cmd)
					continue
				case "NOOP":
					ss.send("250 I have sucessfully done nothing")
					continue
				case "RSET":
					// Reset session
					ss.reset()
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
	if cmd == "HELO" {
		ss.send("250 Great, let's get this show on the road")
		ss.enterState(READY)
	} else {
		ss.ooSeq(cmd)
	}
}

// READY state -> waiting for MAIL
func (ss *Session) readyHandler(cmd string, arg string) {
	if cmd == "MAIL" {
		if (len(arg) < 6) || (strings.ToUpper(arg[0:5]) != "FROM:") {
			ss.send("501 Was expecting MAIL arg syntax of FROM:<address>")
			ss.warn("Bad MAIL argument: \"%v\"", arg)
			return
		}
		// This trim is probably too forgiving
		from := strings.Trim(arg[5:], "<> ")
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
	var msgSize uint64 = 0
	ss.send("354 Start mail input; end with <CRLF>.<CRLF>")
	for {
		line, err := ss.readLine()
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
		if line == ".\r\n" || line == ".\n" {
			// Mail data complete
			ss.send("250 Mail accepted for delivery")
			ss.info("Message size %v bytes", msgSize)
			ss.enterState(READY)
			return
		}
		if line != "" && line[0] == '.' {
			line = line[1:]
		}
		msgSize += uint64(len(line))
		// TODO: Add variable line to something!
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

func (ss *Session) reset() {
	ss.info("Resetting session state on RSET request")
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
	ss.server.trace("<%v> %v", ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) info(msg string, args ...interface{}) {
	ss.server.info("<%v> %v", ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) warn(msg string, args ...interface{}) {
	ss.server.warn("<%v> %v", ss.id, fmt.Sprintf(msg, args...))
}

func (ss *Session) error(msg string, args ...interface{}) {
	ss.server.error("<%v> %v", ss.id, fmt.Sprintf(msg, args...))
}
