package storage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/test"
)

func TestDoRetentionScan(t *testing.T) {
	ds := test.NewStore()
	// Mockup some different aged messages (num is in hours)
	new1 := mockMessage(0)
	new2 := mockMessage(1)
	new3 := mockMessage(2)
	old1 := mockMessage(4)
	old2 := mockMessage(12)
	old3 := mockMessage(24)
	ds.AddMessage("mb1", new1)
	new1.On("Mailbox").Return("mb1")
	ds.AddMessage("mb1", old1)
	old1.On("Mailbox").Return("mb1")
	ds.AddMessage("mb1", old2)
	old2.On("Mailbox").Return("mb1")
	ds.AddMessage("mb2", old3)
	old3.On("Mailbox").Return("mb2")
	ds.AddMessage("mb2", new2)
	new2.On("Mailbox").Return("mb2")
	ds.AddMessage("mb3", new3)
	new3.On("Mailbox").Return("mb3")
	// Test 4 hour retention
	cfg := config.DataStoreConfig{
		RetentionMinutes: 239,
		RetentionSleep:   0,
	}
	shutdownChan := make(chan bool)
	rs := storage.NewRetentionScanner(cfg, ds, shutdownChan)
	if err := rs.DoScan(); err != nil {
		t.Error(err)
	}
	// Delete should not have been called on new messages
	for _, m := range []storage.StoreMessage{new1, new2, new3} {
		if ds.MessageDeleted(m) {
			t.Errorf("Expected %v to be present, was deleted", m.ID())
		}
	}
	// Delete should have been called once on old messages
	for _, m := range []storage.StoreMessage{old1, old2, old3} {
		if !ds.MessageDeleted(m) {
			t.Errorf("Expected %v to be deleted, was present", m.ID())
		}
	}
}

// Make a MockMessage of a specific age
func mockMessage(ageHours int) *storage.MockMessage {
	msg := &storage.MockMessage{}
	msg.On("ID").Return(fmt.Sprintf("MSG[age=%vh]", ageHours))
	msg.On("Date").Return(time.Now().Add(time.Duration(ageHours*-1) * time.Hour))
	msg.On("Delete").Return(nil)
	return msg
}
