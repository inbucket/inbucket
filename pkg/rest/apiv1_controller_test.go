package rest

import (
	"encoding/json"
	"io"
	"net/mail"
	"net/textproto"
	"os"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/jhillyerd/enmime/v2"
)

func TestRestMailboxList(t *testing.T) {
	// Setup
	mm := test.NewManager()
	logbuf := setupWebServer(mm)

	// Test invalid mailbox name
	w, err := testRestGet("http://localhost/api/v1/mailbox/foo%20bar")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test empty mailbox
	w, err = testRestGet("http://localhost/api/v1/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test Mailbox error
	w, err = testRestGet("http://localhost/api/v1/mailbox/messageserr")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test JSON message headers
	tzPDT := time.FixedZone("PDT", -7*3600)
	tzPST := time.FixedZone("PST", -8*3600)
	meta1 := event.MessageMetadata{
		Mailbox: "good",
		ID:      "0001",
		From:    &mail.Address{Name: "", Address: "from1@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, tzPST),
	}
	meta2 := event.MessageMetadata{
		Mailbox: "good",
		ID:      "0002",
		From:    &mail.Address{Name: "", Address: "from2@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, tzPDT),
	}
	mm.AddMessage("good", &message.Message{MessageMetadata: meta1})
	mm.AddMessage("good", &message.Message{MessageMetadata: meta2})

	// Check return code
	w, err = testRestGet("http://localhost/api/v1/mailbox/good")
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

	decodedStringEquals(t, result, "[0]/mailbox", "good")
	decodedStringEquals(t, result, "[0]/id", "0001")
	decodedStringEquals(t, result, "[0]/from", "<from1@host>")
	decodedStringEquals(t, result, "[0]/to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "[0]/subject", "subject 1")
	decodedStringEquals(t, result, "[0]/date", "2012-02-01T10:11:12.000000253-08:00")
	decodedNumberEquals(t, result, "[0]/posix-millis", 1328119872000)
	decodedNumberEquals(t, result, "[0]/size", 0)
	decodedBoolEquals(t, result, "[0]/seen", false)
	decodedStringEquals(t, result, "[1]/mailbox", "good")
	decodedStringEquals(t, result, "[1]/id", "0002")
	decodedStringEquals(t, result, "[1]/from", "<from2@host>")
	decodedStringEquals(t, result, "[1]/to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "[1]/subject", "subject 2")
	decodedStringEquals(t, result, "[1]/date", "2012-07-01T10:11:12.000000253-07:00")
	decodedNumberEquals(t, result, "[1]/posix-millis", 1341162672000)
	decodedNumberEquals(t, result, "[1]/size", 0)
	decodedBoolEquals(t, result, "[1]/seen", false)

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
	w, err := testRestGet("http://localhost/api/v1/mailbox/foo%20bar/0001")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test requesting a message that does not exist
	w, err = testRestGet("http://localhost/api/v1/mailbox/empty/0001")
	expectCode = 404
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test GetMessage error
	w, err = testRestGet("http://localhost/api/v1/mailbox/messageerr/0001")
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
	tzPST := time.FixedZone("PST", -8*3600)
	msg1 := message.New(
		event.MessageMetadata{
			Mailbox: "good",
			ID:      "0001",
			From:    &mail.Address{Name: "", Address: "from1@host"},
			To:      []*mail.Address{{Name: "", Address: "to1@host"}},
			Subject: "subject 1",
			Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, tzPST),
			Seen:    true,
		},
		&enmime.Envelope{
			Text: "This is some text",
			HTML: "This is some HTML",
			Root: &enmime.Part{
				Header: textproto.MIMEHeader{
					"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
					"From": []string{"noreply@inbucket.org"},
				},
			},
			Attachments: []*enmime.Part{{
				FileName:    "favicon.png",
				ContentType: "image/png",
			}},
			Inlines: []*enmime.Part{{
				FileName:    "statement.pdf",
				ContentType: "application/pdf",
			}},
		},
	)
	mm.AddMessage("good", msg1)

	// Check return code
	w, err = testRestGet("http://localhost/api/v1/mailbox/good/0001")
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

	decodedStringEquals(t, result, "mailbox", "good")
	decodedStringEquals(t, result, "id", "0001")
	decodedStringEquals(t, result, "from", "<from1@host>")
	decodedStringEquals(t, result, "to/[0]", "<to1@host>")
	decodedStringEquals(t, result, "subject", "subject 1")
	decodedStringEquals(t, result, "date", "2012-02-01T10:11:12.000000253-08:00")
	decodedNumberEquals(t, result, "posix-millis", 1328119872000)
	decodedNumberEquals(t, result, "size", 0)
	decodedBoolEquals(t, result, "seen", true)
	decodedStringEquals(t, result, "body/text", "This is some text")
	decodedStringEquals(t, result, "body/html", "This is some HTML")
	decodedStringEquals(t, result, "header/To/[0]", "fred@fish.com")
	decodedStringEquals(t, result, "header/To/[1]", "keyword@nsa.gov")
	decodedStringEquals(t, result, "header/From/[0]", "noreply@inbucket.org")
	decodedStringEquals(t, result, "attachments/[0]/filename", "statement.pdf")
	decodedStringEquals(t, result, "attachments/[0]/content-type", "application/pdf")
	decodedStringEquals(t, result, "attachments/[0]/download-link", "http://localhost/serve/mailbox/good/0001/attach/0/statement.pdf")
	decodedStringEquals(t, result, "attachments/[0]/view-link", "http://localhost/serve/mailbox/good/0001/attach/0/statement.pdf")
	decodedStringEquals(t, result, "attachments/[1]/filename", "favicon.png")
	decodedStringEquals(t, result, "attachments/[1]/content-type", "image/png")
	decodedStringEquals(t, result, "attachments/[1]/download-link", "http://localhost/serve/mailbox/good/0001/attach/1/favicon.png")
	decodedStringEquals(t, result, "attachments/[1]/view-link", "http://localhost/serve/mailbox/good/0001/attach/1/favicon.png")

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

func TestRestMarkSeen(t *testing.T) {
	mm := test.NewManager()
	logbuf := setupWebServer(mm)
	// Create some messages.
	tzPDT := time.FixedZone("PDT", -7*3600)
	tzPST := time.FixedZone("PST", -8*3600)
	meta1 := event.MessageMetadata{
		Mailbox: "good",
		ID:      "0001",
		From:    &mail.Address{Name: "", Address: "from1@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, tzPST),
	}
	meta2 := event.MessageMetadata{
		Mailbox: "good",
		ID:      "0002",
		From:    &mail.Address{Name: "", Address: "from2@host"},
		To:      []*mail.Address{{Name: "", Address: "to1@host"}},
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, tzPDT),
	}
	mm.AddMessage("good", &message.Message{MessageMetadata: meta1})
	mm.AddMessage("good", &message.Message{MessageMetadata: meta2})
	// Mark one read.
	w, err := testRestPatch("http://localhost/api/v1/mailbox/good/0002", `{"seen":true}`)
	expectCode := 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}
	// Get mailbox.
	w, err = testRestGet("http://localhost/api/v1/mailbox/good")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}
	// Check JSON.
	dec := json.NewDecoder(w.Body)
	var result []interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %v", len(result))
	}
	decodedStringEquals(t, result, "[0]/id", "0001")
	decodedBoolEquals(t, result, "[0]/seen", false)
	decodedStringEquals(t, result, "[1]/id", "0002")
	decodedBoolEquals(t, result, "[1]/seen", true)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}
