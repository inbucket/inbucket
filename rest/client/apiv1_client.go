package client

import (
	"net/http"
	"net/url"

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
			client:  &http.Client{},
			baseURL: parsedURL,
		},
	}
	return c, nil
}

// ListMailbox returns a list of messages for the requested mailbox
func (c *ClientV1) ListMailbox(name string) (headers []*model.JSONMessageHeaderV1, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name)
	err = c.doGet(uri, &headers)
	return
}

// GetMessage returns the message details given a mailbox name and message ID.
func (c *ClientV1) GetMessage(name, id string) (message *model.JSONMessageV1, err error) {
	uri := "/api/v1/mailbox/" + url.QueryEscape(name) + "/" + id
	err = c.doGet(uri, &message)
	return
}
