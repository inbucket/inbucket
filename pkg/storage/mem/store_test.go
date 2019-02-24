package mem

import (
	"sync"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/test"
)

// TestSuite runs storage package test suite on file store.
func TestSuite(t *testing.T) {
	test.StoreSuite(t, func(conf config.Storage) (storage.Store, func(), error) {
		s, _ := New(conf)
		destroy := func() {}
		return s, destroy, nil
	})
}

// TestMessageList verifies the operation of the global message list: mem.Store.messages.
func TestMaxSize(t *testing.T) {
	maxSize := int64(2048)
	s, _ := New(config.Storage{Params: map[string]string{"maxkb": "2"}})
	boxes := []string{"alpha", "beta", "whiskey", "tango", "foxtrot"}
	n := 10
	// total := 50
	sizeChan := make(chan int64, len(boxes))
	// Populate mailboxes concurrently.
	for _, mailbox := range boxes {
		go func(mailbox string) {
			size := int64(0)
			for i := 0; i < n; i++ {
				_, nbytes := test.DeliverToStore(t, s, mailbox, "subject", time.Now())
				size += nbytes
			}
			sizeChan <- size
		}(mailbox)
	}
	// Wait for sizes.
	sentBytesTotal := int64(0)
	for range boxes {
		sentBytesTotal += <-sizeChan
	}
	// Calculate actual size.
	gotSize := int64(0)
	s.VisitMailboxes(func(messages []storage.Message) bool {
		for _, m := range messages {
			gotSize += m.Size()
		}
		return true
	})
	// Verify state. Messages are ~75 bytes each.
	if gotSize < 2048-75 {
		t.Errorf("Got total size %v, want greater than: %v", gotSize, 2048-75)
	}
	if gotSize > maxSize {
		t.Errorf("Got total size %v, want less than: %v", gotSize, maxSize)
	}
	// Purge all messages concurrently, testing for deadlocks.
	wg := &sync.WaitGroup{}
	wg.Add(len(boxes))
	for _, mailbox := range boxes {
		go func(mailbox string) {
			err := s.PurgeMessages(mailbox)
			if err != nil {
				t.Fatal(err)
			}
			wg.Done()
		}(mailbox)
	}
	wg.Wait()
	count := 0
	s.VisitMailboxes(func(messages []storage.Message) bool {
		count += len(messages)
		return true
	})
	if count != 0 {
		t.Errorf("Got %v total messages, want: %v", count, 0)
	}
}
