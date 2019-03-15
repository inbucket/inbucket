package test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	smtpclient "net/smtp"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/msghub"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/rest"
	"github.com/inbucket/inbucket/pkg/rest/client"
	"github.com/inbucket/inbucket/pkg/server/smtp"
	"github.com/inbucket/inbucket/pkg/server/web"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/storage/mem"
	"github.com/inbucket/inbucket/pkg/webui"
	"github.com/jhillyerd/goldiff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	restBaseURL = "http://127.0.0.1:9000/"
	smtpHost    = "127.0.0.1:2500"
)

func TestSuite(t *testing.T) {
	stopServer, err := startServer()
	if err != nil {
		t.Fatal(err)
	}
	defer stopServer()

	testCases := []struct {
		name string
		test func(*testing.T)
	}{
		{"basic", testBasic},
		{"fullname", testFullname},
		{"encodedHeader", testEncodedHeader},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func testBasic(t *testing.T) {
	client, err := client.New(restBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("basic.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	if err != nil {
		t.Fatal(err)
	}

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if msg == nil {
		t.Errorf("Got nil message, wanted non-nil message.")
	}

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(t, got, "testdata", "basic.golden")
}

func testFullname(t *testing.T) {
	client, err := client.New(restBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("fullname.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	if err != nil {
		t.Fatal(err)
	}

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if msg == nil {
		t.Errorf("Got nil message, wanted non-nil message.")
	}

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(t, got, "testdata", "fullname.golden")
}

func testEncodedHeader(t *testing.T) {
	client, err := client.New(restBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	from := "fromuser@inbucket.org"
	to := []string{"recipient@inbucket.org"}
	input := readTestData("encodedheader.txt")

	// Send mail.
	err = smtpclient.SendMail(smtpHost, nil, from, to, input)
	if err != nil {
		t.Fatal(err)
	}

	// Confirm receipt.
	msg, err := client.GetMessage("recipient", "latest")
	if err != nil {
		t.Fatal(err)
	}
	if msg == nil {
		t.Errorf("Got nil message, wanted non-nil message.")
	}

	// Compare to golden.
	got := formatMessage(msg)
	goldiff.File(t, got, "testdata", "encodedheader.golden")
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
	// TODO Refactor inbucket/main.go so we don't need to repeat all this here.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
	storage.Constructors["memory"] = mem.New
	os.Clearenv()
	conf, err := config.Process()
	if err != nil {
		return nil, err
	}
	rootCtx, rootCancel := context.WithCancel(context.Background())
	shutdownChan := make(chan bool)
	store, err := storage.FromConfig(conf.Storage)
	if err != nil {
		rootCancel()
		return nil, err
	}
	msgHub := msghub.New(rootCtx, conf.Web.MonitorHistory)
	addrPolicy := &policy.Addressing{Config: conf}
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, Hub: msgHub}
	// Start HTTP server.
	webui.SetupRoutes(web.Router.PathPrefix("/serve/").Subrouter())
	rest.SetupRoutes(web.Router.PathPrefix("/api/").Subrouter())
	web.Initialize(conf, shutdownChan, mmanager, msgHub)
	go web.Start(rootCtx)
	// Start SMTP server.
	smtpServer := smtp.NewServer(conf.SMTP, shutdownChan, mmanager, addrPolicy)
	go smtpServer.Start(rootCtx)

	// TODO Implmement an elegant way to determine server readiness.
	time.Sleep(500 * time.Millisecond)

	return func() {
		// Shut everything down.
		close(shutdownChan)
		rootCancel()
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
	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return data
}
