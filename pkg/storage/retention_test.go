package storage

import (
	"fmt"
	"testing"
	"time"
)

func TestDoRetentionScan(t *testing.T) {
	// Create mock objects
	mds := &MockDataStore{}

	mb1 := &MockMailbox{}
	mb2 := &MockMailbox{}
	mb3 := &MockMailbox{}

	// Mockup some different aged messages (num is in hours)
	new1 := mockMessage(0)
	new2 := mockMessage(1)
	new3 := mockMessage(2)
	old1 := mockMessage(4)
	old2 := mockMessage(12)
	old3 := mockMessage(24)

	// First it should ask for all mailboxes
	mds.On("AllMailboxes").Return([]Mailbox{mb1, mb2, mb3}, nil)

	// Then for all messages on each box
	mb1.On("GetMessages").Return([]Message{new1, old1, old2}, nil)
	mb2.On("GetMessages").Return([]Message{old3, new2}, nil)
	mb3.On("GetMessages").Return([]Message{new3}, nil)

	// Test 4 hour retention
	rs := &RetentionScanner{
		ds:              mds,
		retentionPeriod: 4*time.Hour - time.Minute,
		retentionSleep:  0,
	}
	if err := rs.doScan(); err != nil {
		t.Error(err)
	}

	// Check our assertions
	mds.AssertExpectations(t)
	mb1.AssertExpectations(t)
	mb2.AssertExpectations(t)
	mb3.AssertExpectations(t)

	// Delete should not have been called on new messages
	new1.AssertNotCalled(t, "Delete")
	new2.AssertNotCalled(t, "Delete")
	new3.AssertNotCalled(t, "Delete")

	// Delete should have been called once on old messages
	old1.AssertNumberOfCalls(t, "Delete", 1)
	old2.AssertNumberOfCalls(t, "Delete", 1)
	old3.AssertNumberOfCalls(t, "Delete", 1)
}

// Make a MockMessage of a specific age
func mockMessage(ageHours int) *MockMessage {
	msg := &MockMessage{}
	msg.On("ID").Return(fmt.Sprintf("MSG[age=%vh]", ageHours))
	msg.On("Date").Return(time.Now().Add(time.Duration(ageHours*-1) * time.Hour))
	msg.On("Delete").Return(nil)
	return msg
}
