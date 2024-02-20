package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	smtpclient "net/smtp"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/rest"
	"github.com/inbucket/inbucket/v3/pkg/rest/client"
	"github.com/inbucket/inbucket/v3/pkg/server/smtp"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/storage/mem"
	"github.com/inbucket/inbucket/v3/pkg/webui"
	"github.com/jhillyerd/goldiff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

const (
	restBaseURL = "http://127.0.0.1:9000/"
	smtpHost    = "127.0.0.1:2500"
)

// TODO: Add suites for domain and full addressing modes.
type IntegrationSuite struct {
	suite.Suite
	stopServer func()
}

func (s *IntegrationSuite) SetupSuite() {
	stopServer, err := startServer()
	s.Require().NoError(err)
	s.stopServer = stopServer
}

func (s *IntegrationSuite) TearDownSuite() {
	s.stopServer()
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationSuite))
}

func (s *IntegrationSuite) TestBasic() {
	client, err := client.New(restBaseURL)
	s.Require().NoError(err)
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("basic.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	s.Require().NoError(err)

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	s.Require().NoError(err)
	s.NotNil(msg)

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(s.T(), got, "testdata", "basic.golden")
}

func (s *IntegrationSuite) TestFullname() {
	client, err := client.New(restBaseURL)
	s.Require().NoError(err)
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("fullname.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	s.Require().NoError(err)

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	s.Require().NoError(err)
	s.NotNil(msg)

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(s.T(), got, "testdata", "fullname.golden")
}

func (s *IntegrationSuite) TestEncodedHeader() {
	client, err := client.New(restBaseURL)
	s.Require().NoError(err)
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("encodedheader.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	s.Require().NoError(err)

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	s.Require().NoError(err)
	s.NotNil(msg)

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(s.T(), got, "testdata", "encodedheader.golden")
}

func (s *IntegrationSuite) TestIPv4Recipient() {
	client, err := client.New(restBaseURL)
	s.Require().NoError(err)
	from := "fromuser@inbucket.org"
	to := []string{"ip4recipient@[192.168.123.123]"}
	input := readTestData("no-to.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	s.Require().NoError(err)

	// Confirm receipt.
	msg, err := client.GetMessage("ip4recipient", "latest")
	s.Require().NoError(err)
	s.NotNil(msg)

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(s.T(), got, "testdata", "no-to-ipv4.golden")
}

func (s *IntegrationSuite) TestIPv6Recipient() {
	client, err := client.New(restBaseURL)
	s.Require().NoError(err)
	from := "fromuser@inbucket.org"
	to := []string{"ip6recipient@[IPv6:2001:0db8:85a3:0000:0000:8a2e:0370:7334]"}
	input := readTestData("no-to.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	s.Require().NoError(err)

	// Confirm receipt.
	msg, err := client.GetMessage("ip6recipient", "latest")
	s.Require().NoError(err)
	s.NotNil(msg)

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(s.T(), got, "testdata", "no-to-ipv6.golden")
}

func formatMessage(m *client.Message) []byte {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "Mailbox: %v\n", m.Mailbox)
	fmt.Fprintf(b, "From: %v\n", m.From)
	fmt.Fprintf(b, "To: %v\n", m.To)
	fmt.Fprintf(b, "Subject: %v\n", m.Subject)
	fmt.Fprintf(b, "Size: %v\n", m.Size)
	fmt.Fprintf(b, "\nBODY TEXT:\n%v\n", m.Body.Text)
	fmt.Fprintf(b, "\nBODY HTML:\n%v\n", m.Body.HTML)
	return b.Bytes()
}

func startServer() (func(), error) {
	// TODO Move integration setup into lifecycle.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})

	extHost := extension.NewHost()

	// Storage setup.
	storage.Constructors["memory"] = mem.New
	clearEnv()
	conf, err := config.Process()
	if err != nil {
		return nil, err
	}
	svcCtx, svcCancel := context.WithCancel(context.Background())
	store, err := storage.FromConfig(conf.Storage, extHost)
	if err != nil {
		svcCancel()
		return nil, err
	}

	// TODO Test should not pass with unstarted msghub.
	addrPolicy := &policy.Addressing{Config: conf}
	msgHub := msghub.New(conf.Web.MonitorHistory, extHost)
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, ExtHost: extHost}

	// Start HTTP server.
	webui.SetupRoutes(web.Router.PathPrefix("/serve/").Subrouter())
	rest.SetupRoutes(web.Router.PathPrefix("/api/").Subrouter())
	webServer := web.NewServer(conf, mmanager, msgHub)
	go webServer.Start(svcCtx, func() {})

	// Start SMTP server.
	smtpServer := smtp.NewServer(conf.SMTP, mmanager, addrPolicy, extHost)
	go smtpServer.Start(svcCtx, func() {})

	// TODO Use a readyFunc to determine server readiness.
	time.Sleep(500 * time.Millisecond)

	return func() {
		// Shut everything down.
		svcCancel()
		smtpServer.Drain()
	}, nil
}

func readTestData(path ...string) []byte {
	// Prefix path with testdata.
	p := append([]string{"testdata"}, path...)
	f, err := os.Open(filepath.Join(p...))
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return data
}

// clearEnv clears environment variables, preserving any that are critical for this OS.
func clearEnv() {
	preserve := make(map[string]string)
	backup := func(k string) {
		preserve[k] = os.Getenv(k)
	}

	// Backup ciritcal env variables.
	if runtime.GOOS == "windows" {
		backup("SYSTEMROOT")
	}

	os.Clearenv()

	for k, v := range preserve {
		os.Setenv(k, v)
	}
}
