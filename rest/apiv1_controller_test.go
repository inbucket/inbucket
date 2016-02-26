package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/mail"
	"os"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/smtpd"
)

const (
	baseURL = "http://localhost/api/v1"

	// JSON map keys
	mailboxKey = "mailbox"
	idKey      = "id"
	fromKey    = "from"
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
	ds := &MockDataStore{}
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
	emptybox := &MockMailbox{}
	ds.On("MailboxFor", "empty").Return(emptybox, nil)
	emptybox.On("GetMessages").Return([]smtpd.Message{}, nil)

	w, err = testRestGet(baseURL + "/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test MailboxFor error
	ds.On("MailboxFor", "error").Return(&MockMailbox{}, fmt.Errorf("Internal error"))
	w, err = testRestGet(baseURL + "/mailbox/error")
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

	// Test MailboxFor error
	error2box := &MockMailbox{}
	ds.On("MailboxFor", "error2").Return(error2box, nil)
	error2box.On("GetMessages").Return([]smtpd.Message{}, fmt.Errorf("Internal error 2"))

	w, err = testRestGet(baseURL + "/mailbox/error2")
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
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	data2 := &InputMessageData{
		Mailbox: "good",
		ID:      "0002",
		From:    "from2",
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, time.FixedZone("PDT", -700)),
	}
	goodbox := &MockMailbox{}
	ds.On("MailboxFor", "good").Return(goodbox, nil)
	msg1 := data1.MockMessage()
	msg2 := data2.MockMessage()
	goodbox.On("GetMessages").Return([]smtpd.Message{msg1, msg2}, nil)

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
		t.Errorf("Expected 2 results, got %v", len(result))
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
	ds := &MockDataStore{}
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
	emptybox := &MockMailbox{}
	ds.On("MailboxFor", "empty").Return(emptybox, nil)
	emptybox.On("GetMessage", "0001").Return(&MockMessage{}, smtpd.ErrNotExist)

	w, err = testRestGet(baseURL + "/mailbox/empty/0001")
	expectCode = 404
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test MailboxFor error
	ds.On("MailboxFor", "error").Return(&MockMailbox{}, fmt.Errorf("Internal error"))
	w, err = testRestGet(baseURL + "/mailbox/error/0001")
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

	// Test GetMessage error
	error2box := &MockMailbox{}
	ds.On("MailboxFor", "error2").Return(error2box, nil)
	error2box.On("GetMessage", "0001").Return(&MockMessage{}, fmt.Errorf("Internal error 2"))

	w, err = testRestGet(baseURL + "/mailbox/error2/0001")
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
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
		Header: mail.Header{
			"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
			"From": []string{"noreply@inbucket.org"},
		},
		Text: "This is some text",
		HTML: "This is some HTML",
	}
	goodbox := &MockMailbox{}
	ds.On("MailboxFor", "good").Return(goodbox, nil)
	msg1 := data1.MockMessage()
	goodbox.On("GetMessage", "0001").Return(msg1, nil)

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
