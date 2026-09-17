package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// monitorTestTimeout bounds WebSocket reads so a broken server fails the test instead of hanging.
const monitorTestTimeout = 5 * time.Second

// monitorDialer dials test servers directly; the default dialer would honor proxy env vars.
var monitorDialer = &websocket.Dialer{}

// startMsgHub returns a running *msghub.Hub with room for every message a test dispatches
// (replayed to listeners that connect later).  The actor goroutine is stopped via t.Cleanup.
func startMsgHub(t *testing.T) *msghub.Hub {
	t.Helper()
	hub := msghub.New(10, extension.NewHost())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go hub.Start(ctx)
	return hub
}

// setupMonitorServer wires a real (running) msg hub into the web server globals and serves the
// global router over HTTP; upgrading WebSocket endpoints needs a real connection to hijack.
// The returned gauge baseline is the value of web.ExpWebSocketConnectsCurrent before any
// connection is made; pass it to dialMonitor so sessions can be awaited on teardown.
func setupMonitorServer(t *testing.T, hub *msghub.Hub) (*httptest.Server, int64) {
	t.Helper()
	mm := test.NewManager()
	setupWebServer(mm, hub)
	server := httptest.NewServer(web.Router)
	// Registered before any connection cleanup so the listener socket is closed after the
	// sessions below have been awaited.
	t.Cleanup(server.Close)
	return server, web.ExpWebSocketConnectsCurrent.Value()
}

// dialMonitor connects a WebSocket client to path on server.  The cleanup closes the connection
// and waits for the server-side handler goroutine to exit, which is required before the msg hub
// may be stopped: httptest.Server.Close does not wait for hijacked (WebSocket) connections, and
// a handler deregistering its hub listener after the hub was stopped races with it.
func dialMonitor(t *testing.T, server *httptest.Server, path string, gaugeBase int64) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + path
	conn, resp, err := monitorDialer.Dial(url, nil)
	require.NoError(t, err)
	// resp.Body is an empty no-op closer once the connection is upgraded.
	_ = resp.Body.Close()
	t.Cleanup(func() {
		_ = conn.Close()
		require.Eventually(t, func() bool {
			return web.ExpWebSocketConnectsCurrent.Value() == gaugeBase
		}, monitorTestTimeout, time.Millisecond, "server handler did not shut down")
	})
	return conn
}

// readMonitorJSON reads the next data message from conn and decodes it into a generic map.
func readMonitorJSON(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(monitorTestTimeout)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	var decoded interface{}
	require.NoError(t, json.Unmarshal(data, &decoded), "payload: %s", data)
	obj, ok := decoded.(map[string]interface{})
	require.True(t, ok, "expected JSON object, got: %s", data)
	return obj
}

// testMetadata builds a message for dispatch into the hub.
func testMetadata(mailbox, id string) event.MessageMetadata {
	return event.MessageMetadata{
		Mailbox: mailbox,
		ID:      id,
		From:    &mail.Address{Address: "from-" + id + "@host"},
		To:      []*mail.Address{{Address: "to-" + id + "@host"}},
		Subject: "subject " + id,
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 0, time.UTC),
		Size:    1234,
	}
}

// assertMonitorHeader checks the JSONMessageHeaderV1 fields of a decoded monitor payload.
func assertMonitorHeader(t *testing.T, decoded interface{}, want event.MessageMetadata) {
	t.Helper()
	decodedStringEquals(t, decoded, "mailbox", want.Mailbox)
	decodedStringEquals(t, decoded, "id", want.ID)
	decodedStringEquals(t, decoded, "from", "<"+want.From.Address+">")
	decodedStringEquals(t, decoded, "to/[0]", "<"+want.To[0].Address+">")
	decodedStringEquals(t, decoded, "subject", want.Subject)
	decodedStringEquals(t, decoded, "date", "2012-02-01T10:11:12Z")
	decodedNumberEquals(t, decoded, "posix-millis", float64(want.Date.UnixNano()/1000000))
	decodedNumberEquals(t, decoded, "size", float64(want.Size))
}

// monitorEventHeader returns the nested header object of a decoded v2 monitor event.
func monitorEventHeader(t *testing.T, ev map[string]interface{}) map[string]interface{} {
	t.Helper()
	header, ok := ev["header"].(map[string]interface{})
	require.True(t, ok, "event has no header object: %v", ev)
	return header
}

func TestMonitorAllMessagesV1(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	// Messages dispatched before connecting are replayed from hub history to new listeners.
	replayed := testMetadata("replay", "0001")
	hub.Dispatch(replayed)

	conn := dialMonitor(t, server, "/api/v1/monitor/messages", gaugeBase)
	assertMonitorHeader(t, readMonitorJSON(t, conn), replayed)
	assert.Equal(t, gaugeBase+1, web.ExpWebSocketConnectsCurrent.Value(),
		"open WebSocket connections gauge")

	// Messages dispatched after registration are delivered live.  Receiving the replayed
	// message above proves the listener was registered before this dispatch.
	live := testMetadata("live", "0002")
	hub.Dispatch(live)
	assertMonitorHeader(t, readMonitorJSON(t, conn), live)

	// A clean client close tears the session down server-side: the read pump sees the close
	// frame, the listener deregisters from the hub, and the connections gauge drops back.
	require.NoError(t, conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(monitorTestTimeout)))
	require.NoError(t, conn.Close())
	require.Eventually(t, func() bool {
		return web.ExpWebSocketConnectsCurrent.Value() == gaugeBase
	}, monitorTestTimeout, time.Millisecond, "open WebSocket connections gauge")
}

