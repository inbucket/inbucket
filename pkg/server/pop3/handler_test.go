package pop3

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/textproto"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/test"
)

func TestNoTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupPOPServer(t, ds, false, false)
	pipe := setupPOPSession(t, server)
	c := textproto.NewConn(pipe)
	defer func() {
		_ = c.Close()
		server.Drain()
	}()

	reply, err := c.ReadLine()
	if err != nil {
		t.Fatalf("Reading initial line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("Initial line is not +OK")
	}

	// Verify CAPA response does not include STLS.
	if err := c.PrintfLine("CAPA"); err != nil {
		t.Fatalf("Failed to send CAPA; %v.", err)
	}
	replies, err := c.ReadDotLines()
	if err != nil {
		t.Fatalf("Reading CAPA line failed %v", err)
	}
	for _, r := range replies {
		if r == "STLS" {
			t.Errorf("TLS not enabled but received STLS.")
		}
	}
}

func TestSTLSWithTLSDisabled(t *testing.T) {
	ds := test.NewStore()
	server := setupPOPServer(t, ds, false, false)
	pipe := setupPOPSession(t, server)
	_ = pipe.SetDeadline(time.Now().Add(10 * time.Second))
	c := textproto.NewConn(pipe)
	defer func() {
		_ = c.Close()
		server.Drain()
	}()

	reply, err := c.ReadLine()
	if err != nil {
		t.Fatalf("Reading initial line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("Initial line is not +OK")
	}

	if err := c.PrintfLine("STLS"); err != nil {
		t.Fatalf("Failed to send STLS; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading STLS reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "-ERR") {
		t.Errorf("STLS should have errored: %s", reply)
	}
}

func TestStartTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupPOPServer(t, ds, true, false)
	pipe := setupPOPSession(t, server)
	c := textproto.NewConn(pipe)
	defer func() {
		_ = c.Close()
		server.Drain()
	}()

	reply, err := c.ReadLine()
	if err != nil {
		t.Fatalf("Reading initial line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("Initial line is not +OK")
	}

	// Verify CAPA response does not include STLS.
	if err := c.PrintfLine("CAPA"); err != nil {
		t.Fatalf("Failed to send CAPA; %v.", err)
	}
	replies, err := c.ReadDotLines()
	if err != nil {
		t.Fatalf("Reading CAPA line failed %v", err)
	}
	sawTLS := false
	for _, r := range replies {
		if r == "STLS" {
			sawTLS = true
		}
	}
	if !sawTLS {
		t.Errorf("TLS enabled but no STLS capability.")
	}

	if err := c.PrintfLine("STLS"); err != nil {
		t.Fatalf("Failed to send STLS; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading STLS reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("STLS failed: %s", reply)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	tlsConn := tls.Client(pipe, tlsConfig)
	ctx, toCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer toCancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		t.Fatalf("TLS handshake failed; %v", err)
	}
	c = textproto.NewConn(tlsConn)
	if err := c.PrintfLine("CAPA"); err != nil {
		t.Fatalf("Failed to send CAPA; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading CAPA reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("CAPA failed: %s", reply)
	}
	_, err = c.ReadDotLines()
	if err != nil {
		t.Fatalf("Reading CAPA line failed %v", err)
	}
}

func TestDupStartTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupPOPServer(t, ds, true, false)
	pipe := setupPOPSession(t, server)
	_ = pipe.SetDeadline(time.Now().Add(10 * time.Second))
	c := textproto.NewConn(pipe)
	defer func() {
		_ = c.Close()
		server.Drain()
	}()

	reply, err := c.ReadLine()
	if err != nil {
		t.Fatalf("Reading initial line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("Initial line is not +OK")
	}

	// Verify CAPA response includes STLS.
	if err := c.PrintfLine("CAPA"); err != nil {
		t.Fatalf("Failed to send CAPA; %v.", err)
	}
	replies, err := c.ReadDotLines()
	if err != nil {
		t.Fatalf("Reading CAPA line failed %v", err)
	}
	sawTLS := false
	for _, r := range replies {
		if r == "STLS" {
			sawTLS = true
		}
	}
	if !sawTLS {
		t.Errorf("TLS enabled but no STLS capability.")
	}

	t.Log("Sending first STLS command, expected to succeed")
	if err := c.PrintfLine("STLS"); err != nil {
		t.Fatalf("Failed to send STLS; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading STLS reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("STLS failed: %s", reply)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	tlsConn := tls.Client(pipe, tlsConfig)
	ctx, toCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer toCancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		t.Fatalf("TLS handshake failed; %v", err)
	}
	c = textproto.NewConn(tlsConn)

	t.Log("Sending second STLS command, expected to fail")
	if err := c.PrintfLine("STLS"); err != nil {
		t.Fatalf("Failed to send STLS; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading STLS reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "-ERR") {
		t.Fatalf("STLS failed: %s", reply)
	}

	// Send STAT to verify handler has not crashed.
	if err := c.PrintfLine("STAT"); err != nil {
		t.Fatalf("Failed to send STAT; %v.", err)
	}
	reply, err = c.ReadLine()
	if err != nil {
		t.Fatalf("Reading STAT reply line failed %v", err)
	}
	if !strings.HasPrefix(reply, "-ERR") {
		t.Fatalf("STAT failed: %s", reply)
	}
}

func TestForceTLS(t *testing.T) {
	ds := test.NewStore()
	server := setupPOPServer(t, ds, true, true)
	pipe := setupPOPSession(t, server)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	tlsConn := tls.Client(pipe, tlsConfig)
	ctx, toCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer toCancel()
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		t.Fatalf("TLS handshake failed; %v", err)
	}
	c := textproto.NewConn(tlsConn)
	defer func() {
		_ = c.Close()
		server.Drain()
	}()

	reply, err := c.ReadLine()
	if err != nil {
		t.Fatalf("Reading initial line failed %v", err)
	}
	if !strings.HasPrefix(reply, "+OK") {
		t.Fatalf("Initial line is not +OK")
	}

	// Verify CAPA response does not include STLS.
	if err := c.PrintfLine("CAPA"); err != nil {
		t.Fatalf("Failed to send CAPA; %v.", err)
	}
	replies, err := c.ReadDotLines()
	if err != nil {
		t.Fatalf("Reading CAPA line failed %v", err)
	}
	for _, r := range replies {
		if r == "STLS" {
			t.Errorf("STLS in CAPA in forceTLS mode.")
		}
	}
}

// net.Pipe does not implement deadlines
type mockConn struct {
	net.Conn
}

func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func setupPOPServer(t *testing.T, ds storage.Store, tls bool, forceTLS bool) *Server {
	t.Helper()
	cfg := config.POP3{
		Addr:     "127.0.0.1:2500",
		Domain:   "inbucket.local",
		Timeout:  5,
		Debug:    true,
		ForceTLS: forceTLS,
	}
	if tls {
		cert, privKey, err := generateCertificate(t)
		if err != nil {
			t.Fatalf("Failed to generate x.509 certificate; %v", err)
		}
		// we have to write these things into files.

		cfg.TLSEnabled = true
		td := t.TempDir()
		certPath := path.Join(td, "cert.pem")
		keyPath := path.Join(td, "key.pem")
		if err := os.WriteFile(certPath, certToPem(cert), 0700); err != nil {
			t.Fatalf("Failed to write cert PEM file; %v", err)
		}
		if err := os.WriteFile(keyPath, privKeyToPem(privKey), 0700); err != nil {
			t.Fatalf("Failed to write privKey PEM file; %v", err)
		}

		cfg.TLSCert = certPath
		cfg.TLSPrivKey = keyPath
	}

	s, err := NewServer(cfg, ds)
	if err != nil {
		t.Fatalf("Failed to create server: %v.", err)
	}
	return s
}

var sessionNum int

func setupPOPSession(t *testing.T, server *Server) net.Conn {
	t.Helper()
	serverConn, clientConn := net.Pipe()

	// Start the session.
	server.wg.Add(1)
	sessionNum++
	go server.startSession(sessionNum, &mockConn{serverConn})

	return clientConn
}

func privKeyToPem(privkey *rsa.PrivateKey) []byte {
	privkeyBytes := x509.MarshalPKCS1PrivateKey(privkey)
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkeyBytes,
		},
	)
}

func certToPem(cert []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})
}

func generateCertificate(t *testing.T) ([]byte, *rsa.PrivateKey, error) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatalf("Failed to generate key; %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost.local",
		},
		DNSNames:              []string{"localhost", "127.0.0.1", "inbucket.local"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageEmailProtection},
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("certificate generation failed; %v", err)
	}
	return cert, priv, nil
}
