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
			assert.Len(t, gotMsgs, tc.count, "GetMessages() returned wrong number of items")
		input:
			for _, want := range inputMsgs {
				for _, got := range gotMsgs {
					if got.ID() == want.ID() && got.Mailbox() == want.Mailbox() {
						continue input
					}
				}
				t.Errorf("GetMessages() did not return message %q for mailbox %q",
					want.ID(), want.Mailbox())
			}

			// Fetch and verify individual messages.
			for _, want := range inputMsgs {
				got, err := ss.GetMessage(tc.mailbox, want.ID())
				require.NoError(t, err, "GetMessage() should not error")
				assert.Equal(t, want.Mailbox(), got.Mailbox(),
					"GetMessage() returned unexpected Mailbox")
				assert.Equal(t, want.ID(), got.ID(), "GetMessage() returned unexpected ID")
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
			assert.Len(t, m, want, "Unexpected message count for mailbox %q", mailbox)

			delete(expectCounts, mailbox)

			return true
		})
		require.NoError(t, err, "VisitMailboxes() must not fail")

		// Verify all mailboxes were visited.
		assert.Empty(t, expectCounts, "Failed to visit mailboxes: %v",
			reflect.ValueOf(expectCounts).MapKeys())
	})
}

func TestStoreStubMarkSeen(t *testing.T) {
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

	// Mark second message as seen.
	seen := inputMsgs[1]
	err := ss.MarkSeen("box1", seen.ID())
	require.NoError(t, err, "MarkSeen must not fail")

	// Verify message has seen flag.
	got, err := ss.GetMessage("box1", seen.ID())
	require.NoError(t, err)
	assert.True(t, got.Seen(), "Message should have been seen")

	// Verify only one message seen.
	gotMsgs, err := ss.GetMessages("box1")
	require.NoError(t, err, "GetMessages() should not error")
	assert.Len(t, gotMsgs, len(inputMsgs), "GetMessages() returned wrong number of items")
	gotCount := 0
	for _, msg := range gotMsgs {
		if msg.Seen() {
			gotCount++
		}
	}
	assert.Equal(t, 1, gotCount, "Incorrect number of seen messages")
}

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
	require.NoError(t, err, "DeleteMessage must not fail")

	// Verify message is not in mailbox.
	messages, err := ss.GetMessages("box1")
	require.NoError(t, err)
	assert.NotContains(t, messages, deleted, "Mailbox should not contain msg %q", deleted.ID())

	// Verify message is no longer retrievable.
	got, err := ss.GetMessage("box1", deleted.ID())
	require.Error(t, err)
	assert.Nil(t, got, "Message should have been nil")

	// Verify message is in deleted list.
	assert.True(t, ss.MessageDeleted(deleted), "Message %q should be in deleted list", deleted.ID())
}

func TestStoreStubPurgeMessages(t *testing.T) {
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

	// Purge messages.
	err := ss.PurgeMessages("box1")
	require.NoError(t, err, "PurgeMessages must not fail")

	// Verify message is not in mailbox.
	messages, err := ss.GetMessages("box1")
	require.NoError(t, err)
	assert.Empty(t, messages, "Mailbox should be empty")

	// Verify messages are in deleted list.
	for _, want := range inputMsgs {
		assert.True(t, ss.MessageDeleted(want), "Message %q should be in deleted list", want.ID())
	}
}

func TestStoreStubForcedErrors(t *testing.T) {
	ss := test.NewStore()
	var err error

	// Add message to forced error mailboxes.
	msg := makeTestMessage("messageerr", "test 1")
	id1, err := ss.AddMessage(msg)
	require.NoError(t, err)
	msg = makeTestMessage("messageserr", "test 2")
	_, err = ss.AddMessage(msg)
	require.NoError(t, err)

	// Verify methods return error.
	_, err = ss.GetMessage("messageerr", id1)
	require.Error(t, err, "GetMessage()")
	assert.NotEqual(t, storage.ErrNotExist, err)

	_, err = ss.GetMessages("messageserr")
	require.Error(t, err, "GetMessages()")
	assert.NotEqual(t, storage.ErrNotExist, err)

	err = ss.MarkSeen("messageerr", id1)
	require.Error(t, err, "MarkSeen()")
	assert.NotEqual(t, storage.ErrNotExist, err)
}

func TestStoreStubNotExistErrors(t *testing.T) {
	ss := test.NewStore()
	var err error

	// Verify methods return error.
	_, err = ss.GetMessage("fake", "1")
	assert.Equal(t, storage.ErrNotExist, err, "GetMessage()")

	err = ss.MarkSeen("fake", "1")
	assert.Equal(t, storage.ErrNotExist, err, "MarkSeen()")

	err = ss.RemoveMessage("fake", "1")
	assert.Equal(t, storage.ErrNotExist, err, "RemoveMessage()")
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