func TestMonitorMailboxMessagesV1(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	hub.Dispatch(testMetadata("other", "0001"))
	watched := testMetadata("watched", "0002")
	hub.Dispatch(watched)
	hub.Dispatch(testMetadata("other", "0003"))

	conn := dialMonitor(t, server, "/api/v1/monitor/messages/watched", gaugeBase)
	assertMonitorHeader(t, readMonitorJSON(t, conn), watched)

	// Events are relayed in dispatch order, so receiving 0005 next proves both the replayed
	// and live deliveries for other mailboxes were filtered, not merely delayed.
	hub.Dispatch(testMetadata("other", "0004"))
	live := testMetadata("watched", "0005")
	hub.Dispatch(live)
	assertMonitorHeader(t, readMonitorJSON(t, conn), live)

	// Documents current behavior: v1 monitors ignore message deletions and emit no event;
	// v2 emits message-deleted.  The subsequent dispatch arriving next proves the delete
	// produced nothing.  Flagged for a follow-up fix.
	hub.Delete("watched", "0005")
	after := testMetadata("watched", "0006")
	hub.Dispatch(after)
	assertMonitorHeader(t, readMonitorJSON(t, conn), after)
}

func TestMonitorMailboxAddressV1(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	// The {name} path segment goes through Manager.MailboxForAddress; with the test stub's
	// full-address naming policy, user@example.com maps to a mailbox of the same name.
	stored := testMetadata("user@example.com", "0001")
	hub.Dispatch(stored)

	conn := dialMonitor(t, server, "/api/v1/monitor/messages/user%40example.com", gaugeBase)
	assertMonitorHeader(t, readMonitorJSON(t, conn), stored)
}

func TestMonitorMailboxInvalidNameV1(t *testing.T) {
	hub := startMsgHub(t)
	setupWebServer(test.NewManager(), hub)

	// Invalid mailbox names are rejected before the connection is upgraded.
	w, err := testRestGet("http://localhost/api/v1/monitor/messages/foo%20bar")
	require.NoError(t, err)
	assert.Equal(t, 500, w.Code)
}

func TestMonitorUpgradeFailedV1(t *testing.T) {
	hub := startMsgHub(t)
	setupWebServer(test.NewManager(), hub)

	// A plain GET without WebSocket upgrade headers fails the handshake; the upgrader writes
	// the 400 itself before the handler error is reported.
	w, err := testRestGet("http://localhost/api/v1/monitor/messages")
	require.NoError(t, err)
	assert.Equal(t, 400, w.Code)
}

func TestMonitorClientCloseV1(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	stored := testMetadata("close", "0001")
	hub.Dispatch(stored)

	conn := dialMonitor(t, server, "/api/v1/monitor/messages", gaugeBase)

	// Data messages sent by the client are discarded; the monitor keeps relaying events.
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"junk":true}`)))
	assertMonitorHeader(t, readMonitorJSON(t, conn), stored)

	// A clean client close ends the session.  The peer may observe any of: the close-code
	// echo sent by the server read pump's default close handler, the writer's empty close
	// frame, or a dropped TCP connection from the handler teardown; all indicate shutdown.
	require.NoError(t, conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(monitorTestTimeout)))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(monitorTestTimeout)))
	_, _, err := conn.ReadMessage()
	require.Error(t, err)
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		assert.Contains(t, []int{
			websocket.CloseNormalClosure,    // echoed by the server read pump
			websocket.CloseNoStatusReceived, // writer's empty close frame
		}, closeErr.Code)
	}
}

func TestMonitorAllMessagesV2(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	// Stored messages dispatched before connecting are replayed from hub history.
	stored := testMetadata("replay", "0001")
	hub.Dispatch(stored)

	conn := dialMonitor(t, server, "/api/v2/monitor/messages", gaugeBase)
	ev := readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-stored")
	assertMonitorHeader(t, monitorEventHeader(t, ev), stored)

	// Data messages sent by the client are discarded; the monitor keeps relaying events.
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"junk":true}`)))

	live := testMetadata("live", "0002")
	hub.Dispatch(live)
	ev = readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-stored")
	assertMonitorHeader(t, monitorEventHeader(t, ev), live)

	// Deletions are reported as message-deleted events with the message identifier.
	hub.Delete("live", "0002")
	ev = readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-deleted")
	decodedStringEquals(t, ev, "identifier/mailbox", "live")
	decodedStringEquals(t, ev, "identifier/id", "0002")
}

