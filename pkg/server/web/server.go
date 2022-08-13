// Package web provides the plumbing for Inbucket's web GUI and RESTful API
package web

import (
	"context"
	"encoding/json"
	"expvar"
	"html/template"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/msghub"
	"github.com/inbucket/inbucket/pkg/stringutil"
	"github.com/rs/zerolog/log"
)

var (
	// msgHub holds a reference to the message pub/sub system
	msgHub  *msghub.Hub
	manager message.Manager

	// Router is shared between httpd, webui and rest packages. It sends
	// incoming requests to the correct handler function
	Router = mux.NewRouter()

	rootConfig *config.Root
	server     *http.Server
	listener   net.Listener

	// ExpWebSocketConnectsCurrent tracks the number of open WebSockets
	ExpWebSocketConnectsCurrent = new(expvar.Int)
)

func init() {
	m := expvar.NewMap("http")
	m.Set("WebSocketConnectsCurrent", ExpWebSocketConnectsCurrent)
}

// Server defines an instance of the Web server.
type Server struct {
	// TODO Migrate global vars here.
	notify chan error // Notify on fatal error.
}

// NewServer sets up things for unit tests or the Start() method.
func NewServer(
	conf *config.Root,
	mm message.Manager,
	mh *msghub.Hub) *Server {

	rootConfig = conf

	// NewContext() will use this DataStore for the web handlers.
	msgHub = mh
	manager = mm

	// Redirect requests to / if there is a base path configured.
	prefix := stringutil.MakePathPrefixer(conf.Web.BasePath)
	redirectBase := prefix("/")
	if redirectBase != "/" {
		log.Info().Str("module", "web").Str("phase", "startup").Str("path", redirectBase).
			Msg("Base path configured")
		Router.Path("/").Handler(http.RedirectHandler(redirectBase, http.StatusFound))
	}

	// Dynamic paths.
	log.Info().Str("module", "web").Str("phase", "startup").Str("path", conf.Web.UIDir).
		Msg("Web UI content mapped")
	Router.Handle(prefix("/debug/vars"), expvar.Handler())
	if conf.Web.PProf {
		Router.HandleFunc(prefix("/debug/pprof/cmdline"), pprof.Cmdline)
		Router.HandleFunc(prefix("/debug/pprof/profile"), pprof.Profile)
		Router.HandleFunc(prefix("/debug/pprof/symbol"), pprof.Symbol)
		Router.HandleFunc(prefix("/debug/pprof/trace"), pprof.Trace)
		Router.PathPrefix(prefix("/debug/pprof/")).HandlerFunc(pprof.Index)
		log.Warn().Str("module", "web").Str("phase", "startup").
			Msg("Go pprof tools installed to " + prefix("/debug/pprof"))
	}

	// Static paths.
	Router.PathPrefix(prefix("/static")).Handler(
		http.StripPrefix(prefix("/"), http.FileServer(http.Dir(conf.Web.UIDir))))
	Router.Path(prefix("/favicon.png")).Handler(
		fileHandler(filepath.Join(conf.Web.UIDir, "favicon.png")))

	// Parse index.html template, allowing for configuration to be passed to the SPA.
	indexPath := filepath.Join(conf.Web.UIDir, "index.html")
	indexTmpl, err := template.ParseFiles(indexPath)
	if err != nil {
		msg := "Failed to parse HTML template"
		cwd, _ := os.Getwd()
		log.Error().
			Str("module", "web").
			Str("phase", "startup").
			Str("path", indexPath).
			Str("cwd", cwd).
			Err(err).
			Msg(msg)
		// Create a dummy template to allow tests to pass.
		indexTmpl, _ = template.New("index.html").Parse(msg)
	}

	// SPA managed paths.
	spaHandler := cookieHandler(appConfigCookie(conf.Web),
		spaTemplateHandler(indexTmpl, prefix("/"), conf.Web))
	Router.Path(prefix("/")).Handler(spaHandler)
	Router.Path(prefix("/monitor")).Handler(spaHandler)
	Router.Path(prefix("/status")).Handler(spaHandler)
	Router.PathPrefix(prefix("/m/")).Handler(spaHandler)

	// Error handlers.
	Router.NotFoundHandler = noMatchHandler(
		http.StatusNotFound, "No route matches URI path")
	Router.MethodNotAllowedHandler = noMatchHandler(
		http.StatusMethodNotAllowed, "Method not allowed for URI path")

	s := &Server{
		notify: make(chan error, 1),
	}

	return s
}

// Start begins listening for HTTP requests
func (s *Server) Start(ctx context.Context, readyFunc func()) {
	server = &http.Server{
		Addr:         rootConfig.Web.Addr,
		Handler:      requestLoggingWrapper(Router),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// We don't use ListenAndServe because it lacks a way to close the listener
	log.Info().Str("module", "web").Str("phase", "startup").Str("addr", server.Addr).
		Msg("HTTP listening on tcp4")
	var err error
	listener, err = net.Listen("tcp", server.Addr)
	if err != nil {
		log.Error().Str("module", "web").Str("phase", "startup").Err(err).
			Msg("HTTP failed to start TCP4 listener")
		s.notify <- err
		close(s.notify)
		return
	}

	// Start listener go routine
	go s.serve(ctx)
	readyFunc()

	// Wait for shutdown
	select {
	case _ = <-ctx.Done():
		log.Debug().Str("module", "web").Str("phase", "shutdown").
			Msg("HTTP server shutting down on request")
	}

	// Closing the listener will cause the serve() go routine to exit
	if err := listener.Close(); err != nil {
		log.Debug().Str("module", "web").Str("phase", "shutdown").Err(err).
			Msg("Failed to close HTTP listener")
	}
}

func appConfigCookie(webConfig config.Web) *http.Cookie {
	o := &jsonAppConfig{
		BasePath:       webConfig.BasePath,
		MonitorVisible: webConfig.MonitorVisible,
	}
	b, err := json.Marshal(o)
	if err != nil {
		log.Error().Str("module", "web").Str("phase", "startup").Err(err).
			Msg("Failed to convert app-config to JSON")
	}
	return &http.Cookie{
		Name:  "app-config",
		Value: url.PathEscape(string(b)),
		Path:  "/",
	}
}

// serve begins serving HTTP requests
func (s *Server) serve(ctx context.Context) {
	// server.Serve blocks until we close the listener
	err := server.Serve(listener)

	select {
	case _ = <-ctx.Done():
		// Nop
	default:
		log.Error().Str("module", "web").Str("phase", "startup").Err(err).
			Msg("HTTP server failed")
		s.notify <- err
		close(s.notify)
		return
	}
}

// Notify allows the running Web server to be monitored for a fatal error.
func (s *Server) Notify() <-chan error {
	return s.notify
}
