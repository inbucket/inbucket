package rest

import (
	"io"
	"net/mail"
	"time"

	"github.com/jhillyerd/enmime"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/stretchr/testify/mock"
)

// Mock DataStore object
type MockDataStore struct {
	mock.Mock
}

func (m *MockDataStore) MailboxFor(name string) (smtpd.Mailbox, error) {
	args := m.Called(name)
	return args.Get(0).(smtpd.Mailbox), args.Error(1)
}

func (m *MockDataStore) AllMailboxes() ([]smtpd.Mailbox, error) {
	args := m.Called()
	return args.Get(0).([]smtpd.Mailbox), args.Error(1)
}

// Mock Mailbox object
type MockMailbox struct {
	mock.Mock
}

func (m *MockMailbox) GetMessages() ([]smtpd.Message, error) {
	args := m.Called()
	return args.Get(0).([]smtpd.Message), args.Error(1)
}

func (m *MockMailbox) GetMessage(id string) (smtpd.Message, error) {
	args := m.Called(id)
	return args.Get(0).(smtpd.Message), args.Error(1)
}

func (m *MockMailbox) Purge() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMailbox) NewMessage() (smtpd.Message, error) {
	args := m.Called()
	return args.Get(0).(smtpd.Message), args.Error(1)
}

func (m *MockMailbox) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMailbox) String() string {
	args := m.Called()
	return args.String(0)
}

// Mock Message object
type MockMessage struct {
	mock.Mock
}

func (m *MockMessage) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) From() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMessage) To() []string {
	args := m.Called()
	return args.Get(0).([]string)
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

func (m *MockMessage) ReadBody() (body *enmime.Envelope, err error) {
	args := m.Called()
	return args.Get(0).(*enmime.Envelope), args.Error(1)
}

func (m *MockMessage) ReadRaw() (raw *string, err error) {
	args := m.Called()
	return args.Get(0).(*string), args.Error(1)
}

func (m *MockMessage) RawReader() (reader io.ReadCloser, err error) {
	args := m.Called()
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockMessage) Size() int64 {
	args := m.Called()
	return int64(args.Int(0))
}

func (m *MockMessage) Append(data []byte) error {
	// []byte arg seems to mess up testify/mock
	return nil
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
