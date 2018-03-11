package storage

import (
	"io"
	"net/mail"
	"sync"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a shared mock for unit testing
type MockDataStore struct {
	mock.Mock
}

// GetMessage mock function
func (m *MockDataStore) GetMessage(name, id string) (Message, error) {
	args := m.Called(name, id)
	return args.Get(0).(Message), args.Error(1)
}

// GetMessages mock function
func (m *MockDataStore) GetMessages(name string) ([]Message, error) {
	args := m.Called(name)
	return args.Get(0).([]Message), args.Error(1)
}

// PurgeMessages mock function
func (m *MockDataStore) PurgeMessages(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MailboxFor mock function
func (m *MockDataStore) MailboxFor(name string) (Mailbox, error) {
	args := m.Called(name)
	return args.Get(0).(Mailbox), args.Error(1)
}

// AllMailboxes mock function
func (m *MockDataStore) AllMailboxes() ([]Mailbox, error) {
	args := m.Called()
	return args.Get(0).([]Mailbox), args.Error(1)
}

// LockFor mock function returns a new RWMutex, never errors.
func (m *MockDataStore) LockFor(name string) (*sync.RWMutex, error) {
	return &sync.RWMutex{}, nil
}

// MockMailbox is a shared mock for unit testing
type MockMailbox struct {
	mock.Mock
}

// GetMessages mock function
func (m *MockMailbox) GetMessages() ([]Message, error) {
	args := m.Called()
	return args.Get(0).([]Message), args.Error(1)
}

// GetMessage mock function
func (m *MockMailbox) GetMessage(id string) (Message, error) {
	args := m.Called(id)
	return args.Get(0).(Message), args.Error(1)
}

// Purge mock function
func (m *MockMailbox) Purge() error {
	args := m.Called()
	return args.Error(0)
}

// NewMessage mock function
func (m *MockMailbox) NewMessage() (Message, error) {
	args := m.Called()
	return args.Get(0).(Message), args.Error(1)
}

// String mock function
func (m *MockMailbox) String() string {
	args := m.Called()
	return args.String(0)
}

// MockMessage is a shared mock for unit testing
type MockMessage struct {
	mock.Mock
}

// ID mock function
func (m *MockMessage) ID() string {
	args := m.Called()
	return args.String(0)
}

// From mock function
func (m *MockMessage) From() string {
	args := m.Called()
	return args.String(0)
}

// To mock function
func (m *MockMessage) To() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// Date mock function
func (m *MockMessage) Date() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// Subject mock function
func (m *MockMessage) Subject() string {
	args := m.Called()
	return args.String(0)
}

// ReadHeader mock function
func (m *MockMessage) ReadHeader() (msg *mail.Message, err error) {
	args := m.Called()
	return args.Get(0).(*mail.Message), args.Error(1)
}

// ReadBody mock function
func (m *MockMessage) ReadBody() (body *enmime.Envelope, err error) {
	args := m.Called()
	return args.Get(0).(*enmime.Envelope), args.Error(1)
}

// ReadRaw mock function
func (m *MockMessage) ReadRaw() (raw *string, err error) {
	args := m.Called()
	return args.Get(0).(*string), args.Error(1)
}

// RawReader mock function
func (m *MockMessage) RawReader() (reader io.ReadCloser, err error) {
	args := m.Called()
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Size mock function
func (m *MockMessage) Size() int64 {
	args := m.Called()
	return int64(args.Int(0))
}

// Append mock function
func (m *MockMessage) Append(data []byte) error {
	// []byte arg seems to mess up testify/mock
	return nil
}

// Close mock function
func (m *MockMessage) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Delete mock function
func (m *MockMessage) Delete() error {
	args := m.Called()
	return args.Error(0)
}

// String mock function
func (m *MockMessage) String() string {
	args := m.Called()
	return args.String(0)
}
