package server

import (
	"context"

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
	notify           chan error
}

// Prod wires up the production Inbucket environment.
func Prod(rootCtx context.Context, shutdownChan chan bool, conf *config.Root) (*Services, error) {
	// Configure storage.
	store, err := storage.FromConfig(conf.Storage)
	if err != nil {
		return nil, err
	}

	msgHub := msghub.New(rootCtx, conf.Web.MonitorHistory)
	addrPolicy := &policy.Addressing{Config: conf}
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, Hub: msgHub}

	// Start Retention scanner.
	retentionScanner := storage.NewRetentionScanner(conf.Storage, store, shutdownChan)
	retentionScanner.Start()

	// Configure routes and start HTTP server.
	prefix := stringutil.MakePathPrefixer(conf.Web.BasePath)
	webui.SetupRoutes(web.Router.PathPrefix(prefix("/serve/")).Subrouter())
	rest.SetupRoutes(web.Router.PathPrefix(prefix("/api/")).Subrouter())
	webServer := web.NewServer(conf, shutdownChan, mmanager, msgHub)
	go webServer.Start(rootCtx)

	// Start POP3 server.
	pop3Server := pop3.NewServer(conf.POP3, shutdownChan, store)
	go pop3Server.Start(rootCtx)

	// Start SMTP server.
	smtpServer := smtp.NewServer(conf.SMTP, shutdownChan, mmanager, addrPolicy)
	go smtpServer.Start(rootCtx)

	return &Services{
		MsgHub:           msgHub,
		RetentionScanner: retentionScanner,
		POP3Server:       pop3Server,
		SMTPServer:       smtpServer,
		WebServer:        webServer,
	}, nil
}

// Notify merges the error notification channels of all services, allowing the process to be
// shutdown if any fail.
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
