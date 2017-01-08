package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// httpClient allows http.Client to be mocked for tests
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Generic REST restClient
type restClient struct {
	client  httpClient
	baseURL *url.URL
}

// do performs an HTTP request with this client and returns the response
func (c *restClient) do(method, uri string) (*http.Response, error) {
	rel, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	url := c.baseURL.ResolveReference(rel)

	// Build the request
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Send the request
	return c.client.Do(req)
}

// doGet performs a GET request with this client and marshalls the JSON response into v
func (c *restClient) doGet(uri string, v interface{}) error {
	resp, err := c.do("GET", uri)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusOK {
		// Decode response body
		return json.NewDecoder(resp.Body).Decode(v)
	}

	return fmt.Errorf("Unexpected HTTP response status %v: %s", resp.StatusCode, resp.Status)
}
