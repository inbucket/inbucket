// Package tlscert provides X509 key pairs that follow changes to their files on disk.
package tlscert

import (
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// retryInterval is the minimum time between attempts to reload a key pair that failed to load
// while its files are unchanged, such as when a chmod or chown fixes its permissions.
const retryInterval = 30 * time.Second

// fileState identifies a version of a file by its modification time and size.
type fileState struct {
	modTime time.Time
	size    int64
}

// Reloader serves a certificate and private key loaded from files, reloading them when either
// file changes. It is safe for concurrent use.
type Reloader struct {
	certFile      string           // X509 public certificate file.
	keyFile       string           // X509 private key file.
	logger        zerolog.Logger   // Logger for reload events.
	retryInterval time.Duration    // Minimum time between retries of a failed, unchanged pair.
	now           func() time.Time // Clock used to schedule retries.

	mu        sync.Mutex       // Guards the fields below.
	cert      *tls.Certificate // Last successfully loaded key pair.
	certState fileState        // State of certFile at the last load attempt.
	keyState  fileState        // State of keyFile at the last load attempt.
	failedAt  time.Time        // Time of the last failed load attempt, zero after a success.
}

// NewReloader loads the key pair from certFile and keyFile, returning an error if it cannot be
// loaded.
func NewReloader(certFile, keyFile string, logger zerolog.Logger) (*Reloader, error) {
	r := &Reloader{
		certFile:      certFile,
		keyFile:       keyFile,
		logger:        logger,
		retryInterval: retryInterval,
		now:           time.Now,
	}
	certState, keyState, err := r.stat()
	if err != nil {
		return nil, err
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	r.cert = &cert
	r.certState = certState
	r.keyState = keyState
	return r, nil
}

// GetCertificate returns the current key pair, reloading it first if either file changed since
// the last load attempt, or if the last attempt failed and retryInterval has elapsed. Failures
// are logged and the previous key pair is served. crypto/tls calls it on full handshakes only.
func (r *Reloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// On error, stat returns zero states, so repeated stat failures count as unchanged files.
	certState, keyState, err := r.stat()
	changed := certState != r.certState || keyState != r.keyState
	retryDue := !r.failedAt.IsZero() && r.now().Sub(r.failedAt) >= r.retryInterval
	if !changed && !retryDue {
		return r.cert, nil
	}

	r.certState = certState
	r.keyState = keyState
	if err == nil {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(r.certFile, r.keyFile)
		if err == nil {
			r.cert = &cert
			r.failedAt = time.Time{}
			r.logger.Info().Msg("Reloaded X509 KeyPair")
			return r.cert, nil
		}
	}
	r.failedAt = r.now()
	r.logger.Warn().Err(err).Msg("Failed to reload X509 KeyPair, serving previous certificate")
	return r.cert, nil
}

// stat returns the current state of the certificate and key files, or zero states and an error
// if either cannot be read.
func (r *Reloader) stat() (certState, keyState fileState, err error) {
	certState, err = statFile(r.certFile)
	if err != nil {
		return fileState{}, fileState{}, err
	}
	keyState, err = statFile(r.keyFile)
	if err != nil {
		return fileState{}, fileState{}, err
	}
	return certState, keyState, nil
}

func statFile(name string) (fileState, error) {
	info, err := os.Stat(name)
	if err != nil {
		return fileState{}, fmt.Errorf("stat %s: %w", name, err)
	}
	return fileState{modTime: info.ModTime(), size: info.Size()}, nil
}
