package rest

import (
	"errors"
	"net/http"
	"sync"
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
	// socketWriteWait is the time allowed to write a message to the peer.
	socketWriteWait = 10 * time.Second

	// socketPingPeriod is how often pings are sent to the peer. Must be less than
	// socketPongWait.
	socketPingPeriod = (socketPongWait * 9) / 10

	// socketPongWait is the time allowed to read the next pong message from the peer.
	socketPongWait = 60 * time.Second

	// socketMaxMessageSize is the maximum message size allowed from the peer.
	socketMaxMessageSize = 512

	// socketChanLen is the event queue length of a monitor listener.
	socketChanLen = 100
)

// upgrader holds the options for monitor WebSocket connection upgrades.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// msgListener handles messages from the msghub for a single monitor connection.  Stored and
// deleted events are shaped into the API-specific payload by toStored/toDeleted; a nil
// toDeleted drops deletions (socketv1 API behavior).
type msgListener[T any] struct {
	hub       *msghub.Hub
	c         chan T
	done      chan struct{} // Closed by Close; stops event delivery and the write pump
	mailbox   string        // Name of mailbox to monitor, "" == all mailboxes
	toStored  func(msg event.MessageMetadata) T
	toDeleted func(mailbox string, id string) T
	closeOnce sync.Once
}

// newMsgListener creates a listener and registers it with the hub.  The optional mailbox
// parameter restricts events relayed to the client to that mailbox only.
func newMsgListener[T any](
	hub *msghub.Hub,
	mailbox string,
	toStored func(msg event.MessageMetadata) T,
	toDeleted func(mailbox string, id string) T,
) *msgListener[T] {
	ml := &msgListener[T]{
		hub:       hub,
		c:         make(chan T, socketChanLen),
		done:      make(chan struct{}),
		mailbox:   mailbox,
		toStored:  toStored,
		toDeleted: toDeleted,
	}
	hub.AddListener(ml)
	return ml
}

// watches reports whether events for the given mailbox should be relayed to the client.
func (ml *msgListener[T]) watches(mailbox string) bool {
	return ml.mailbox == "" || ml.mailbox == mailbox
}

// errListenerClosed is returned by Receive/Delete after Close; the hub removes listeners
// that return errors, providing deregistration even if RemoveListener has not run yet.
var errListenerClosed = errors.New("monitor listener closed")

// Receive handles an incoming message.  If the listener is closed while its event queue is
// full, the enqueue is abandoned rather than blocking the hub actor forever.
func (ml *msgListener[T]) Receive(msg event.MessageMetadata) error {
	if !ml.watches(msg.Mailbox) {
		return nil
	}
	select {
	case ml.c <- ml.toStored(msg):
		return nil
	case <-ml.done:
		return errListenerClosed
	}
}

// Delete handles a deleted message.
func (ml *msgListener[T]) Delete(mailbox string, id string) error {
	if ml.toDeleted == nil || !ml.watches(mailbox) {
		return nil
	}
	select {
	case ml.c <- ml.toDeleted(mailbox, id):
		return nil
	case <-ml.done:
		return errListenerClosed
	}
}

// Close removes the listener registration and stops event delivery.  Safe for concurrent
// use: the read and write pumps both invoke it when the connection drops.  The event queue
// is not closed: RemoveListener returns once the op is queued, not processed, so a
// concurrent hub-actor send into the queue could otherwise panic on the close; the done
// channel ends delivery and unblocks any such send.
func (ml *msgListener[T]) Close() {
	ml.closeOnce.Do(func() {
		ml.hub.RemoveListener(ml)
		close(ml.done)
	})
}

// socketReadPump makes sure the websocket client is still connected, discarding any messages
// the client sends.  It returns once the connection fails or the peer closes it; closeFn is
// then invoked to tear the listener down.
func socketReadPump(conn *websocket.Conn, closeFn func()) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()

	defer closeFn()

	conn.SetReadLimit(socketMaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(socketPongWait)); err != nil {
		slog.Warn().Err(err).Msg("Failed to setup read deadline")
	}
	conn.SetPongHandler(func(string) error {
		slog.Debug().Msg("Got pong")
		if err := conn.SetReadDeadline(time.Now().Add(socketPongWait)); err != nil {
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

// socketWritePump relays events from c to the peer as JSON messages until done is closed,
// sending pings while idle to keep the connection alive.  closeFn is invoked on exit.
func socketWritePump[T any](c chan T, done <-chan struct{}, conn *websocket.Conn, closeFn func()) {
	slog := log.With().Str("module", "rest").Str("proto", "WebSocket").
		Str("remote", conn.RemoteAddr().String()).Logger()

	ticker := time.NewTicker(socketPingPeriod)
	defer func() {
		ticker.Stop()
		closeFn()
	}()

	// Handle messages from hub until listener is closed
	for {
		select {
		case event, ok := <-c:
			if err := conn.SetWriteDeadline(time.Now().Add(socketWriteWait)); err != nil {
				slog.Warn().Err(err).Msg("Failed to set write deadline for msg")
			}
			if !ok {
				// Listener closed, exit
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if conn.WriteJSON(event) != nil {
				// Write failed
				return
			}
		case <-done:
			// Listener closed, exit
			_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case <-ticker.C:
			// Send ping
			if err := conn.SetWriteDeadline(time.Now().Add(socketWriteWait)); err != nil {
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

// serveMonitor upgrades the connection to a WebSocket, registers a new listener with the hub,
// and relays events to the client until it disconnects.
func serveMonitor[T any](
	w http.ResponseWriter,
	req *http.Request,
	hub *msghub.Hub,
	mailbox string,
	newListener func(hub *msghub.Hub, mailbox string) *msgListener[T],
) error {
	// Upgrade to Websocket.
	conn, err := upgrader.Upgrade(w, req, nil)
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
	ml := newListener(hub, mailbox)
	go socketWritePump(ml.c, ml.done, conn, ml.Close)
	socketReadPump(conn, ml.Close)
	return nil
}

// metadataToHeader converts hub event metadata into a JSON message header.
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
