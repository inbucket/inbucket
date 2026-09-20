package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/mail"
	"strconv"
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
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored" },
		func(mailbox, id string) string { return "deleted" })

	ml.Close()
	ml.Close() // second close must not panic or double-close the done channel

	// Close is exactly-once: the done channel ends delivery and the write pump.
	select {
	case <-ml.done:
	default:
		t.Fatal("done channel should be closed after Close")
	}

	// Closed listeners either deliver into the never-again-read queue or report closed;
	// both outcomes are terminal, as the hub-actor send races the done channel.
	err := ml.Receive(testMetadata("watched", "0001"))
	if err != nil {
		require.ErrorIs(t, err, errListenerClosed)
	}
	err = ml.Delete("watched", "0001")
	if err != nil {
		require.ErrorIs(t, err, errListenerClosed)
	}
}

func TestMsgListenerCloseFullQueue(t *testing.T) {
	// A disconnecting client may leave a full event queue behind; Close must still
	// deregister the listener, or the hub actor blocks on its next Receive into the full
	// queue forever, wedging all monitor dispatch.  Regression: the old check-then-act
	// select mistook the buffered events for a closed-channel signal and skipped
	// RemoveListener entirely.
	hub := startMsgHub(t)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.ID }, nil)

	// Fill the event queue (capacity 100) with no reader draining it.
	for i := range 100 {
		hub.Dispatch(testMetadata("watched", strconv.Itoa(i)))
	}
	hub.Sync()

	ml.Close()
	hub.Sync()

	// The backlog remains queued, but the listener is deregistered: dispatch must not
	// block the hub, and must not be delivered.  Pre-fix, the no-op Close left the
	// listener registered, so the dispatch was delivered (and the next dispatch would
	// have wedged the actor inside Receive).
	backlog, _ := drainEvents(ml.c)
	assert.Equal(t, 100, backlog, "backlog should remain queued after Close")

	hub.Dispatch(testMetadata("watched", "after-close"))
	hub.Sync()

	count, _ := drainEvents(ml.c)
	assert.Equal(t, 0, count, "closed listener must not receive further events")
}

// drainEvents reads buffered events from c until it is empty; it reports how many events
// were drained and whether the channel was found closed.  A receive from a closed channel
// succeeds immediately, so a plain select-default loop would never terminate on one.
func drainEvents[T any](c chan T) (count int, closed bool) {
	for {
		select {
		case _, ok := <-c:
			if !ok {
				return count, true
			}
			count++
		default:
			return count, false
		}
	}
}

// blockingListener stalls the hub actor inside Receive until released.
type blockingListener struct {
	release chan struct{}
}

func (l *blockingListener) Receive(msg event.MessageMetadata) error {
	<-l.release
	return nil
}

func (l *blockingListener) Delete(mailbox string, id string) error {
	return nil
}

func TestMsgListenerCloseConcurrent(t *testing.T) {
	// The read and write pumps both call Close when the connection drops, so Close must be
	// safe to call concurrently.  Regression: the check-then-act select let both callers
	// through the default branch and panicked with "close of closed channel" on the second
	// close.
	//
	// Deterministic reproduction: a blocking listener stalls the hub actor mid-dispatch,
	// and enough further dispatches are queued to keep the hub op queue (capacity 100)
	// full.  Every concurrent Close then blocks inside RemoveListener -- already past its
	// check of the listener state -- before any of them can proceed.  Releasing the actor
	// lets it process the pending RemoveListeners, resuming all the blocked closers at
	// once.
	hub := startMsgHub(t)
	blocker := &blockingListener{release: make(chan struct{})}
	hub.AddListener(blocker)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.ID },
		func(mailbox, id string) string { return "deleted:" + id })

	const queued = 101 // 1 dispatch stalls the actor; 100 fill the hub op queue
	for i := range queued {
		hub.Dispatch(testMetadata("watched", strconv.Itoa(i)))
	}

	// The stalled dispatch may have delivered 0 or 1 events to ml (listener iteration
	// order is random); drain whatever arrived so the queue is empty when the closers
	// check it.
	drainEvents(ml.c)

	const callers = 8
	done := make(chan struct{}, callers) // ensure capacity so we do not block population
	for range callers {
		go func() {
			ml.Close()
			done <- struct{}{}
		}()
	}
	// Give every closer time to block inside RemoveListener before the actor resumes;
	// otherwise their ops interleave with the queued dispatches and the reproduction
	// becomes timing-dependent.
	time.Sleep(50 * time.Millisecond)

	close(blocker.release)
	for range callers {
		<-done
	}

	// Events racing in before the hub processed the RemoveListeners may sit in the queue,
	// but once it has (Sync), the listener must be deregistered: later dispatches are
	// neither delivered to it nor blocking the hub.
	hub.Sync()
	drainEvents(ml.c)
	hub.Dispatch(testMetadata("watched", "after-close"))
	hub.Sync()
	count, _ := drainEvents(ml.c)
	assert.Equal(t, 0, count, "closed listener must not receive further events")
}

