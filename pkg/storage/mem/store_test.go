package mem

import (
	"sync"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/require"
)

// TestSuite runs storage package test suite on file store.
func TestSuite(t *testing.T) {
	test.StoreSuite(t,
		func(conf config.Storage, extHost *extension.Host) (storage.Store, func(), error) {
			s, _ := New(conf, extHost)
			destroy := func() {}
			return s, destroy, nil
		})
}

// TestMessageList verifies the operation of the global message list: mem.Store.messages.
func TestMaxSize(t *testing.T) {
	extHost := extension.NewHost()
	maxSize := int64(2048)
	s, _ := New(config.Storage{Params: map[string]string{"maxkb": "2"}}, extHost)
	boxes := []string{"alpha", "beta", "whiskey", "tango", "foxtrot"}

	// Ensure capacity so we do not block population.
	n := 10
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
	err := s.VisitMailboxes(func(messages []storage.Message) bool {
		for _, m := range messages {
			gotSize += m.Size()
		}
		return true
	})
	require.NoError(t, err, "VisitMailboxes() must succeed")

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
				panic(err) // Cannot call t.Fatal from non-test goroutine.
			}
			wg.Done()
		}(mailbox)
	}
	wg.Wait()

	// Verify zero stored messages.
	count := 0
	err = s.VisitMailboxes(func(messages []storage.Message) bool {
		count += len(messages)
		return true
	})
	require.NoError(t, err, "VisitMailboxes() must succeed")
	if count != 0 {
		t.Errorf("Got %v total messages, want: %v", count, 0)
	}
}
