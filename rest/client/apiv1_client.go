package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/jhillyerd/inbucket/rest/model"
)

// ClientV1 accesses the Inbucket REST API v1
type ClientV1 struct {
	restClient
}

// NewV1 creates a new v1 REST API client given the base URL of an Inbucket server, ex:
// "http://localhost:9000"
func NewV1(baseURL string) (*ClientV1, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	c := &ClientV1{
		restClient{
			client: &http.Client{
				Timeout: 30 * time.Second,
			},
			baseURL: parsedURL,
		},
	}
	return c, nil
}

// ListMailbox returns a list of messages for the requested mailbox
func (c *ClientV1) ListMailbox(name string) (headers []*model.JSONMessageHeaderV1, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	err = c.doJSON("GET", uri, &headers)
	return
}

// GetMessage returns the message details given a mailbox name and message ID.
func (c *ClientV1) GetMessage(name, id string) (message *model.JSONMessageV1, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	err = c.doJSON("GET", uri, &message)
	return
}

// GetMessageSource returns the message source given a mailbox name and message ID.
func (c *ClientV1) GetMessageSource(name, id string) (*bytes.Buffer, error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id + "/source"
	resp, err := c.do("GET", uri)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil,
			fmt.Errorf("Unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	return buf, err
}

// DeleteMessage deletes a single message given the mailbox name and message ID.
func (c *ClientV1) DeleteMessage(name, id string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	resp, err := c.do("DELETE", uri)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}
	return nil
}

// PurgeMailbox deletes all messages in the given mailbox
func (c *ClientV1) PurgeMailbox(name string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	resp, err := c.do("DELETE", uri)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}
	return nil
}
