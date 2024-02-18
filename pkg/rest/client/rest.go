package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// httpClient allows http.Client to be mocked for tests
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Generic REST restClient
type restClient struct {
	client  httpClient
	baseURL *url.URL
}

// do performs an HTTP request with this client and returns the response.
func (c *restClient) do(ctx context.Context, method, uri string, body []byte) (*http.Response, error) {
	url := c.baseURL.JoinPath(uri)
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), r)
	if err != nil {
		return nil, fmt.Errorf("%s for %q: %v", method, url, err)
	}

	return c.client.Do(req)
}

// doJSON performs an HTTP request with this client and marshalls the JSON response into v.
func (c *restClient) doJSON(ctx context.Context, method string, uri string, v interface{}) error {
	resp, err := c.do(ctx, method, uri, nil)
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
