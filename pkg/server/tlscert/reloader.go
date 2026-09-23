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

// fileState identifies a version of a file by its modification time and size.
type fileState struct {
	modTime time.Time
	size    int64
}

// Reloader serves a certificate and private key loaded from files, reloading them when either
// file changes. It is safe for concurrent use.
type Reloader struct {
	certFile string         // X509 public certificate file.
	keyFile  string         // X509 private key file.
	logger   zerolog.Logger // Logger for reload events.

	mu        sync.Mutex       // Guards the fields below.
	cert      *tls.Certificate // Last successfully loaded key pair.
	certState fileState        // State of certFile at the last load attempt.
	keyState  fileState        // State of keyFile at the last load attempt.
}

// NewReloader loads the key pair from certFile and keyFile, returning an error if it cannot be
// loaded.
func NewReloader(certFile, keyFile string, logger zerolog.Logger) (*Reloader, error) {
	r := &Reloader{certFile: certFile, keyFile: keyFile, logger: logger}
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
// the last load attempt. A failed reload is logged, and the previously loaded key pair is served
// until the files change again. It matches the signature of tls.Config.GetCertificate.
func (r *Reloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	certState, keyState, err := r.stat()
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to check X509 KeyPair files, serving previous certificate")
		return r.cert, nil
	}
	if certState == r.certState && keyState == r.keyState {
		return r.cert, nil
	}

	// Record the attempt even if it fails, so a bad pair is retried only after another change.
	r.certState = certState
	r.keyState = keyState
	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to reload X509 KeyPair, serving previous certificate")
		return r.cert, nil
	}
	r.cert = &cert
	r.logger.Info().Msg("Reloaded X509 KeyPair")
	return r.cert, nil
}

// stat returns the current state of the certificate and key files.
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
