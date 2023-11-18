package test_test

import (
	"fmt"
	"net/mail"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testMessageIDSource uint32

func TestStoreStubRemoveMessage(t *testing.T) {
	ss := test.NewStore()

	// Add messages.
	inputMsgs := make([]*message.Delivery, 5)
	for i := range inputMsgs {
		subject := fmt.Sprintf("%s message %v", "box1", i)
		inputMsgs[i] = makeTestMessage("box1", subject)
		id, err := ss.AddMessage(inputMsgs[i])
		require.NoError(t, err)
		require.NotEmpty(t, id, "AddMessage() must return an ID")
	}

	// Delete second message.
	deleted := inputMsgs[1]
	err := ss.RemoveMessage("box1", deleted.ID())
	assert.NoError(t, err, "DeleteMessage must not fail")

	// Verify message is not in mailbox.
	messages, err := ss.GetMessages("box1")
	assert.NoError(t, err)
	assert.NotContains(t, messages, deleted, "Mailbox should not contain msg %q", deleted.ID())

	// Verify message is no longer retrievable.
	got, err := ss.GetMessage("box1", deleted.ID())
	assert.Error(t, err)
	assert.Nil(t, got, "Message should have been nil")

	// Verify message is in deleted list.
	assert.True(t, ss.MessageDeleted(deleted), "Message %q should be in deleted list", deleted.ID())
}

func TestStoreStubMailboxAddGetVisit(t *testing.T) {
	ss := test.NewStore()

	tcs := []struct {
		mailbox string
		count   int
	}{
		{mailbox: "box1", count: 1},
		{mailbox: "box2", count: 1},
		{mailbox: "box3", count: 3},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.mailbox, func(t *testing.T) {
			var err error

			// Add messages.
			inputMsgs := make([]*message.Delivery, tc.count)
			for i := range inputMsgs {
				subject := fmt.Sprintf("%s message %v", tc.mailbox, i)
				inputMsgs[i] = makeTestMessage(tc.mailbox, subject)
				id, err := ss.AddMessage(inputMsgs[i])
				require.NoError(t, err)
				require.NotEmpty(t, id, "AddMessage() must return an ID")
			}

			// Verify entire mailbox contents.
			gotMsgs, err := ss.GetMessages(tc.mailbox)
			require.NoError(t, err, "GetMessages() should not error")
			require.NoError(t, err)
			assert.Len(t, gotMsgs, tc.count, "GetMessages() returned wrong number of items")
			for _, want := range inputMsgs {
				assert.Contains(t, gotMsgs, want, "GetMessages() did not return expected message")
			}

			// Fetch and verify individual messages.
			for _, want := range inputMsgs {
				got, err := ss.GetMessage(tc.mailbox, want.ID())
				require.NoError(t, err, "GetMessage() should not error")
				assert.Equal(t, want, got, "GetMessage() returned unexpected value")
			}
		})
	}

	t.Run("VisitMailboxes counts", func(t *testing.T) {
		expectCounts := make(map[string]int, len(tcs))
		for _, tc := range tcs {
			expectCounts[tc.mailbox] = tc.count
		}

		// Verify message count of each visited mailbox.
		err := ss.VisitMailboxes(func(m []storage.Message) (cont bool) {
			require.NotEmpty(t, m, "Visitor called with empty message slice")
			mailbox := m[0].Mailbox()

			want, ok := expectCounts[mailbox]
			assert.True(t, ok, "Mailbox %q was unexpected", mailbox)
			assert.Equal(t, want, len(m), "Unexpected message count for mailbox %q", mailbox)

			delete(expectCounts, mailbox)

			return true
		})
		require.NoError(t, err, "VisitMailboxes() must not fail")

		// Verify all mailboxes were visited.
		assert.Empty(t, expectCounts, "Failed to visit mailboxes: %v",
			reflect.ValueOf(expectCounts).MapKeys())
	})
}

func makeTestMessage(mailbox string, subject string) *message.Delivery {
	id := fmt.Sprintf("%06d", atomic.AddUint32(&testMessageIDSource, 1))
	from := mail.Address{Name: "From Test", Address: "from@example.com"}
	to := mail.Address{Name: "To Test", Address: "to@example.com"}

	return &message.Delivery{
		Meta: event.MessageMetadata{
			ID:      id,
			Mailbox: mailbox,
			From:    &from,
			To:      []*mail.Address{&to},
			Date:    time.Now(),
			Subject: subject,
		},
		Reader: strings.NewReader(fmt.Sprintf(
			"From: %s\nTo: %s\nSubject: %s\n\nTest message about %q\n",
			from, to, subject, subject,
		)),
	}
}
