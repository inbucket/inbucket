package client_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/inbucket/inbucket/v3/pkg/rest/client"
)

func TestClientV1ListMailbox(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	listHandler := &jsonHandler{json: `[
		{
			"mailbox": "testbox",
			"id": "1",
			"from": "fromuser",
			"subject": "test subject",
			"date": "2013-10-15T16:12:02.231532239-07:00",
			"size": 264,
			"seen": true
		}
	]`}

	router.Path("/api/v1/mailbox/testbox").Methods("GET").Handler(listHandler)

	// Method under test.
	headers, err := c.ListMailbox("testbox")
	if err != nil {
		t.Fatal(err)
	}

	if len(headers) != 1 {
		t.Fatalf("Got %v headers, want 1", len(headers))
	}
	h := headers[0]

	got := h.Mailbox
	want := "testbox"
	if got != want {
		t.Errorf("Mailbox got %q, want %q", got, want)
	}

	got = h.ID
	want = "1"
	if got != want {
		t.Errorf("ID got %q, want %q", got, want)
	}

	got = h.From
	want = "fromuser"
	if got != want {
		t.Errorf("From got %q, want %q", got, want)
	}

	got = h.Subject
	want = "test subject"
	if got != want {
		t.Errorf("Subject got %q, want %q", got, want)
	}

	gotTime := h.Date
	wantTime := time.Date(2013, 10, 15, 16, 12, 02, 231532239, time.FixedZone("UTC-7", -7*60*60))
	if !wantTime.Equal(gotTime) {
		t.Errorf("Date got %v, want %v", gotTime, wantTime)
	}

	gotInt := h.Size
	wantInt := int64(264)
	if gotInt != wantInt {
		t.Errorf("Size got %v, want %v", gotInt, wantInt)
	}

	wantBool := true
	gotBool := h.Seen
	if gotBool != wantBool {
		t.Errorf("Seen got %v, want %v", gotBool, wantBool)
	}
}

func TestClientV1GetMessage(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	messageHandler := &jsonHandler{json: `{
		"mailbox": "testbox",
		"id": "20170107T224128-0000",
		"from": "fromuser",
		"subject": "test subject",
		"date": "2013-10-15T16:12:02.231532239-07:00",
		"size": 264,
		"seen": true,
		"body": {
			"text": "Plain text",
			"html": "<html>"
		}
	}`}

	router.Path("/api/v1/mailbox/testbox/20170107T224128-0000").Methods("GET").Handler(messageHandler)

	// Method under test.
	m, err := c.GetMessage("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatalf("message was nil, wanted a value")
	}

	got := m.Mailbox
	want := "testbox"
	if got != want {
		t.Errorf("Mailbox got %q, want %q", got, want)
	}

	got = m.ID
	want = "20170107T224128-0000"
	if got != want {
		t.Errorf("ID got %q, want %q", got, want)
	}

	got = m.From
	want = "fromuser"
	if got != want {
		t.Errorf("From got %q, want %q", got, want)
	}

	got = m.Subject
	want = "test subject"
	if got != want {
		t.Errorf("Subject got %q, want %q", got, want)
	}

	gotTime := m.Date
	wantTime := time.Date(2013, 10, 15, 16, 12, 02, 231532239, time.FixedZone("UTC-7", -7*60*60))
	if !wantTime.Equal(gotTime) {
		t.Errorf("Date got %v, want %v", gotTime, wantTime)
	}

	gotInt := m.Size
	wantInt := int64(264)
	if gotInt != wantInt {
		t.Errorf("Size got %v, want %v", gotInt, wantInt)
	}

	gotBool := m.Seen
	wantBool := true
	if gotBool != wantBool {
		t.Errorf("Seen got %v, want %v", gotBool, wantBool)
	}

	got = m.Body.Text
	want = "Plain text"
	if got != want {
		t.Errorf("Body Text got %q, want %q", got, want)
	}

	got = m.Body.HTML
	want = "<html>"
	if got != want {
		t.Errorf("Body HTML got %q, want %q", got, want)
	}
}

func TestClientV1MarkSeen(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	handler := &jsonHandler{}
	router.Path("/api/v1/mailbox/testbox/20170107T224128-0000").Methods("PATCH").
		Handler(handler)

	// Method under test.
	err := c.MarkSeen("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	if !handler.called {
		t.Error("Wanted HTTP handler to be called, but it was not")
	}
}

