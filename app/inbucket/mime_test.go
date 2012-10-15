package inbucket

import (
	"bufio"
	"net/mail"
	"os"
	"testing"
)

func TestSomething(t *testing.T) {
	// Open test email for parsing
	raw, err := os.Open("../../test-data/html-mime-attach.raw")
	if err != nil {
		t.Fatalf("Failed to open test data: %v", err)
	}

	// Parse email into a mail.Message object like we do
	reader := bufio.NewReader(raw)
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	_, err = ParseMIMEMessage(msg)
	if err != nil {
		t.Fatalf("Failed to parse mime: %v", err)
	}
}
