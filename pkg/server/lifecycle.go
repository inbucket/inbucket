package server

import (
	"context"
	"sync"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/msghub"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/inbucket/inbucket/pkg/rest"
	"github.com/inbucket/inbucket/pkg/server/pop3"
	"github.com/inbucket/inbucket/pkg/server/smtp"
	"github.com/inbucket/inbucket/pkg/server/web"
	"github.com/inbucket/inbucket/pkg/storage"
	"github.com/inbucket/inbucket/pkg/stringutil"
	"github.com/inbucket/inbucket/pkg/webui"
)

// Services holds the configured services.
type Services struct {
	MsgHub           *msghub.Hub
	POP3Server       *pop3.Server
	RetentionScanner *storage.RetentionScanner
	SMTPServer       *smtp.Server
	WebServer        *web.Server
	notify           chan error      // Combined notification for failed services.
	ready            *sync.WaitGroup // Tracks services that have not reported ready.
}

// Prod wires up the production Inbucket environment.
func Prod(conf *config.Root) (*Services, error) {
	// Configure storage.
	store, err := storage.FromConfig(conf.Storage)
	if err != nil {
		return nil, err
	}

	addrPolicy := &policy.Addressing{Config: conf}
	msgHub := msghub.New(conf.Web.MonitorHistory)
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, Hub: msgHub}

	// Start Retention scanner.
	retentionScanner := storage.NewRetentionScanner(conf.Storage, store)

	// Configure routes and build HTTP server.
	prefix := stringutil.MakePathPrefixer(conf.Web.BasePath)
	webui.SetupRoutes(web.Router.PathPrefix(prefix("/serve/")).Subrouter())
	rest.SetupRoutes(web.Router.PathPrefix(prefix("/api/")).Subrouter())
	webServer := web.NewServer(conf, mmanager, msgHub)

	pop3Server := pop3.NewServer(conf.POP3, store)
	smtpServer := smtp.NewServer(conf.SMTP, mmanager, addrPolicy)

	return &Services{
		MsgHub:           msgHub,
		RetentionScanner: retentionScanner,
		POP3Server:       pop3Server,
		SMTPServer:       smtpServer,
		WebServer:        webServer,
		ready:            &sync.WaitGroup{},
	}, nil
}

// Start all services, returns immediately.  Callers may use Notify to detect failed services.
func (s *Services) Start(ctx context.Context, readyFunc func()) {
	// TODO: Try some bad listening configs to ensure startup aborts correctly.
	go s.MsgHub.Start(ctx)
	go s.WebServer.Start(ctx, s.makeReadyFunc())
	go s.SMTPServer.Start(ctx, s.makeReadyFunc())
	go s.POP3Server.Start(ctx, s.makeReadyFunc())
	go s.RetentionScanner.Start(ctx)

	// Notify when all services report ready.
	go func() {
		s.ready.Wait()
		readyFunc()
	}()
}

// Notify merges the error notification channels of all fallible services, allowing the process to
// be shutdown if needed.
func (s *Services) Notify() <-chan error {
	c := make(chan error, 1)
	go func() {
		// TODO: What level to log failure.
		select {
		case err := <-s.POP3Server.Notify():
			c <- err
		case err := <-s.SMTPServer.Notify():
			c <- err
		case err := <-s.WebServer.Notify():
			c <- err
		}
	}()

	return c
}

func (s *Services) makeReadyFunc() func() {
	s.ready.Add(1)
	var once sync.Once
	return func() {
		once.Do(s.ready.Done)
	}
}