func TestClientV1GetMessageSource(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	router.Path("/api/v1/mailbox/testbox/20170107T224128-0000/source").Methods("GET").
		Handler(&jsonHandler{json: `message source`})

	// Method under test.
	source, err := c.GetMessageSource("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	want := "message source"
	got := source.String()
	if got != want {
		t.Errorf("Source got %q, want %q", got, want)
	}
}

func TestClientV1WithCustomTransport(t *testing.T) {
	// Call setup, passing a custom roundtripper and make sure it was used during the request.
	mockRoundTripper := &mockRoundTripper{ResponseBody: "Custom Transport"}
	c, router, teardown := setup(client.WithTransport(mockRoundTripper))

	defer teardown()

	router.Path("/api/v1/mailbox/testbox/20170107T224128-0000/source").Methods("GET").
		Handler(&jsonHandler{json: `message source`})

	// Method under test.
	source, err := c.GetMessageSource("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	want := mockRoundTripper.ResponseBody
	got := source.String()
	if got != want {
		t.Errorf("Source got %q, want %q", got, want)
	}

	if mockRoundTripper.CallCount != 1 {
		t.Errorf("RoundTripper called %v times, want 1", mockRoundTripper.CallCount)
	}
}

func TestClientV1DeleteMessage(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	handler := &jsonHandler{}
	router.Path("/api/v1/mailbox/testbox/20170107T224128-0000").Methods("DELETE").
		Handler(handler)

	// Method under test.
	err := c.DeleteMessage("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	if !handler.called {
		t.Error("Wanted HTTP handler to be called, but it was not")
	}
}

func TestClientV1PurgeMailbox(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	handler := &jsonHandler{}
	router.Path("/api/v1/mailbox/testbox").Methods("DELETE").Handler(handler)

	// Method under test.
	err := c.PurgeMailbox("testbox")
	if err != nil {
		t.Fatal(err)
	}

	if !handler.called {
		t.Error("Wanted HTTP handler to be called, but it was not")
	}
}

func TestClientV1MessageHeader(t *testing.T) {
	// Setup.
	c, router, teardown := setup()
	defer teardown()

	listHandler := &jsonHandler{json: `[
		{
			"mailbox":"mailbox1",
			"id":"id1",
			"from":"from1",
			"subject":"subject1",
			"date":"2017-01-01T00:00:00.000-07:00",
			"size":100,
			"seen":true
		}
	]`}
	router.Path("/api/v1/mailbox/testbox").Methods("GET").Handler(listHandler)

	// Method under test.
	headers, err := c.ListMailbox("testbox")
	if err != nil {
		t.Fatal(err)
	}

	if len(headers) != 1 {
		t.Fatalf("len(headers) == %v, want 1", len(headers))
	}
	header := headers[0]

	// Test MessageHeader.Delete().
	handler := &jsonHandler{}
	router.Path("/api/v1/mailbox/mailbox1/id1").Methods("DELETE").Handler(handler)
	err = header.Delete()
	if err != nil {
		t.Fatal(err)
	}

	// Test MessageHeader.GetSource().
	router.Path("/api/v1/mailbox/mailbox1/id1/source").Methods("GET").
		Handler(&jsonHandler{json: `source1`})
	buf, err := header.GetSource()
	if err != nil {
		t.Fatal(err)
	}

	want := "source1"
	got := buf.String()
	if got != want {
		t.Errorf("Got source %q, want %q", got, want)
	}

	// Test MessageHeader.GetMessage().
	messageHandler := &jsonHandler{json: `{
		"mailbox":"mailbox1",
		"id":"id1",
		"from":"from1",
		"subject":"subject1",
		"date":"2017-01-01T00:00:00.000-07:00",
		"size":100
	}`}
	router.Path("/api/v1/mailbox/mailbox1/id1").Methods("GET").Handler(messageHandler)
	message, err := header.GetMessage()
	if err != nil {
		t.Fatal(err)
	}
	if message == nil {
		t.Fatalf("message was nil, wanted a value")
	}

	// Test Message.Delete().
	err = message.Delete()
	if err != nil {
		t.Fatal(err)
	}

	// Test Message.GetSource().
	buf, err = message.GetSource()
	if err != nil {
		t.Fatal(err)
	}

	want = "source1"
	got = buf.String()
	if got != want {
		t.Errorf("Got source %q, want %q", got, want)
	}
}

type mockRoundTripper struct {
	ResponseBody string
	CallCount    int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.CallCount++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(m.ResponseBody)),
	}, nil
}

// setup returns a client, router and server for API testing.
func setup(opts ...client.Option) (c *client.Client, router *mux.Router, teardown func()) {
	router = mux.NewRouter()
	server := httptest.NewServer(router)
	c, err := client.New(server.URL, opts...)
	if err != nil {
		panic(err)
	}
	return c, router, func() {
		server.Close()
	}
}

// jsonHandler returns the string in json when servicing a request.
type jsonHandler struct {
	json   string
	called bool
}

func (j *jsonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	j.called = true
	_, _ = w.Write([]byte(j.json))
}
