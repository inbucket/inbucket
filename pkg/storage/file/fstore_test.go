package file

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/test"
	"github.com/stretchr/testify/assert"
)

// TestSuite runs storage package test suite on file store.
func TestSuite(t *testing.T) {
	test.StoreSuite(t, func() (storage.Store, func(), error) {
		ds, _ := setupDataStore(config.Storage{})
		destroy := func() {
			teardownDataStore(ds)
		}
		return ds, destroy, nil
	})
}

// Test directory structure created by filestore
func TestFSDirStructure(t *testing.T) {
	ds, logbuf := setupDataStore(config.Storage{})
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
	err := ds.RemoveMessage(mbName, id1)
	assert.Nil(t, err)

	// Message should be removed
	expect = filepath.Join(mbPath, id1+".raw")
	assert.False(t, isPresent(expect), "Did not expect %q to exist", expect)
	expect = filepath.Join(mbPath, "index.gob")
	assert.True(t, isFile(expect), "Expected %q to be a file", expect)

	// Delete message
	err = ds.RemoveMessage(mbName, id2)
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

// Test missing files
func TestFSMissing(t *testing.T) {
	ds, logbuf := setupDataStore(config.Storage{})
	defer teardownDataStore(ds)

	mbName := "fred"
	subjects := []string{"a", "b", "c"}
	sentIds := make([]string, len(subjects))

	for i, subj := range subjects {
		// Add a message
		id, _ := deliverMessage(ds, mbName, subj, time.Now())
		sentIds[i] = id
	}

	// Delete a message file without removing it from index
	msg, err := ds.GetMessage(mbName, sentIds[1])
	assert.Nil(t, err)
	fmsg := msg.(*Message)
	_ = os.Remove(fmsg.rawPath())
	msg, err = ds.GetMessage(mbName, sentIds[1])
	assert.Nil(t, err)

	// Try to read parts of message
	_, err = msg.Source()
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
	ds, logbuf := setupDataStore(config.Storage{MailboxMsgCap: mbCap})
	defer teardownDataStore(ds)

	mbName := "captain"
	for i := 0; i < 20; i++ {
		// Add a message
		subj := fmt.Sprintf("subject %v", i)
		deliverMessage(ds, mbName, subj, time.Now())
		t.Logf("Delivered %q", subj)

		// Check number of messages
		msgs, err := ds.GetMessages(mbName)
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
	ds, logbuf := setupDataStore(config.Storage{MailboxMsgCap: mbCap})
	defer teardownDataStore(ds)

	mbName := "captain"
	for i := 0; i < 20; i++ {
		// Add a message
		subj := fmt.Sprintf("subject %v", i)
		deliverMessage(ds, mbName, subj, time.Now())
		t.Logf("Delivered %q", subj)

		// Check number of messages
		msgs, err := ds.GetMessages(mbName)
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
	ds, logbuf := setupDataStore(config.Storage{})
	defer teardownDataStore(ds)

	// james hashes to 474ba67bdb289c6263b36dfd8a7bed6c85b04943
	mbName := "james"

	// Test empty mailbox
	msg, err := ds.GetMessage(mbName, "latest")
	assert.Nil(t, msg)
	assert.Error(t, err)

	// Deliver test message
	deliverMessage(ds, mbName, "test", time.Now())

	// Deliver test message 2
	id2, _ := deliverMessage(ds, mbName, "test 2", time.Now())

	// Test get the latest message
	msg, err = ds.GetMessage(mbName, "latest")
	assert.Nil(t, err)
	assert.True(t, msg.ID() == id2, "Expected %q to be equal to %q", msg.ID(), id2)

	// Deliver test message 3
	id3, _ := deliverMessage(ds, mbName, "test 3", time.Now())

	msg, err = ds.GetMessage(mbName, "latest")
	assert.Nil(t, err)
	assert.True(t, msg.ID() == id3, "Expected %q to be equal to %q", msg.ID(), id3)

	// Test wrong id
	_, err = ds.GetMessage(mbName, "wrongid")
	assert.Error(t, err)

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

// setupDataStore creates a new FileDataStore in a temporary directory
func setupDataStore(cfg config.Storage) (*Store, *bytes.Buffer) {
	path, err := ioutil.TempDir("", "inbucket")
	if err != nil {
		panic(err)
	}

	// Capture log output
	buf := new(bytes.Buffer)
	log.SetOutput(buf)

	cfg.Path = path
	return New(cfg).(*Store), buf
}

// deliverMessage creates and delivers a message to the specific mailbox, returning
// the size of the generated message.
func deliverMessage(ds *Store, mbName string, subject string, date time.Time) (string, int64) {
	// Build message for delivery
	meta := message.Metadata{
		Mailbox: mbName,
		To:      []*mail.Address{{Name: "", Address: "somebody@host"}},
		From:    &mail.Address{Name: "", Address: "somebodyelse@host"},
		Subject: subject,
		Date:    date,
	}
	testMsg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\nTest Body\r\n",
		meta.To[0].Address, meta.From.Address, subject)
	delivery := &message.Delivery{
		Meta:   meta,
		Reader: ioutil.NopCloser(strings.NewReader(testMsg)),
	}
	id, err := ds.AddMessage(delivery)
	if err != nil {
		panic(err)
	}
	return id, int64(len(testMsg))
}

func teardownDataStore(ds *Store) {
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
