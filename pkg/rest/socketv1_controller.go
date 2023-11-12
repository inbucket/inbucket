package rest

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/rest/model"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/inbucket/inbucket/v3/pkg/stringutil"
	"github.com/rs/zerolog/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWaitV1 = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriodV1 = (pongWaitV1 * 9) / 10

	// Time allowed to read the next pong message from the peer.
	pongWaitV1 = 60 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSizeV1 = 512
)

// options for gorilla connection upgrader
var upgraderV1 = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// msgListenerV1 handles messages from the msghub
type msgListenerV1 struct {
	hub     *msghub.Hub                // Global message hub
	c       chan event.MessageMetadata // Queue of messages from Receive()
	mailbox string                     // Name of mailbox to monitor, "" == all mailboxes
}

// newMsgListenerV1 creates a listener and registers it.  Optional mailbox parameter will restrict
// messages sent to WebSocket to that mailbox only.
func newMsgListenerV1(hub *msghub.Hub, mailbox string) *msgListenerV1 {
	ml := &msgListenerV1{
		hub:     hub,
		c:       make(chan event.MessageMetadata, 100),
		mailbox: mailbox,
	}
	hub.AddListener(ml)
	return ml
}

// Receive handles an incoming message.
func (ml *msgListenerV1) Receive(msg event.MessageMetadata) error {
	if ml.mailbox != "" && ml.mailbox != msg.Mailbox {
		// Did not match the watched mailbox name.
		return nil
	}
	ml.c <- msg
	return nil
}

// Delete handles a deleted message.
func (ml *msgListenerV1) Delete(mailbox string, id string) error {
	// Deletes are ignored in socketv1 API.
	return nil
}

// WSReader makes sure the websocket client is still connected, discards any messages from client
func (ml *msgListenerV1) WSReader(conn *websocket.Conn) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()

	defer ml.Close()

	conn.SetReadLimit(maxMessageSizeV1)
	if err := conn.SetReadDeadline(time.Now().Add(pongWaitV1)); err != nil {
		slog.Warn().Err(err).Msg("Failed to setup read deadline")
	}
	conn.SetPongHandler(func(string) error {
		slog.Debug().Msg("Got pong")
		if err := conn.SetReadDeadline(time.Now().Add(pongWaitV1)); err != nil {
			slog.Warn().Err(err).Msg("Failed to set read deadline in pong")
		}
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
			) {
				// Unexpected close code
				slog.Warn().Err(err).Msg("Socket error")
			} else {
				slog.Debug().Msg("Closing socket")
			}
			break
		}
	}
}

// WSWriter makes sure the websocket client is still connected
func (ml *msgListenerV1) WSWriter(conn *websocket.Conn) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()

	ticker := time.NewTicker(pingPeriodV1)
	defer func() {
		ticker.Stop()
		ml.Close()
	}()

	// Handle messages from hub until msgListener is closed
	for {
		select {
		case msg, ok := <-ml.c:
			if err := conn.SetWriteDeadline(time.Now().Add(writeWaitV1)); err != nil {
				slog.Warn().Err(err).Msg("Failed to set write deadline for msg")
			}
			if !ok {
				// msgListener closed, exit
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if conn.WriteJSON(metadataToHeader(&msg)) != nil {
				// Write failed
				return
			}
		case <-ticker.C:
			// Send ping
			if err := conn.SetWriteDeadline(time.Now().Add(writeWaitV1)); err != nil {
				slog.Warn().Err(err).Msg("Failed to set write deadline for ping")
			}
			if conn.WriteMessage(websocket.PingMessage, []byte{}) != nil {
				// Write error
				return
			}
			slog.Debug().Msg("Sent ping")
		}
	}
}

// Close removes the listener registration
func (ml *msgListenerV1) Close() {
	select {
	case <-ml.c:
		// Already closed
	default:
		ml.hub.RemoveListener(ml)
		close(ml.c)
	}
}

// MonitorAllMessagesV1 is a web handler which upgrades the connection to a websocket and notifies
// the client of all messages received.
func MonitorAllMessagesV1(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	// Upgrade to Websocket.
	conn, err := upgraderV1.Upgrade(w, req, nil)
	if err != nil {
		return err
	}
	web.ExpWebSocketConnectsCurrent.Add(1)
	defer func() {
		_ = conn.Close()
		web.ExpWebSocketConnectsCurrent.Add(-1)
	}()
	log.Debug().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Msg("Upgraded to WebSocket")
	// Create, register listener; then interact with conn.
	ml := newMsgListenerV1(ctx.MsgHub, "")
	go ml.WSWriter(conn)
	ml.WSReader(conn)
	return nil
}

// MonitorMailboxMessagesV1 is a web handler which upgrades the connection to a websocket and
// notifies the client of messages received by a particular mailbox.
func MonitorMailboxMessagesV1(
	w http.ResponseWriter, req *http.Request, ctx *web.Context) (err error) {
	name, err := ctx.Manager.MailboxForAddress(ctx.Vars["name"])
	if err != nil {
		return err
	}
	// Upgrade to Websocket.
	conn, err := upgraderV1.Upgrade(w, req, nil)
	if err != nil {
		return err
	}
	web.ExpWebSocketConnectsCurrent.Add(1)
	defer func() {
		_ = conn.Close()
		web.ExpWebSocketConnectsCurrent.Add(-1)
	}()
	log.Debug().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Msg("Upgraded to WebSocket")
	// Create, register listener; then interact with conn.
	ml := newMsgListenerV1(ctx.MsgHub, name)
	go ml.WSWriter(conn)
	ml.WSReader(conn)
	return nil
}

func metadataToHeader(msg *event.MessageMetadata) *model.JSONMessageHeaderV1 {
	return &model.JSONMessageHeaderV1{
		Mailbox:     msg.Mailbox,
		ID:          msg.ID,
		From:        stringutil.StringAddress(msg.From),
		To:          stringutil.StringAddressList(msg.To),
		Subject:     msg.Subject,
		Date:        msg.Date,
		PosixMillis: msg.Date.UnixNano() / 1000000,
		Size:        msg.Size,
	}
}
