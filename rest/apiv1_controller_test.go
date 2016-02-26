package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"os"
	"testing"
	"time"

	"github.com/jhillyerd/go.enmime"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/stretchr/testify/mock"
)

const (
	baseURL = "http://localhost/api/v1"

	// JSON map keys
	mailboxKey = "Mailbox"
	idKey      = "Id"
	fromKey    = "From"
	subjectKey = "Subject"
	dateKey    = "Date"
	sizeKey    = "Size"
	headerKey  = "Header"
	bodyKey    = "Body"
	textKey    = "Text"
	htmlKey    = "Html"
)

type InputMessageData struct {
	Mailbox, ID, From, Subject string
	Date                       time.Time
	Size                       int
	Header                     mail.Header
	HTML, Text                 string
}

func (d *InputMessageData) MockMessage() *MockMessage {
	msg := &MockMessage{}
	msg.On("ID").Return(d.ID)
	msg.On("From").Return(d.From)
	msg.On("Subject").Return(d.Subject)
	msg.On("Date").Return(d.Date)
	msg.On("Size").Return(d.Size)
	gomsg := &mail.Message{
		Header: d.Header,
	}
	msg.On("ReadHeader").Return(gomsg, nil)
	body := &enmime.MIMEBody{
		Text: d.Text,
		Html: d.HTML,
	}
	msg.On("ReadBody").Return(body, nil)
	return msg
}

// isJSONStringEqual is a utility function to return a nicely formatted message when
// comparing a string to a value received from a JSON map.
func isJSONStringEqual(key, expected string, received interface{}) (message string, ok bool) {
	if value, ok := received.(string); ok {
		if expected == value {
			return "", true
		}
		return fmt.Sprintf("Expected value of key %v to be %q, got %q", key, expected, value), false
	}
	return fmt.Sprintf("Expected value of key %v to be a string, got %T", key, received), false
}

// isJSONNumberEqual is a utility function to return a nicely formatted message when
// comparing an float64 to a value received from a JSON map.
func isJSONNumberEqual(key string, expected float64, received interface{}) (message string, ok bool) {
	if value, ok := received.(float64); ok {
		if expected == value {
			return "", true
		}
		return fmt.Sprintf("Expected %v to be %v, got %v", key, expected, value), false
	}
	return fmt.Sprintf("Expected %v to be a string, got %T", key, received), false
}