func TestMsgListenerCloseUnblocksHub(t *testing.T) {
	// Close must unblock a hub actor stuck sending into the listener's full event queue:
	// a slow monitor client disconnecting with a backlog of events would otherwise wedge
	// every hub dispatch.  (With the original close(ml.c) approach, this
	// send-on-closed-channel panicked inside the actor instead.)
	hub := startMsgHub(t)
	blocker := &blockingListener{release: make(chan struct{})}
	hub.AddListener(blocker)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.ID }, nil)

	// Stall the actor, then queue enough dispatches to fill both the hub op queue and the
	// listener's event queue once deliveries resume.
	const queued = 101 // 1 dispatch stalls the actor; 100 fill the hub op queue
	for i := range queued {
		hub.Dispatch(testMetadata("watched", strconv.Itoa(i)))
	}
	drainEvents(ml.c) // drop the 0 or 1 events delivered before the actor stalled
	close(blocker.release)

	// Once the queued dispatches fill the event queue, the next dispatch blocks the actor
	// inside Receive; run it from its own goroutine so the test can proceed to Close.
	dispatched := make(chan struct{}, 1)
	go func() {
		hub.Dispatch(testMetadata("watched", "one-too-many"))
		dispatched <- struct{}{}
	}()

	ml.Close()

	// The blocked send must abort and the hub must keep processing.
	synced := make(chan struct{}, 1)
	go func() {
		hub.Sync()
		synced <- struct{}{}
	}()
	select {
	case <-synced:
	case <-time.After(5 * time.Second):
		t.Fatal("hub actor still wedged after Close")
	}
	select {
	case <-dispatched:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch still blocked after Close")
	}
}

func TestMsgListenerCloseFullHubOpQueue(t *testing.T) {
	// Regression (PR #623 review): with the hub actor blocked inside this listener's
	// Receive (full event queue) and the hub op queue (capacity 100) also full,
	// RemoveListener cannot enqueue.  Close must therefore close done before queueing the
	// removal: with the removal queued first, execution never reached close(ml.done) -- the
	// only operation able to release the actor -- and Close deadlocked under the same
	// slow-client condition while concurrent dispatchers kept the hub queue saturated.
	hub := startMsgHub(t)
	blocker := &blockingListener{release: make(chan struct{})}
	hub.AddListener(blocker)
	ml := newMsgListener(hub, "watched",
		func(msg event.MessageMetadata) string { return "stored:" + msg.ID }, nil)

	// Stall the actor inside the blocker, then fill the hub op queue behind it: the first
	// dispatch wedges the actor, the next 100 fill the queue.  The stalled dispatch may
	// have delivered 0 or 1 events to ml (listener iteration order is random); drain
	// whatever arrived so the counts below are stable.
	for i := range 101 {
		hub.Dispatch(testMetadata("watched", strconv.Itoa(i)))
	}
	drainEvents(ml.c)

	// Once the blocker is released, a dispatcher goroutine keeps the op queue saturated:
	// every slot the actor frees is refilled, so by the time the event queue (capacity 100)
	// fills and the actor wedges inside Receive, the op queue is full again and the
	// dispatcher is blocked on its next enqueue.  150 refills exceed the ~102 slots freed
	// until the wedge, leaving spare dispatches to top the queue up.
	dispatched := make(chan struct{})
	go func() {
		defer close(dispatched)
		for i := range 150 {
			hub.Dispatch(testMetadata("watched", strconv.Itoa(100+i)))
		}
	}()
	close(blocker.release)

	// Wait for the wedge: the event queue is full and nothing drains it.  The sleep lets
	// the dispatcher refill the one slot the wedging op vacated and block itself.
	require.Eventually(t, func() bool {
		return len(ml.c) == socketChanLen
	}, monitorTestTimeout, time.Millisecond, "event queue should fill and wedge the hub actor")
	time.Sleep(50 * time.Millisecond)

	// Close must return despite the wedged actor and the saturated op queue.
	closed := make(chan struct{})
	go func() {
		ml.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close deadlocked: hub actor blocked in Receive with a full op queue")
	}

	// The released actor must drain the saturated op queue: the dispatcher completes and
	// the hub answers Sync again.
	select {
	case <-dispatched:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatches still blocked after Close")
	}
	synced := make(chan struct{}, 1)
	go func() {
		hub.Sync()
		synced <- struct{}{}
	}()
	select {
	case <-synced:
	case <-time.After(5 * time.Second):
		t.Fatal("hub actor still wedged after Close")
	}

	// The listener is deregistered: later dispatches are not delivered to it.
	drainEvents(ml.c)
	hub.Dispatch(testMetadata("watched", "after-close"))
	hub.Sync()
	count, _ := drainEvents(ml.c)
	assert.Equal(t, 0, count, "closed listener must not receive further events")
}