func TestMonitorMailboxMessagesV2(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	hub.Dispatch(testMetadata("other", "0001"))
	watched := testMetadata("watched", "0002")
	hub.Dispatch(watched)

	conn := dialMonitor(t, server, "/api/v2/monitor/messages/watched", gaugeBase)
	ev := readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-stored")
	assertMonitorHeader(t, monitorEventHeader(t, ev), watched)

	// Events are relayed in dispatch order, so receiving message-stored for 0003 next proves
	// the other-mailbox delete was filtered, not merely delayed.
	hub.Delete("other", "0001")
	live := testMetadata("watched", "0003")
	hub.Dispatch(live)
	ev = readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-stored")
	assertMonitorHeader(t, monitorEventHeader(t, ev), live)

	hub.Delete("other", "9999")
	hub.Delete("watched", "0003")
	ev = readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-deleted")
	decodedStringEquals(t, ev, "identifier/mailbox", "watched")
	decodedStringEquals(t, ev, "identifier/id", "0003")
}

func TestMonitorMailboxAddressV2(t *testing.T) {
	hub := startMsgHub(t)
	server, gaugeBase := setupMonitorServer(t, hub)

	// The {name} path segment goes through Manager.MailboxForAddress; with the test stub's
	// full-address naming policy, user@example.com maps to a mailbox of the same name.
	stored := testMetadata("user@example.com", "0001")
	hub.Dispatch(stored)

	conn := dialMonitor(t, server, "/api/v2/monitor/messages/user%40example.com", gaugeBase)
	ev := readMonitorJSON(t, conn)
	decodedStringEquals(t, ev, "variant", "message-stored")
	assertMonitorHeader(t, monitorEventHeader(t, ev), stored)
}

func TestMonitorMailboxInvalidNameV2(t *testing.T) {
	hub := startMsgHub(t)
	setupWebServer(test.NewManager(), hub)

	// Invalid mailbox names are rejected before the connection is upgraded.
	w, err := testRestGet("http://localhost/api/v2/monitor/messages/foo%20bar")
	require.NoError(t, err)
	assert.Equal(t, 500, w.Code)
}

func TestMonitorUpgradeFailedV2(t *testing.T) {
	hub := startMsgHub(t)
	setupWebServer(test.NewManager(), hub)

	// A plain GET without WebSocket upgrade headers fails the handshake; the upgrader writes
	// the 400 itself before the handler error is reported.
	w, err := testRestGet("http://localhost/api/v2/monitor/messages")
	require.NoError(t, err)
	assert.Equal(t, 400, w.Code)
}

// assertNoMoreEvents fails the test if any further events are queued on the channel.
func assertNoMoreEvents[T any](t *testing.T, c chan T) {
	t.Helper()
	select {
	case ev := <-c:
		t.Fatalf("unexpected event: %v", ev)
	default:
	}
}

func TestMsgListenerWatches(t *testing.T) {
	testCases := []struct {
		name    string
		mailbox string // listener watch; "" watches all mailboxes
		event   string // mailbox of the incoming event
		want    bool
	}{
		{name: "all watcher matches any mailbox", mailbox: "", event: "anything", want: true},
		{name: "watched mailbox matches", mailbox: "watched", event: "watched", want: true},
		{name: "other mailbox filtered", mailbox: "watched", event: "other", want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ml := &msgListener[string]{mailbox: tc.mailbox}
			assert.Equal(t, tc.want, ml.watches(tc.event))
		})
	}
}

func TestMsgListenerReceiveFiltersAndShapes(t *testing.T) {
	hub := startMsgHub(t)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.Mailbox + "/" + msg.ID },
		func(mailbox, id string) string { return "deleted:" + mailbox + "/" + id })

	require.NoError(t, ml.Receive(testMetadata("other", "0001")))
	require.NoError(t, ml.Receive(testMetadata("watched", "0002")))
	require.NoError(t, ml.Delete("other", "0003"))
	require.NoError(t, ml.Delete("watched", "0004"))

	// Events for other mailboxes are filtered; payloads are shaped by the transforms.
	assert.Equal(t, "stored:watched/0002", <-ml.c)
	assert.Equal(t, "deleted:watched/0004", <-ml.c)
	assertNoMoreEvents(t, ml.c)
}

func TestMsgListenerNilDeletedDropsDeletes(t *testing.T) {
	hub := startMsgHub(t)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.ID },
		nil) // socketv1 API ignores deletions

	require.NoError(t, ml.Receive(testMetadata("watched", "0001")))
	require.NoError(t, ml.Delete("watched", "0001"))

	assert.Equal(t, "stored:0001", <-ml.c)
	assertNoMoreEvents(t, ml.c)
}

func TestMsgListenerCloseIdempotent(t *testing.T) {
	hub := startMsgHub(t)
	ml := newMsgListener(hub, "",
		func(msg event.MessageMetadata) string { return "stored" }, nil)

	ml.Close()
	ml.Close() // second close must not double-close the event channel

	_, ok := <-ml.c
	assert.False(t, ok, "event channel should be closed")
}
