package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// do performs an HTTP request with this client and returns the response.
func (c *restClient) do(method, uri string, body []byte) (*http.Response, error) {
	rel, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	url := c.baseURL.ResolveReference(rel)
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url.String(), r)
	if err != nil {
		return nil, fmt.Errorf("%s for %q: %v", method, url, err)
	}
	return c.client.Do(req)
}

// doJSON performs an HTTP request with this client and marshalls the JSON response into v.
func (c *restClient) doJSON(method string, uri string, v interface{}) error {
	resp, err := c.do(method, uri, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusOK {
		if v == nil {
			return nil
		}
		// Decode response body
		return json.NewDecoder(resp.Body).Decode(v)
	}

	return fmt.Errorf("%s for %q, unexpected %v: %s", method, uri, resp.StatusCode, resp.Status)
}

// doJSONBody performs an HTTP request with this client and marshalls the JSON response into v.
func (c *restClient) doJSONBody(method string, uri string, body []byte, v interface{}) error {
	resp, err := c.do(method, uri, body)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode == http.StatusOK {
		if v == nil {
			return nil
		}
		// Decode response body
		return json.NewDecoder(resp.Body).Decode(v)
	}

	return fmt.Errorf("%s for %q, unexpected %v: %s", method, uri, resp.StatusCode, resp.Status)
}
