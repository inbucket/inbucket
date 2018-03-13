package rest

import (
	"encoding/json"
	"io"
	"net/mail"
	"net/textproto"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/test"
)

const (
	baseURL = "http://localhost/api/v1"

	// JSON map keys
	mailboxKey = "mailbox"
	idKey      = "id"
	fromKey    = "from"
	toKey      = "to"
	subjectKey = "subject"
	dateKey    = "date"
	sizeKey    = "size"
	headerKey  = "header"
	bodyKey    = "body"
	textKey    = "text"
	htmlKey    = "html"
)

func TestRestMailboxList(t *testing.T) {
	// Setup
	mm := test.NewManager()
	logbuf := setupWebServer(mm)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test empty mailbox
	w, err = testRestGet(baseURL + "/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test Mailbox error
	w, err = testRestGet(baseURL + "/mailbox/messageserr")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test JSON message headers
	data1 := &InputMessageData{
		Mailbox: "good",
		ID:      "0001",
		From:    "<from1@host>",
		To:      []string{"<to1@host>"},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	data2 := &InputMessageData{
		Mailbox: "good",
		ID:      "0002",
		From:    "<from2@host>",
		To:      []string{"<to1@host>"},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, time.FixedZone("PDT", -700)),
	}
	meta1 := message.Metadata{
		Mailbox: "good",
		ID:      "0001",
		From:    &mail.Address{Name: "", Address: "from1@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	meta2 := message.Metadata{
		Mailbox: "good",
		ID:      "0002",
		From:    &mail.Address{Name: "", Address: "from2@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, time.FixedZone("PDT", -700)),
	}
	mm.AddMessage("good", &message.Message{Metadata: meta1})
	mm.AddMessage("good", &message.Message{Metadata: meta2})

	// Check return code
	w, err = testRestGet(baseURL + "/mailbox/good")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	got := w.Body.String()
	testStrings := []string{
		`{"mailbox":"good","id":"0001","from":"\u003cfrom1@host\u003e",` +
			`"to":["\u003cto1@host\u003e"],"subject":"subject 1",` +
			`"date":"2012-02-01T10:11:12.000000253-00:13","size":0}`,
		`{"mailbox":"good","id":"0002","from":"\u003cfrom2@host\u003e",` +
			`"to":["\u003cto1@host\u003e"],"subject":"subject 2",` +
			`"date":"2012-07-01T10:11:12.000000253-00:11","size":0}`,
	}
	for _, ts := range testStrings {
		t.Run(ts, func(t *testing.T) {
			if !strings.Contains(got, ts) {
				t.Errorf("got:\n%s\nwant to contain:\n%s", got, ts)
			}
		})
	}

	// Check JSON
	// TODO transitional while refactoring
	dec := json.NewDecoder(w.Body)
	var result []interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %v", len(result))
	}
	if errors := data1.CompareToJSONHeaderMap(result[0]); len(errors) > 0 {
		t.Logf("%v", result[0])
		for _, e := range errors {
			t.Error(e)
		}
	}
	if errors := data2.CompareToJSONHeaderMap(result[1]); len(errors) > 0 {
		t.Logf("%v", result[1])
		for _, e := range errors {
			t.Error(e)
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

func TestRestMessage(t *testing.T) {
	// Setup
	mm := test.NewManager()
	logbuf := setupWebServer(mm)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar/0001")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test requesting a message that does not exist
	w, err = testRestGet(baseURL + "/mailbox/empty/0001")
	expectCode = 404
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test GetMessage error
	w, err = testRestGet(baseURL + "/mailbox/messageerr/0001")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}

	// Test JSON message headers
	msg1 := &message.Message{
		Metadata: message.Metadata{
			Mailbox: "good",
			ID:      "0001",
			From:    &mail.Address{Name: "", Address: "from1@host"},
			To:      []*mail.Address{{Name: "", Address: "to1@host"}},
			Subject: "subject 1",
			Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
		},
		Envelope: &enmime.Envelope{
			Text: "This is some text",
			HTML: "This is some HTML",
			Root: &enmime.Part{
				Header: textproto.MIMEHeader{
					"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
					"From": []string{"noreply@inbucket.org"},
				},
			},
		},
	}
	data1 := &InputMessageData{
		Mailbox: "good",
		ID:      "0001",
		From:    "<from1@host>",
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
		Header: mail.Header{
			"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
			"From": []string{"noreply@inbucket.org"},
		},
		Text: "This is some text",
		HTML: "This is some HTML",
	}
	mm.AddMessage("good", msg1)

	// Check return code
	w, err = testRestGet(baseURL + "/mailbox/good/0001")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	// TODO transitional while refactoring
	dec := json.NewDecoder(w.Body)
	var result map[string]interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}

	if errors := data1.CompareToJSONMessageMap(result); len(errors) > 0 {
		t.Logf("%v", result)
		for _, e := range errors {
			t.Error(e)
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}
