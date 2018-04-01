package client

import "testing"

func TestClientV1ListMailbox(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{}
	c.client = mth

	// Method under test
	_, _ = c.ListMailbox("testbox")

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestClientV1GetMessage(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{}
	c.client = mth

	// Method under test
	_, _ = c.GetMessage("testbox", "20170107T224128-0000")

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox/20170107T224128-0000"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestClientV1MarkSeen(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{}
	c.client = mth

	// Method under test
	_ = c.MarkSeen("testbox", "20170107T224128-0000")

	want = "PATCH"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox/20170107T224128-0000"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestClientV1GetMessageSource(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{
		body: "message source",
	}
	c.client = mth

	// Method under test
	source, err := c.GetMessageSource("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox/20170107T224128-0000/source"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	want = "message source"
	got = source.String()
	if got != want {
		t.Errorf("Source == %q, want: %q", got, want)
	}
}

func TestClientV1DeleteMessage(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{}
	c.client = mth

	// Method under test
	err = c.DeleteMessage("testbox", "20170107T224128-0000")
	if err != nil {
		t.Fatal(err)
	}

	want = "DELETE"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox/20170107T224128-0000"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestClientV1PurgeMailbox(t *testing.T) {
	var want, got string

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{}
	c.client = mth

	// Method under test
	err = c.PurgeMailbox("testbox")
	if err != nil {
		t.Fatal(err)
	}

	want = "DELETE"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}

func TestClientV1MessageHeader(t *testing.T) {
	var want, got string
	response := `[
		{
			"mailbox":"mailbox1",
			"id":"id1",
			"from":"from1",
			"subject":"subject1",
			"date":"2017-01-01T00:00:00.000-07:00",
			"size":100,
			"seen":true
		}
	]`

	c, err := New(baseURLStr)
	if err != nil {
		t.Fatal(err)
	}
	mth := &mockHTTPClient{body: response}
	c.client = mth

	// Method under test
	headers, err := c.ListMailbox("testbox")
	if err != nil {
		t.Fatal(err)
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/testbox"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	if len(headers) != 1 {
		t.Fatalf("len(headers) == %v, want 1", len(headers))
	}
	header := headers[0]

	want = "mailbox1"
	got = header.Mailbox
	if got != want {
		t.Errorf("Mailbox == %q, want %q", got, want)
	}

	want = "id1"
	got = header.ID
	if got != want {
		t.Errorf("ID == %q, want %q", got, want)
	}

	want = "from1"
	got = header.From
	if got != want {
		t.Errorf("From == %q, want %q", got, want)
	}

	want = "subject1"
	got = header.Subject
	if got != want {
		t.Errorf("Subject == %q, want %q", got, want)
	}

	wantb := true
	gotb := header.Seen
	if gotb != wantb {
		t.Errorf("Seen == %v, want %v", gotb, wantb)
	}

	// Test MessageHeader.Delete()
	mth.body = ""
	err = header.Delete()
	if err != nil {
		t.Fatal(err)
	}

	want = "DELETE"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/mailbox1/id1"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	// Test MessageHeader.GetSource()
	mth.body = "source1"
	_, err = header.GetSource()
	if err != nil {
		t.Fatal(err)
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/mailbox1/id1/source"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	// Test MessageHeader.GetMessage()
	mth.body = `{
		"mailbox":"mailbox1",
		"id":"id1",
		"from":"from1",
		"subject":"subject1",
		"date":"2017-01-01T00:00:00.000-07:00",
		"size":100
	}`
	message, err := header.GetMessage()
	if err != nil {
		t.Fatal(err)
	}
	if message == nil {
		t.Fatalf("message was nil, wanted a value")
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/mailbox1/id1"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	// Test Message.Delete()
	mth.body = ""
	err = message.Delete()
	if err != nil {
		t.Fatal(err)
	}

	want = "DELETE"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/mailbox1/id1"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}

	// Test MessageHeader.GetSource()
	mth.body = "source1"
	_, err = message.GetSource()
	if err != nil {
		t.Fatal(err)
	}

	want = "GET"
	got = mth.req.Method
	if got != want {
		t.Errorf("req.Method == %q, want %q", got, want)
	}

	want = baseURLStr + "/api/v1/mailbox/mailbox1/id1/source"
	got = mth.req.URL.String()
	if got != want {
		t.Errorf("req.URL == %q, want %q", got, want)
	}
}
