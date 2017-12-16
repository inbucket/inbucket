package smtpd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/stretchr/testify/assert"
)

// Test directory structure created by filestore
func TestFSDirStructure(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)
	root := ds.path

	// james hashes to 474ba67bdb289c6263b36dfd8a7bed6c85b04943
	mbName := "james"

	// Check filestore root exists
	assert.True(t, isDir(root), "Expected %q to be a directory", root)

	// Check mail dir exists
	expect := filepath.Join(root, "mail")
	assert.True(t, isDir(expect), "Expected %q to be a directory", expect)

	// Check first hash section does not exist
	expect = filepath.Join(root, "mail", "474")
	assert.False(t, isDir(expect), "Expected %q to not exist", expect)

	// Deliver test message
	id1, _ := deliverMessage(ds, mbName, "test", time.Now())

	// Check path to message exists
	assert.True(t, isDir(expect), "Expected %q to be a directory", expect)
	expect = filepath.Join(expect, "474ba6")
	assert.True(t, isDir(expect), "Expected %q to be a directory", expect)
	expect = filepath.Join(expect, "474ba67bdb289c6263b36dfd8a7bed6c85b04943")
	assert.True(t, isDir(expect), "Expected %q to be a directory", expect)

	// Check files
	mbPath := expect
	expect = filepath.Join(mbPath, "index.gob")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)
	expect = filepath.Join(mbPath, id1+".raw")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)

	// Deliver second test message
	id2, _ := deliverMessage(ds, mbName, "test 2", time.Now())

	// Check files
	expect = filepath.Join(mbPath, "index.gob")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)
	expect = filepath.Join(mbPath, id2+".raw")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)

	// Delete message
	mb, err := ds.MailboxFor(mbName)
	assert.Nil(t, err)
	msg, err := mb.GetMessage(id1)
	assert.Nil(t, err)
	err = msg.Delete()
	assert.Nil(t, err)

	// Message should be removed
	expect = filepath.Join(mbPath, id1+".raw")
	assert.False(t, isPresent(expect), "Did not expect %q to exist", expect)
	expect = filepath.Join(mbPath, "index.gob")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)

	// Delete message
	msg, err = mb.GetMessage(id2)
	assert.Nil(t, err)
	err = msg.Delete()
	assert.Nil(t, err)

	// Message should be removed
	expect = filepath.Join(mbPath, id2+".raw")
	assert.False(t, isPresent(expect), "Did not expect %q to exist", expect)

	// No messages, index & maildir should be removed
	expect = filepath.Join(mbPath, "index.gob")
	assert.False(t, isPresent(expect), "Did not expect %q to exist", expect)
	expect = mbPath
	assert.False(t, isPresent(expect), "Did not expect %q to exist", expect)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test FileDataStore.AllMailboxes()