// CompareToJSONHeaderMap compares InputMessageData to a header map decoded from JSON,
// returning a list of things that did not match.
func (d *InputMessageData) CompareToJSONHeaderMap(json interface{}) (errors []string) {
	if m, ok := json.(map[string]interface{}); ok {
		if msg, ok := isJSONStringEqual(mailboxKey, d.Mailbox, m[mailboxKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONStringEqual(idKey, d.ID, m[idKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONStringEqual(fromKey, d.From, m[fromKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONStringEqual(subjectKey, d.Subject, m[subjectKey]); !ok {
			errors = append(errors, msg)
		}
		exDate := d.Date.Format("2006-01-02T15:04:05.999999999-07:00")
		if msg, ok := isJSONStringEqual(dateKey, exDate, m[dateKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONNumberEqual(sizeKey, float64(d.Size), m[sizeKey]); !ok {
			errors = append(errors, msg)
		}
		return errors
	}
	panic(fmt.Sprintf("Expected map[string]interface{} in json, got %T", json))
}

// CompareToJSONMessageMap compares InputMessageData to a message map decoded from JSON,
// returning a list of things that did not match.
func (d *InputMessageData) CompareToJSONMessageMap(json interface{}) (errors []string) {
	// We need to check the same values as header first
	errors = d.CompareToJSONHeaderMap(json)

	if m, ok := json.(map[string]interface{}); ok {
		// Get nested body map
		if body := m[bodyKey].(map[string]interface{}); ok {
			if msg, ok := isJSONStringEqual(textKey, d.Text, body[textKey]); !ok {
				errors = append(errors, msg)
			}
			if msg, ok := isJSONStringEqual(htmlKey, d.HTML, body[htmlKey]); !ok {
				errors = append(errors, msg)
			}
		} else {
			panic(fmt.Sprintf("Expected map[string]interface{} in json key %q, got %T",
				bodyKey, m[bodyKey]))
		}
		exDate := d.Date.Format("2006-01-02T15:04:05.999999999-07:00")
		if msg, ok := isJSONStringEqual(dateKey, exDate, m[dateKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONNumberEqual(sizeKey, float64(d.Size), m[sizeKey]); !ok {
			errors = append(errors, msg)
		}

		// Get nested header map
		if header := m[headerKey].(map[string]interface{}); ok {
			// Loop over input (expected) header names
			for name, keyInputHeaders := range d.Header {
				// Make sure expected header name exists in received JSON
				if keyOutputVals, ok := header[name]; ok {
					if keyOutputHeaders, ok := keyOutputVals.([]interface{}); ok {
						// Loop over input (expected) header values
						for _, inputHeader := range keyInputHeaders {
							hasValue := false
							// Look for expected value in received headers
							for _, outputHeader := range keyOutputHeaders {
								if inputHeader == outputHeader {
									hasValue = true
									break
								}
							}
							if !hasValue {
								errors = append(errors, fmt.Sprintf(
									"JSON %v[%q] missing value %q", headerKey, name, inputHeader))
							}
						}
					} else {
						// keyOutputValues was not a slice of interface{}
						panic(fmt.Sprintf("Expected []interface{} in %v[%q], got %T", headerKey,
							name, keyOutputVals))
					}
				} else {
					errors = append(errors, fmt.Sprintf("JSON %v missing key %q", headerKey, name))
				}
			}
		}
	} else {
		panic(fmt.Sprintf("Expected map[string]interface{} in json, got %T", json))
	}

	return errors
}

func TestRestMailboxList(t *testing.T) {
	// Setup
	ds := &MockDataStore{}
	logbuf := setupWebServer(ds)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar")
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

	w, err = testRestGet(baseURL + "/mailbox/empty")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test MailboxFor error
	ds.On("MailboxFor", "error").Return(&MockMailbox{}, fmt.Errorf("Internal error"))
	w, err = testRestGet(baseURL + "/mailbox/error")
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
		_, _ = io.Copy(os.Stderr, logbuf)
	}

	// Test MailboxFor error
	error2box := &MockMailbox{}
	ds.On("MailboxFor", "error2").Return(error2box, nil)
	error2box.On("GetMessages").Return([]smtpd.Message{}, fmt.Errorf("Internal error 2"))

	w, err = testRestGet(baseURL + "/mailbox/error2")
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
		ID:      "0001",
		From:    "from1",
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
	}
	data2 := &InputMessageData{
		Mailbox: "good",
		ID:      "0002",
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
	w, err = testRestGet(baseURL + "/mailbox/good")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	dec := json.NewDecoder(w.Body)
	var result []interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %v", len(result))
	}
	if errors := data1.CompareToJSONHeaderMap(result[0]); len(errors) > 0 {
		t.Logf("%v", result[0])
		for _, e := range errors {
			t.Error(e)
		}
	}
	if errors := data2.CompareToJSONHeaderMap(result[1]); len(errors) > 0 {
		t.Logf("%v", result[1])
		for _, e := range errors {
			t.Error(e)
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

func TestRestMessage(t *testing.T) {
	// Setup
	ds := &MockDataStore{}
	logbuf := setupWebServer(ds)

	// Test invalid mailbox name
	w, err := testRestGet(baseURL + "/mailbox/foo@bar/0001")
	expectCode := 500
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test requesting a message that does not exist
	emptybox := &MockMailbox{}
	ds.On("MailboxFor", "empty").Return(emptybox, nil)
	emptybox.On("GetMessage", "0001").Return(&MockMessage{}, smtpd.ErrNotExist)

	w, err = testRestGet(baseURL + "/mailbox/empty/0001")
	expectCode = 404
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Errorf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Test MailboxFor error
	ds.On("MailboxFor", "error").Return(&MockMailbox{}, fmt.Errorf("Internal error"))
	w, err = testRestGet(baseURL + "/mailbox/error/0001")
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
		_, _ = io.Copy(os.Stderr, logbuf)
	}

	// Test GetMessage error
	error2box := &MockMailbox{}
	ds.On("MailboxFor", "error2").Return(error2box, nil)
	error2box.On("GetMessage", "0001").Return(&MockMessage{}, fmt.Errorf("Internal error 2"))

	w, err = testRestGet(baseURL + "/mailbox/error2/0001")
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
		ID:      "0001",
		From:    "from1",
		Subject: "subject 1",
		Date:    time.Date(2012, 2, 1, 10, 11, 12, 253, time.FixedZone("PST", -800)),
		Header: mail.Header{
			"To":   []string{"fred@fish.com", "keyword@nsa.gov"},
			"From": []string{"noreply@inbucket.org"},
		},
		Text: "This is some text",
		HTML: "This is some HTML",
	}
	goodbox := &MockMailbox{}
	ds.On("MailboxFor", "good").Return(goodbox, nil)
	msg1 := data1.MockMessage()
	goodbox.On("GetMessage", "0001").Return(msg1, nil)

	// Check return code
	w, err = testRestGet(baseURL + "/mailbox/good/0001")
	expectCode = 200
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != expectCode {
		t.Fatalf("Expected code %v, got %v", expectCode, w.Code)
	}

	// Check JSON
	dec := json.NewDecoder(w.Body)
	var result map[string]interface{}
	if err := dec.Decode(&result); err != nil {
		t.Errorf("Failed to decode JSON: %v", err)
	}

	if errors := data1.CompareToJSONMessageMap(result); len(errors) > 0 {
		t.Logf("%v", result)
		for _, e := range errors {
			t.Error(e)
		}
	}

	if t.Failed() {
		// Wait for handler to finish logging
		time.Sleep(2 * time.Second)
		// Dump buffered log data if there was a failure
		_, _ = io.Copy(os.Stderr, logbuf)
	}
}

func testRestGet(url string) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	httpd.Router.ServeHTTP(w, req)
	return w, nil
}

func setupWebServer(ds smtpd.DataStore) *bytes.Buffer {
	// Capture log output
	buf := new(bytes.Buffer)
	log.SetOutput(buf)

	// Have to reset default mux to prevent duplicate routes
	http.DefaultServeMux = http.NewServeMux()
	cfg := config.WebConfig{
		TemplateDir: "../themes/integral/templates",
		PublicDir:   "../themes/integral/public",
	}
	httpd.Initialize(cfg, ds)
	SetupRoutes(httpd.Router)

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

func (m *MockMessage) ID() string {
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
