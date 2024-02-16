package server

import (
	"context"
	"sync"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/luahost"
	"github.com/inbucket/inbucket/v3/pkg/message"
	"github.com/inbucket/inbucket/v3/pkg/msghub"
	"github.com/inbucket/inbucket/v3/pkg/policy"
	"github.com/inbucket/inbucket/v3/pkg/rest"
	"github.com/inbucket/inbucket/v3/pkg/server/pop3"
	"github.com/inbucket/inbucket/v3/pkg/server/smtp"
	"github.com/inbucket/inbucket/v3/pkg/server/web"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/inbucket/inbucket/v3/pkg/stringutil"
	"github.com/inbucket/inbucket/v3/pkg/webui"
)

// Services holds the configured services.
type Services struct {
	MsgHub           *msghub.Hub
	POP3Server       *pop3.Server
	RetentionScanner *storage.RetentionScanner
	SMTPServer       *smtp.Server
	WebServer        *web.Server
	ExtHost          *extension.Host
	LuaHost          *luahost.Host
	notify           chan error      // Combined notification for failed services.
	ready            *sync.WaitGroup // Tracks services that have not reported ready.
}

// FullAssembly wires up a complete Inbucket environment.
func FullAssembly(conf *config.Root) (*Services, error) {
	// Configure extensions.
	extHost := extension.NewHost()
	luaHost, err := luahost.New(conf.Lua, extHost)
	if err != nil && err != luahost.ErrNoScript {
		return nil, err
	}

	// Configure storage.
	store, err := storage.FromConfig(conf.Storage, extHost)
	if err != nil {
		return nil, err
	}

	addrPolicy := &policy.Addressing{Config: conf}
	// Configure shared components.
	msgHub := msghub.New(conf.Web.MonitorHistory, extHost)
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, ExtHost: extHost}

	// Start Retention scanner.
	retentionScanner := storage.NewRetentionScanner(conf.Storage, store)

	// Configure routes and build HTTP server.
	prefix := stringutil.MakePathPrefixer(conf.Web.BasePath)
	webui.SetupRoutes(web.Router.PathPrefix(prefix("/serve/")).Subrouter())
	rest.SetupRoutes(web.Router.PathPrefix(prefix("/api/")).Subrouter())
	webServer := web.NewServer(conf, mmanager, msgHub)

	pop3Server, err := pop3.NewServer(conf.POP3, store)
	if err != nil {
		return nil, err
	}
	smtpServer := smtp.NewServer(conf.SMTP, mmanager, addrPolicy, extHost)

	s := &Services{
		MsgHub:           msgHub,
		RetentionScanner: retentionScanner,
		POP3Server:       pop3Server,
		SMTPServer:       smtpServer,
		WebServer:        webServer,
		ExtHost:          extHost,
		LuaHost:          luaHost,
		ready:            &sync.WaitGroup{},
	}
	s.setupNotify()

	return s, nil
}

// Start all services, returns immediately.  Callers may use Notify to detect failed services.
func (s *Services) Start(ctx context.Context, readyFunc func()) {
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

// Notify returns a merged channel of the error notification channels of all fallible services,
// allowing the process to be shutdown if needed.
func (s *Services) Notify() <-chan error {
	return s.notify
}

// setupNotify merges the error notification channels of all fallible services.
func (s *Services) setupNotify() {
	c := make(chan error, 1)
	s.notify = c
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
}

// makeReadyFunc returns a function used to signal that a service is ready. The `Services.ready`
// wait group can then be used to await all services being ready.
func (s *Services) makeReadyFunc() func() {
	s.ready.Add(1)
	var once sync.Once
	return func() {
		once.Do(s.ready.Done)
	}
}
