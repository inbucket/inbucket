package rest

import (
	"encoding/json"
	"io"
	"net/mail"
	"os"
	"testing"
	"time"

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
	ds := test.NewStore()
	logbuf := setupWebServer(ds)

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
		From:    "from1",
		To:      []string{"to1"},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	data2 := &InputMessageData{
		Mailbox: "good",
		ID:      "0002",
		From:    "from2",
		To:      []string{"to1"},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, time.FixedZone("PDT", -700)),
	}
	msg1 := data1.MockMessage()
	msg2 := data2.MockMessage()
	ds.AddMessage("good", msg1)
	ds.AddMessage("good", msg2)

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
	ds := test.NewStore()
	logbuf := setupWebServer(ds)

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
	data1 := &InputMessageData{
		Mailbox: "good",
		ID:      "0001",
		From:    "from1",
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
		Header: mail.Header{
			"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
			"From": []string{"noreply@inbucket.org"},
		},
		Text: "This is some text",
		HTML: "This is some HTML",
	}
	msg1 := data1.MockMessage()
	ds.AddMessage("good", msg1)

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
