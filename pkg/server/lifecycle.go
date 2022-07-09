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

// Services holds the configured and started services.
type Services struct {
	MsgHub           *msghub.Hub
	POP3Server       *pop3.Server
	RetentionScanner *storage.RetentionScanner
	SMTPServer       *smtp.Server
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
	web.Initialize(conf, shutdownChan, mmanager, msgHub)
	go web.Start(rootCtx)

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
	}, nil
}
