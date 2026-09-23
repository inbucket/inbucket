package tlscert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReloaderMissingFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := NewReloader(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), testLogger(t))
	require.Error(t, err)
}

func TestNewReloaderMismatchedPair(t *testing.T) {
	dir := t.TempDir()
	certA, _ := generatePair(t, "a")
	_, keyB := generatePair(t, "b")
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	writeFile(t, certFile, certA, 1)
	writeFile(t, keyFile, keyB, 1)

	_, err := NewReloader(certFile, keyFile, testLogger(t))
	require.Error(t, err)
}

func TestGetCertificateReloadsOnChange(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	writePairFiles(t, certFile, keyFile, "a", 1)

	r, err := NewReloader(certFile, keyFile, testLogger(t))
	require.NoError(t, err)
	assert.Equal(t, "a", commonName(t, r))

	writePairFiles(t, certFile, keyFile, "b", 2)
	assert.Equal(t, "b", commonName(t, r))
}

func TestGetCertificateKeepsPreviousOnFailure(t *testing.T) {
	tcs := []struct {
		name   string
		mangle func(t *testing.T, certFile, keyFile string)
	}{
		{
			name: "invalid certificate",
			mangle: func(t *testing.T, certFile, keyFile string) {
				t.Helper()
				writeFile(t, certFile, []byte("not a certificate"), 2)
			},
		},
		{
			name: "certificate rotated before key",
			mangle: func(t *testing.T, certFile, keyFile string) {
				t.Helper()
				certB, _ := generatePair(t, "b")
				writeFile(t, certFile, certB, 2)
			},
		},
		{
			name: "files removed",
			mangle: func(t *testing.T, certFile, keyFile string) {
				t.Helper()
				require.NoError(t, os.Remove(certFile))
				require.NoError(t, os.Remove(keyFile))
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
			writePairFiles(t, certFile, keyFile, "a", 1)

			r, err := NewReloader(certFile, keyFile, testLogger(t))
			require.NoError(t, err)

			tc.mangle(t, certFile, keyFile)
			assert.Equal(t, "a", commonName(t, r))

			// A valid pair written afterward is picked up.
			writePairFiles(t, certFile, keyFile, "c", 3)
			assert.Equal(t, "c", commonName(t, r))
		})
	}
}

// TestGetCertificateSymlinkSwap mirrors how the kubelet updates a mounted Secret: the files are
// symlinks through a ..data symlink, which is atomically repointed at a new directory.
func TestGetCertificateSymlinkSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows cannot rename over an existing directory symlink")
	}
	dir := t.TempDir()
	writePairFiles(t, filepath.Join(dir, "v1", "cert.pem"), filepath.Join(dir, "v1", "key.pem"), "a", 1)
	require.NoError(t, os.Symlink("v1", filepath.Join(dir, "..data")))
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	require.NoError(t, os.Symlink(filepath.Join("..data", "cert.pem"), certFile))
	require.NoError(t, os.Symlink(filepath.Join("..data", "key.pem"), keyFile))

	r, err := NewReloader(certFile, keyFile, testLogger(t))
	require.NoError(t, err)
	assert.Equal(t, "a", commonName(t, r))

	writePairFiles(t, filepath.Join(dir, "v2", "cert.pem"), filepath.Join(dir, "v2", "key.pem"), "b", 2)
	require.NoError(t, os.Symlink("v2", filepath.Join(dir, "..data_tmp")))
	require.NoError(t, os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")))
	assert.Equal(t, "b", commonName(t, r))
}

func TestGetCertificateConcurrent(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	writePairFiles(t, certFile, keyFile, "a", 1)

	r, err := NewReloader(certFile, keyFile, testLogger(t))
	require.NoError(t, err)

	done := make(chan struct{})
	for range 8 {
		go func() {
			defer func() { done <- struct{}{} }()
			for range 50 {
				cert, err := r.GetCertificate(nil)
				assert.NoError(t, err)
				assert.NotNil(t, cert)
			}
		}()
	}
	writePairFiles(t, certFile, keyFile, "b", 2)
	for range 8 {
		<-done
	}
	assert.Equal(t, "b", commonName(t, r))
}

func testLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(zerolog.NewTestWriter(t))
}

// commonName returns the subject common name of the certificate r currently serves.
func commonName(t *testing.T, r *Reloader) string {
	t.Helper()
	cert, err := r.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	return leaf.Subject.CommonName
}

// writePairFiles writes a new key pair for commonName, stamping both files with a modification
// time derived from version so each version is distinguishable regardless of timestamp
// granularity.
func writePairFiles(t *testing.T, certFile, keyFile, commonName string, version int) {
	t.Helper()
	certPEM, keyPEM := generatePair(t, commonName)
	writeFile(t, certFile, certPEM, version)
	writeFile(t, keyFile, keyPEM, version)
}

func writeFile(t *testing.T, name string, data []byte, version int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o700))
	require.NoError(t, os.WriteFile(name, data, 0o600))
	mtime := time.Date(2026, 1, 1, 0, 0, version, 0, time.UTC)
	require.NoError(t, os.Chtimes(name, mtime, mtime))
}

// generatePair returns PEM encoded certificate and private key for a self-signed certificate.
func generatePair(t *testing.T, commonName string) (certPEM, keyPEM []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return certPEM, keyPEM
}
