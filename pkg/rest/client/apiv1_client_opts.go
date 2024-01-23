package client

import (
	"net/http"
	"time"
)

// ClientOptions is a struct that holds the options for the client
type ClientOptions struct {
	transport *http.Transport
	timeout   time.Duration
}

// getDefaultClientOptions returns the default options for the client
func getDefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		timeout: 30 * time.Second,
	}
}

// WithTransport returns a function that sets the transport object
func WithTransport(transport *http.Transport) func(*ClientOptions) {
	return func(options *ClientOptions) {
		options.transport = transport
	}
}