func TestFSAllMailboxes(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
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

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test delivering several messages to the same mailbox, meanwhile querying its
// contents with a new mailbox object each time
func TestFSDeliverMany(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for i, subj := range subjects {
		// Check number of messages
		mb, err := ds.MailboxFor(mbName)
		if err != nil {
			t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
		}
		msgs, err := mb.GetMessages()
		if err != nil {
			t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
		}
		assert.Equal(t, i, len(msgs), "Expected %v message(s), but got %v", i, len(msgs))

		// Add a message
		deliverMessage(ds, mbName, subj, time.Now())
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err := mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
	}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))

	// Confirm delivery order
	for i, expect := range subjects {
		subj := msgs[i].Subject()
		assert.Equal(t, expect, subj, "Expected subject %q, got %q", expect, subj)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test deleting messages
func TestFSDelete(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for _, subj := range subjects {
		// Add a message
		deliverMessage(ds, mbName, subj, time.Now())
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err := mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
	}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))

	// Delete a couple messages
	_ = msgs[1].Delete()
	_ = msgs[3].Delete()

	// Confirm deletion
	mb, err = ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err = mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
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
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err = mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
	}

	subjects = []string{"alpha", "charlie", "echo", "foxtrot"}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))
	for i, expect := range subjects {
		subj := msgs[i].Subject()
		assert.Equal(t, expect, subj, "Expected subject %q, got %q", expect, subj)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test purging a mailbox
func TestFSPurge(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for _, subj := range subjects {
		// Add a message
		deliverMessage(ds, mbName, subj, time.Now())
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err := mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
	}
	assert.Equal(t, len(subjects), len(msgs), "Expected %v message(s), but got %v",
		len(subjects), len(msgs))

	// Purge mailbox
	err = mb.Purge()
	assert.Nil(t, err)

	// Confirm deletion
	mb, err = ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	msgs, err = mb.GetMessages()
	if err != nil {
		t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
	}

	assert.Equal(t, len(msgs), 0, "Expected mailbox to have zero messages, got %v", len(msgs))

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test message size calculation
func TestFSSize(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"a", "br", "much longer than the others"}
	sentIds := make([]string, len(subjects))
	sentSizes := make([]int64, len(subjects))

	for i, subj := range subjects {
		// Add a message
		id, size := deliverMessage(ds, mbName, subj, time.Now())
		sentIds[i] = id
		sentSizes[i] = size
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}
	for i, id := range sentIds {
		msg, err := mb.GetMessage(id)
		assert.Nil(t, err)

		expect := sentSizes[i]
		size := msg.Size()
		assert.Equal(t, expect, size, "Expected size of %v, got %v", expect, size)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test missing files
func TestFSMissing(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"a", "b", "c"}
	sentIds := make([]string, len(subjects))

	for i, subj := range subjects {
		// Add a message
		id, _ := deliverMessage(ds, mbName, subj, time.Now())
		sentIds[i] = id
	}

	mb, err := ds.MailboxFor(mbName)
	if err != nil {
		t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
	}

	// Delete a message file without removing it from index
	msg, err := mb.GetMessage(sentIds[1])
	assert.Nil(t, err)
	fmsg := msg.(*FileMessage)
	_ = os.Remove(fmsg.rawPath())
	msg, err = mb.GetMessage(sentIds[1])
	assert.Nil(t, err)

	// Try to read parts of message
	_, err = msg.ReadHeader()
	assert.Error(t, err)
	_, err = msg.ReadBody()
	assert.Error(t, err)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test delivering several messages to the same mailbox, see if message cap works
func TestFSMessageCap(t *testing.T) {
	mbCap := 10
	ds, logbuf := setupDataStore(config.DataStoreConfig{MailboxMsgCap: mbCap})
	defer teardownDataStore(ds)

	mbName := "captain"
	for i := 0; i < 20; i++ {
		// Add a message
		subj := fmt.Sprintf("subject %v", i)
		deliverMessage(ds, mbName, subj, time.Now())
		t.Logf("Delivered %q", subj)

		// Check number of messages
		mb, err := ds.MailboxFor(mbName)
		if err != nil {
			t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
		}
		msgs, err := mb.GetMessages()
		if err != nil {
			t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
		}
		if len(msgs) > mbCap {
			t.Errorf("Mailbox should be capped at %v messages, but has %v", mbCap, len(msgs))
		}

		// Check that the first message is correct
		first := i - mbCap + 1
		if first < 0 {
			first = 0
		}
		firstSubj := fmt.Sprintf("subject %v", first)
		if firstSubj != msgs[0].Subject() {
			t.Errorf("Expected first subject to be %q, got %q", firstSubj, msgs[0].Subject())
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test delivering several messages to the same mailbox, see if no message cap works
func TestFSNoMessageCap(t *testing.T) {
	mbCap := 0
	ds, logbuf := setupDataStore(config.DataStoreConfig{MailboxMsgCap: mbCap})
	defer teardownDataStore(ds)

	mbName := "captain"
	for i := 0; i < 20; i++ {
		// Add a message
		subj := fmt.Sprintf("subject %v", i)
		deliverMessage(ds, mbName, subj, time.Now())
		t.Logf("Delivered %q", subj)

		// Check number of messages
		mb, err := ds.MailboxFor(mbName)
		if err != nil {
			t.Fatalf("Failed to MailboxFor(%q): %v", mbName, err)
		}
		msgs, err := mb.GetMessages()
		if err != nil {
			t.Fatalf("Failed to GetMessages for %q: %v", mbName, err)
		}
		if len(msgs) != i+1 {
			t.Errorf("Expected %v messages, got %v", i+1, len(msgs))
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// Test Get the latest message
func TestGetLatestMessage(t *testing.T) {
	ds, logbuf := setupDataStore(config.DataStoreConfig{})
	defer teardownDataStore(ds)

	// james hashes to 474ba67bdb289c6263b36dfd8a7bed6c85b04943
	mbName := "james"

	// Test empty mailbox
	mb, err := ds.MailboxFor(mbName)
	assert.Nil(t, err)
	msg, err := mb.GetMessage("latest")
	assert.Error(t, err)
	fmt.Println(msg)

	// Deliver test message
	deliverMessage(ds, mbName, "test", time.Now())

	// Deliver test message 2
	id2, _ := deliverMessage(ds, mbName, "test 2", time.Now())

	// Test get the latest message
	mb, err = ds.MailboxFor(mbName)
	assert.Nil(t, err)
	msg, err = mb.GetMessage("latest")
	assert.Nil(t, err)
	assert.True(t, msg.ID() == id2, "Expected %q to be equal to %q", msg.ID(), id2)

	// Deliver test message 3
	id3, _ := deliverMessage(ds, mbName, "test 3", time.Now())

	mb, err = ds.MailboxFor(mbName)
	assert.Nil(t, err)
	msg, err = mb.GetMessage("latest")
	assert.Nil(t, err)
	assert.True(t, msg.ID() == id3, "Expected %q to be equal to %q", msg.ID(), id3)

	// Test wrong id
	msg, err = mb.GetMessage("wrongid")
	assert.Error(t, err)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// setupDataStore creates a new FileDataStore in a temporary directory
func setupDataStore(cfg config.DataStoreConfig) (*FileDataStore, *bytes.Buffer) {
	path, err := ioutil.TempDir("", "inbucket")
	if err != nil {
		panic(err)
	}

	// Capture log output
	buf := new(bytes.Buffer)
	log.SetOutput(buf)

	cfg.Path = path
	return NewFileDataStore(cfg).(*FileDataStore), buf
}

// deliverMessage creates and delivers a message to the specific mailbox, returning
// the size of the generated message.
func deliverMessage(ds *FileDataStore, mbName string, subject string,
	date time.Time) (id string, size int64) {
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
	// Create message object
	id = generateID(date)
	msg, err := mb.NewMessage()
	if err != nil {
		panic(err)
	}
	fmsg := msg.(*FileMessage)
	fmsg.Fdate = date
	fmsg.Fid = id
	if err = msg.Append(testMsg); err != nil {
		panic(err)
	}
	if err = msg.Close(); err != nil {
		panic(err)
	}

	return id, int64(len(testMsg))
}

func teardownDataStore(ds *FileDataStore) {
	if err := os.RemoveAll(ds.path); err != nil {
		panic(err)
	}
}

func isPresent(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func isFile(path string) bool {
	if fi, err := os.Lstat(path); err == nil {
		return !fi.IsDir()
	}
	return false
}

func isDir(path string) bool {
	if fi, err := os.Lstat(path); err == nil {
		return fi.IsDir()
	}
	return false
}
