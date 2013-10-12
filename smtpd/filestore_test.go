package smtpd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test FileDataStore.AllMailboxes()
func TestFSAllMailboxes(t *testing.T) {
	ds := setupDataStore()
	defer teardownDataStore(ds)

	for _, name := range []string{"abby", "bill", "christa", "donald", "evelyn"} {
		// Create day old message
		date := time.Now().Add(-24 * time.Hour)
		deliverMessage(ds, name, "Old Message", date)

		// Create current message
		date = time.Now()
		deliverMessage(ds, name, "New Message", date)
	}

	mboxes, err := ds.AllMailboxes()
	assert.Nil(t, err)
	assert.Equal(t, len(mboxes), 5)
}

// Test delivering several messages to the same mailbox, meanwhile querying its
// contents with a new mailbox object each time
func TestFSDeliverMany(t *testing.T) {
	ds := setupDataStore()
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for i, subj := range subjects {
		// Check number of messages
		mb, err := ds.MailboxFor(mbName)
		if err != nil {
			panic(err)
		}
		msgs, err := mb.GetMessages()
		if err != nil {
			panic(err)
		}
		assert.Equal(t, i, len(msgs), "Expected %v message(s), but got %v", i, len(msgs))

		// Add a message
		deliverMessage(ds, mbName, subj, time.Now())
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		panic(err)
	}
	msgs, err := mb.GetMessages()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))

	// Confirm delivery order
	for i, expect := range subjects {
		subj := msgs[i].Subject()
		assert.Equal(t, expect, subj, "Expected subject %q, got %q", expect, subj)
	}
}

// Test deleting messages
func TestFSDelete(t *testing.T) {
	ds := setupDataStore()
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for _, subj := range subjects {
		// Add a message
		deliverMessage(ds, mbName, subj, time.Now())
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		panic(err)
	}
	msgs, err := mb.GetMessages()
	if err != nil {
		panic(err)
	}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))

	// Delete a couple messages
	msgs[1].Delete()
	msgs[3].Delete()

	// Confirm deletion
	mb, err = ds.MailboxFor(mbName)
	if err != nil {
		panic(err)
	}
	msgs, err = mb.GetMessages()
	if err != nil {
		panic(err)
	}

	subjects = []string{"alpha", "charlie", "echo"}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))
	for i, expect := range subjects {
		subj := msgs[i].Subject()
		assert.Equal(t, expect, subj, "Expected subject %q, got %q", expect, subj)
	}

	// Try appending one more
	deliverMessage(ds, mbName, "foxtrot", time.Now())

	mb, err = ds.MailboxFor(mbName)
	if err != nil {
		panic(err)
	}
	msgs, err = mb.GetMessages()
	if err != nil {
		panic(err)
	}

	subjects = []string{"alpha", "charlie", "echo", "foxtrot"}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))
	for i, expect := range subjects {
		subj := msgs[i].Subject()
		assert.Equal(t, expect, subj, "Expected subject %q, got %q", expect, subj)
	}

}

// setupDataStore creates a new FileDataStore in a temporary directory
func setupDataStore() *FileDataStore {
	path, err := ioutil.TempDir("", "inbucket")
	if err != nil {
		panic(err)
	}
	mailPath := filepath.Join(path, "mail")
	return &FileDataStore{path: path, mailPath: mailPath}
}

// deliverMessage creates and delivers a message to the specific mailbox, returning
// the size of the generated message.
func deliverMessage(ds *FileDataStore, mbName string, subject string, date time.Time) int {
	// Build fake SMTP message for delivery
	testMsg := make([]byte, 0, 300)
	testMsg = append(testMsg, []byte("To: somebody@host\r\n")...)
	testMsg = append(testMsg, []byte("From: somebodyelse@host\r\n")...)
	testMsg = append(testMsg, []byte(fmt.Sprintf("Subject: %s\r\n", subject))...)
	testMsg = append(testMsg, []byte("\r\n")...)
	testMsg = append(testMsg, []byte("Test Body\r\n")...)

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		panic(err)
	}
	// Create day old message
	msg := &FileMessage{
		mailbox:  mb.(*FileMailbox),
		writable: true,
		Fdate:    date,
		Fid:      generateId(date),
	}
	msg.Append(testMsg)
	if err = msg.Close(); err != nil {
		panic(err)
	}

	return len(testMsg)
}

func teardownDataStore(ds *FileDataStore) {
	if err := os.RemoveAll(ds.path); err != nil {
		panic(err)
	}
}
