// Package client provides a basic REST client for Inbucket
package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/inbucket/inbucket/v3/pkg/rest/model"
)

// Client accesses the Inbucket REST API v1
type Client struct {
	restClient
}

// New creates a new v1 REST API client given the base URL of an Inbucket server, ex:
// "http://localhost:9000"
func New(baseURL string, opts ...Option) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	mergedOpts := getDefaultOptions()
	for _, opt := range opts {
		opt.apply(mergedOpts)
	}

	c := &Client{
		restClient{
			client: &http.Client{
				Timeout:   mergedOpts.timeout,
				Transport: mergedOpts.transport,
			},
			baseURL: parsedURL,
		},
	}
	return c, nil
}

// ListMailbox returns a list of messages for the requested mailbox
func (c *Client) ListMailbox(name string) ([]*MessageHeader, error) {
	return c.ListMailboxWithContext(context.Background(), name)
}

// ListMailboxWithContext returns a list of messages for the requested mailbox
func (c *Client) ListMailboxWithContext(ctx context.Context, name string) ([]*MessageHeader, error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	headers := make([]*MessageHeader, 0, 32)

	err := c.doJSON(ctx, "GET", uri, &headers)
	if err != nil {
		return nil, err
	}

	// Add Client ref to each MessageHeader for convenience funcs.
	for _, h := range headers {
		h.client = c
	}

	return headers, nil
}

// GetMessage returns the message details given a mailbox name and message ID.
func (c *Client) GetMessage(name, id string) (message *Message, err error) {
	return c.GetMessageWithContext(context.Background(), name, id)
}

// GetMessageWithContext returns the message details given a mailbox name and message ID.
func (c *Client) GetMessageWithContext(ctx context.Context, name, id string) (*Message, error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	var message Message

	err := c.doJSON(ctx, "GET", uri, &message)
	if err != nil {
		return nil, err
	}

	message.client = c
	return &message, nil
}

// MarkSeen marks the specified message as having been read.
func (c *Client) MarkSeen(name, id string) error {
	return c.MarkSeenWithContext(context.Background(), name, id)
}

// MarkSeenWithContext marks the specified message as having been read.
func (c *Client) MarkSeenWithContext(ctx context.Context, name, id string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	err := c.doJSON(ctx, "PATCH", uri, nil)
	if err != nil {
		return err
	}
	return nil
}

// GetMessageSource returns the message source given a mailbox name and message ID.
func (c *Client) GetMessageSource(name, id string) (*bytes.Buffer, error) {
	return c.GetMessageSourceWithContext(context.Background(), name, id)
}

// GetMessageSourceWithContext returns the message source given a mailbox name and message ID.
func (c *Client) GetMessageSourceWithContext(ctx context.Context, name, id string) (*bytes.Buffer, error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id + "/source"
	resp, err := c.do(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil,
			fmt.Errorf("unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	return buf, err
}

// DeleteMessage deletes a single message given the mailbox name and message ID.
func (c *Client) DeleteMessage(name, id string) error {
	return c.DeleteMessageWithContext(context.Background(), name, id)
}

// DeleteMessageWithContext deletes a single message given the mailbox name and message ID.
func (c *Client) DeleteMessageWithContext(ctx context.Context, name, id string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	resp, err := c.do(ctx, "DELETE", uri, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// PurgeMailbox deletes all messages in the given mailbox
func (c *Client) PurgeMailbox(name string) error {
	return c.PurgeMailboxWithContext(context.Background(), name)
}

// PurgeMailboxWithContext deletes all messages in the given mailbox
func (c *Client) PurgeMailboxWithContext(ctx context.Context, name string) error {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	resp, err := c.do(ctx, "DELETE", uri, nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
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
	return h.GetMessageWithContext(context.Background())
}

// GetMessageWithContext returns this message with content
func (h *MessageHeader) GetMessageWithContext(ctx context.Context) (message *Message, err error) {
	return h.client.GetMessageWithContext(ctx, h.Mailbox, h.ID)
}

// GetSource returns the source for this message
func (h *MessageHeader) GetSource() (*bytes.Buffer, error) {
	return h.GetSourceWithContext(context.Background())
}

// GetSourceWithContext returns the source for this message
func (h *MessageHeader) GetSourceWithContext(ctx context.Context) (*bytes.Buffer, error) {
	return h.client.GetMessageSourceWithContext(ctx, h.Mailbox, h.ID)
}

// Delete deletes this message from the mailbox
func (h *MessageHeader) Delete() error {
	return h.DeleteWithContext(context.Background())
}

// DeleteWithContext deletes this message from the mailbox
func (h *MessageHeader) DeleteWithContext(ctx context.Context) error {
	return h.client.DeleteMessageWithContext(ctx, h.Mailbox, h.ID)
}

// Message represents an Inbucket message including content
type Message struct {
	*model.JSONMessageV1
	client *Client
}

// GetSource returns the source for this message
func (m *Message) GetSource() (*bytes.Buffer, error) {
	return m.GetSourceWithContext(context.Background())
}

// GetSourceWithContext returns the source for this message
func (m *Message) GetSourceWithContext(ctx context.Context) (*bytes.Buffer, error) {
	return m.client.GetMessageSourceWithContext(ctx, m.Mailbox, m.ID)
}

// Delete deletes this message from the mailbox
func (m *Message) Delete() error {
	return m.DeleteWithContext(context.Background())
}

// DeleteWithContext deletes this message from the mailbox
func (m *Message) DeleteWithContext(ctx context.Context) error {
	return m.client.DeleteMessageWithContext(ctx, m.Mailbox, m.ID)
}
