package rest

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"time"

	"github.com/jhillyerd/go.enmime"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/smtpd"
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
		if m[bodyKey] != nil {
			if body, ok := m[bodyKey].(map[string]interface{}); ok {
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
		} else {
			errors = append(errors, fmt.Sprintf("Expected body in JSON %q but it was nil", bodyKey))
		}
		exDate := d.Date.Format("2006-01-02T15:04:05.999999999-07:00")
		if msg, ok := isJSONStringEqual(dateKey, exDate, m[dateKey]); !ok {
			errors = append(errors, msg)
		}
		if msg, ok := isJSONNumberEqual(sizeKey, float64(d.Size), m[sizeKey]); !ok {
			errors = append(errors, msg)
		}

		// Get nested header map
		if m[headerKey] != nil {
			if header, ok := m[headerKey].(map[string]interface{}); ok {
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
			errors = append(errors, fmt.Sprintf("Expected header in JSON %q but it was nil", headerKey))
		}
	} else {
		panic(fmt.Sprintf("Expected map[string]interface{} in json, got %T", json))
	}

	return errors
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
