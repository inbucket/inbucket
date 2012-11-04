package smtpd

import (
	"fmt"
	"github.com/jhillyerd/go.enmime"
	"github.com/stretchrcom/testify/mock"
	"net/mail"
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
	doRetentionScan(mds, 4*time.Hour, 0)

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
	msg.On("Id").Return(fmt.Sprintf("MSG[age=%vh]", ageHours))
	msg.On("Date").Return(time.Now().Add(time.Duration(ageHours*-1) * time.Hour))
	msg.On("Delete").Return(nil)
	return msg
}

// Mock DataStore object
type MockDataStore struct {
	mock.Mock
}

func (m *MockDataStore) MailboxFor(name string) (Mailbox, error) {
	return nil, nil
}

func (m *MockDataStore) AllMailboxes() ([]Mailbox, error) {
	args := m.Called()
	return args.Get(0).([]Mailbox), args.Error(1)
}

// Mock Mailbox object
type MockMailbox struct {
	mock.Mock
}

func (m *MockMailbox) GetMessages() ([]Message, error) {
	args := m.Called()
	return args.Get(0).([]Message), args.Error(1)
}

func (m *MockMailbox) GetMessage(id string) (Message, error) {
	args := m.Called(id)
	return args.Get(0).(Message), args.Error(1)
}

func (m *MockMailbox) NewMessage() Message {
	args := m.Called()
	return args.Get(0).(Message)
}

func (m *MockMailbox) String() string {
	args := m.Called()
	return args.String(0)
}

// Mock Message object
type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) Id() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) From() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) Date() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockMessage) Subject() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) ReadHeader() (msg *mail.Message, err error) {
	args := m.Called()
	return args.Get(0).(*mail.Message), args.Error(1)
}

func (m *MockMessage) ReadBody() (msg *mail.Message, body *enmime.MIMEBody, err error) {
	args := m.Called()
	return args.Get(0).(*mail.Message), args.Get(1).(*enmime.MIMEBody), args.Error(2)
}

func (m *MockMessage) ReadRaw() (raw *string, err error) {
	args := m.Called()
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockMessage) Append(data []byte) error {
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockMessage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMessage) Delete() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMessage) String() string {
	args := m.Called()
	return args.String(0)
}
