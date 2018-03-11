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
func (m *MockDataStore) GetMessage(name, id string) (StoreMessage, error) {
	args := m.Called(name, id)
	return args.Get(0).(StoreMessage), args.Error(1)
}

// GetMessages mock function
func (m *MockDataStore) GetMessages(name string) ([]StoreMessage, error) {
	args := m.Called(name)
	return args.Get(0).([]StoreMessage), args.Error(1)
}

// RemoveMessage mock function
func (m *MockDataStore) RemoveMessage(name, id string) error {
	args := m.Called(name, id)
	return args.Error(0)
}

// PurgeMessages mock function
func (m *MockDataStore) PurgeMessages(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// LockFor mock function returns a new RWMutex, never errors.
func (m *MockDataStore) LockFor(name string) (*sync.RWMutex, error) {
	return &sync.RWMutex{}, nil
}

// NewMessage temporary for #69
func (m *MockDataStore) NewMessage(mailbox string) (StoreMessage, error) {
	args := m.Called(mailbox)
	return args.Get(0).(StoreMessage), args.Error(1)
}

// VisitMailboxes accepts a function that will be called with the messages in each mailbox while it
// continues to return true.
func (m *MockDataStore) VisitMailboxes(f func([]StoreMessage) (cont bool)) error {
	return nil
}

// MockMessage is a shared mock for unit testing
type MockMessage struct {
	mock.Mock
}

// Mailbox mock function
func (m *MockMessage) Mailbox() string {
	args := m.Called()
	return args.String(0)
}

// ID mock function
func (m *MockMessage) ID() string {
	args := m.Called()
	return args.String(0)
}

// From mock function
func (m *MockMessage) From() *mail.Address {
	args := m.Called()
	return args.Get(0).(*mail.Address)
}

// To mock function
func (m *MockMessage) To() []*mail.Address {
	args := m.Called()
	return args.Get(0).([]*mail.Address)
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

// String mock function
func (m *MockMessage) String() string {
	args := m.Called()
	return args.String(0)
}
