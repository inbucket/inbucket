package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jhillyerd/go.enmime"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/stretchr/testify/mock"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"os"
	"testing"
	"time"
)

type OutputJsonHeader struct {
	Mailbox, Id, From, Subject, Date string
	Size                             int
}

type InputMessageData struct {
	Mailbox, Id, From, Subject string
	Date                       time.Time
	Size                       int
}

func (d *InputMessageData) MockMessage() *MockMessage {
	msg := &MockMessage{}
	msg.On("Id").Return(d.Id)
	msg.On("From").Return(d.From)
	msg.On("Subject").Return(d.Subject)
	msg.On("Date").Return(d.Date)
	msg.On("Size").Return(d.Size)
	return msg
}

func (d *InputMessageData) CompareToJson(j *OutputJsonHeader) (errors []string) {
	if d.Mailbox != j.Mailbox {
		errors = append(errors, fmt.Sprintf("Expected JSON.Mailbox=%q, got %q", d.Mailbox,
			j.Mailbox))
	}
	if d.Id != j.Id {
		errors = append(errors, fmt.Sprintf("Expected JSON.Id=%q, got %q", d.Id,
			j.Id))
	}
	if d.From != j.From {
		errors = append(errors, fmt.Sprintf("Expected JSON.From=%q, got %q", d.From,
			j.From))
	}
	if d.Subject != j.Subject {
		errors = append(errors, fmt.Sprintf("Expected JSON.Subject=%q, got %q", d.Subject,
			j.Subject))
	}
	exDate := d.Date.Format("2006-01-02T15:04:05.999999999-07:00")
	if exDate != j.Date {
		errors = append(errors, fmt.Sprintf("Expected JSON.Date=%q, got %q", exDate,
			j.Date))
	}
	if d.Size != j.Size {
		errors = append(errors, fmt.Sprintf("Expected JSON.Size=%v, got %v", d.Size,
			j.Size))
	}

	return errors
}

func TestRestMailboxList(t *testing.T) {
	// Setup
	ds := &MockDataStore{}
	logbuf := setupWebServer(ds)

	// Test invalid mailbox name
	w, err := testRestGet("http://localhost/mailbox/foo@bar")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test empty mailbox
	emptybox := &MockMailbox{}
	ds.On("MailboxFor", "empty").Return(emptybox, nil)
	emptybox.On("GetMessages").Return([]smtpd.Message{}, nil)

	w, err = testRestGet("http://localhost/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test MailboxFor error
	ds.On("MailboxFor", "error").Return(&MockMailbox{}, fmt.Errorf("Internal error"))
	w, err = testRestGet("http://localhost/mailbox/error")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		io.Copy(os.Stderr, logbuf)
	}

	// Test MailboxFor error
	error2box := &MockMailbox{}
	ds.On("MailboxFor", "error2").Return(error2box, nil)
	error2box.On("GetMessages").Return([]smtpd.Message{}, fmt.Errorf("Internal error 2"))

	w, err = testRestGet("http://localhost/mailbox/error2")
	expectCode = 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test JSON message headers
	data1 := &InputMessageData{
		Mailbox: "good",
		Id:      "0001",
		From:    "from1",
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	data2 := &InputMessageData{
		Mailbox: "good",
		Id:      "0002",
		From:    "from2",
		Subject: "subject 2",
		Date:    time.Date(2012, 7, 1, 10, 11, 12, 253, time.FixedZone("PDT", -700)),
	}
	goodbox := &MockMailbox{}
	ds.On("MailboxFor", "good").Return(goodbox, nil)
	msg1 := data1.MockMessage()
	msg2 := data2.MockMessage()
	goodbox.On("GetMessages").Return([]smtpd.Message{msg1, msg2}, nil)

	// Check return code
	w, err = testRestGet("http://localhost/mailbox/good")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	dec := json.NewDecoder(w.Body)
	var result []OutputJsonHeader
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %v", len(result))
	}
	if errors := data1.CompareToJson(&result[0]); len(errors) > 0 {
		for _, e := range errors {
			t.Error(e)
		}
	}
	if errors := data2.CompareToJson(&result[1]); len(errors) > 0 {
		for _, e := range errors {
			t.Error(e)
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		io.Copy(os.Stderr, logbuf)
	}
}

func testRestGet(url string) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	Router.ServeHTTP(w, req)
	return w, nil
}

func setupWebServer(ds smtpd.DataStore) *bytes.Buffer {
	// Capture log output
	buf := new(bytes.Buffer)
	log.SetOutput(buf)

	cfg := config.WebConfig{
		TemplateDir: "../themes/integral/templates",
		PublicDir:   "../themes/integral/public",
	}
	Initialize(cfg, ds)

	return buf
}

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

func (m *MockMessage) ReadBody() (body *enmime.MIMEBody, err error) {
	args := m.Called()
	return args.Get(0).(*enmime.MIMEBody), args.Error(1)
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
