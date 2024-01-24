package client

import (
	"net/http"
	"time"
)

// ClientOptions is a struct that holds the options for the client
type clientOptions struct {
	transport http.RoundTripper
	timeout   time.Duration
}

type ClientOption interface {
	apply(*clientOptions)
}

// getDefaultClientOptions returns the default options for the client
func getDefaultClientOptions() *clientOptions {
	return &clientOptions{
		timeout: 30 * time.Second,
	}
}

type transportOption struct {
	transport http.RoundTripper
}

func (t transportOption) apply(opts *clientOptions) {
	opts.transport = t.transport
}

// WithTransport returns a function that sets the transport object
func WithClientOptsTransport(transport http.RoundTripper) ClientOption {
	return transportOption{transport}
}
