// Package client provides a basic REST client for Inbucket
package client

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/inbucket/inbucket/pkg/rest/model"
)

// Client accesses the Inbucket REST API v1
type Client struct {
	restClient
}

// New creates a new v1 REST API client given the base URL of an Inbucket server, ex:
// "http://localhost:9000"
func New(baseURL string) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
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
func (c *Client) ListMailbox(name string) (headers []*MessageHeader, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	err = c.doJSON("GET", uri, &headers)
	if err != nil {
		return nil, err
	}
	for _, h := range headers {
		h.client = c
	}
	return
}

// GetMessage returns the message details given a mailbox name and message ID.
func (c *Client) GetMessage(name, id string) (message *Message, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	err = c.doJSON("GET", uri, &message)
	if err != nil {
		return nil, err
	}
	message.client = c
	return
}

// MarkSeen marks the specified message as having been read.
func (c *Client) MarkSeen(name, id string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	err := c.doJSON("PATCH", uri, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetMessageSource returns the message source given a mailbox name and message ID.
func (c *Client) GetMessageSource(name, id string) (*bytes.Buffer, error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id + "/source"
	resp, err := c.do("GET", uri, nil)
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
func (c *Client) DeleteMessage(name, id string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	resp, err := c.do("DELETE", uri, nil)
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
func (c *Client) PurgeMailbox(name string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	resp, err := c.do("DELETE", uri, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}
	return nil
}

// MessageHeader represents an Inbucket message sans content
type MessageHeader struct {
	*model.JSONMessageHeaderV1
	client *Client
}

// GetMessage returns this message with content
func (h *MessageHeader) GetMessage() (message *Message, err error) {
	return h.client.GetMessage(h.Mailbox, h.ID)
}

// GetSource returns the source for this message
func (h *MessageHeader) GetSource() (*bytes.Buffer, error) {
	return h.client.GetMessageSource(h.Mailbox, h.ID)
}

// Delete deletes this message from the mailbox
func (h *MessageHeader) Delete() error {
	return h.client.DeleteMessage(h.Mailbox, h.ID)
}

// Message represents an Inbucket message including content
type Message struct {
	*model.JSONMessageV1
	client *Client
}

// GetSource returns the source for this message
func (m *Message) GetSource() (*bytes.Buffer, error) {
	return m.client.GetMessageSource(m.Mailbox, m.ID)
}

// Delete deletes this message from the mailbox
func (m *Message) Delete() error {
	return m.client.DeleteMessage(m.Mailbox, m.ID)
}
