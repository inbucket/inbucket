package storage_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/test"
)

func TestDoRetentionScan(t *testing.T) {
	ds := test.NewStore()

	// Mockup some different aged messages (num is in hours)
	new1 := stubMessage("mb1", 0)
	new2 := stubMessage("mb2", 1)
	new3 := stubMessage("mb3", 2)
	old1 := stubMessage("mb1", 4)
	old2 := stubMessage("mb1", 12)
	old3 := stubMessage("mb2", 24)
	ds.AddMessage(new1)
	ds.AddMessage(old1)
	ds.AddMessage(old2)
	ds.AddMessage(old3)
	ds.AddMessage(new2)
	ds.AddMessage(new3)

	// Test 4 hour retention
	cfg := config.Storage{
		RetentionPeriod: 239 * time.Minute,
		RetentionSleep:  0,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rs := storage.NewRetentionScanner(cfg, ds)
	if err := rs.DoScan(ctx); err != nil {
		t.Error(err)
	}

	// Delete should not have been called on new messages
	for _, m := range []storage.Message{new1, new2, new3} {
		if ds.MessageDeleted(m) {
			t.Errorf("Expected %v to be present, was deleted", m.ID())
		}
	}

	// Delete should have been called once on old messages
	for _, m := range []storage.Message{old1, old2, old3} {
		if !ds.MessageDeleted(m) {
			t.Errorf("Expected %v to be deleted, was present", m.ID())
		}
	}
}

// stubMessage creates a message stub of a specific age
func stubMessage(mailbox string, ageHours int) storage.Message {
	return &message.Delivery{
		Meta: message.Metadata{
			Mailbox: mailbox,
			ID:      fmt.Sprintf("MSG[age=%vh]", ageHours),
			Date:    time.Now().Add(time.Duration(ageHours*-1) * time.Hour),
		},
	}
}
