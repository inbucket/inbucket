// Package web provides the plumbing for Inbucket's web GUI and RESTful API
package web

import (
	"context"
	"expvar"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/log"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/msghub"
)

// Handler is a function type that handles an HTTP request in Inbucket
type Handler func(http.ResponseWriter, *http.Request, *Context) error

const (
	staticDir   = "static"
	templateDir = "templates"
)

var (
	// msgHub holds a reference to the message pub/sub system
	msgHub  *msghub.Hub
	manager message.Manager

	// Router is shared between httpd, webui and rest packages. It sends
	// incoming requests to the correct handler function
	Router = mux.NewRouter()

	rootConfig     *config.Root
	server         *http.Server
	listener       net.Listener
	sessionStore   sessions.Store
	globalShutdown chan bool

	// ExpWebSocketConnectsCurrent tracks the number of open WebSockets
	ExpWebSocketConnectsCurrent = new(expvar.Int)
)

func init() {
	m := expvar.NewMap("http")
	m.Set("WebSocketConnectsCurrent", ExpWebSocketConnectsCurrent)
}

// Initialize sets up things for unit tests or the Start() method
func Initialize(
	conf *config.Root,
	shutdownChan chan bool,
	mm message.Manager,
	mh *msghub.Hub) {

	rootConfig = conf
	globalShutdown = shutdownChan

	// NewContext() will use this DataStore for the web handlers
	msgHub = mh
	manager = mm

	// Content Paths
	staticPath := filepath.Join(conf.Web.UIDir, staticDir)
	log.Infof("Web UI content mapped to path: %s", conf.Web.UIDir)
	Router.PathPrefix("/public/").Handler(http.StripPrefix("/public/",
		http.FileServer(http.Dir(staticPath))))
	http.Handle("/", Router)

	// Session cookie setup
	if conf.Web.CookieAuthKey == "" {
		log.Infof("HTTP generating random cookie.auth.key")
		sessionStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(64))
	} else {
		log.Tracef("HTTP using configured cookie.auth.key")
		sessionStore = sessions.NewCookieStore([]byte(conf.Web.CookieAuthKey))
	}
}

// Start begins listening for HTTP requests
func Start(ctx context.Context) {
	server = &http.Server{
		Addr:         rootConfig.Web.Addr,
		Handler:      nil,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// We don't use ListenAndServe because it lacks a way to close the listener
	log.Infof("HTTP listening on TCP4 %v", server.Addr)
	var err error
	listener, err = net.Listen("tcp", server.Addr)
	if err != nil {
		log.Errorf("HTTP failed to start TCP4 listener: %v", err)
		emergencyShutdown()
		return
	}

	// Listener go routine
	go serve(ctx)

	// Wait for shutdown
	select {
	case _ = <-ctx.Done():
		log.Tracef("HTTP server shutting down on request")
	}

	// Closing the listener will cause the serve() go routine to exit
	if err := listener.Close(); err != nil {
		log.Errorf("Failed to close HTTP listener: %v", err)
	}
}

// serve begins serving HTTP requests
func serve(ctx context.Context) {
	// server.Serve blocks until we close the listener
	err := server.Serve(listener)

	select {
	case _ = <-ctx.Done():
		// Nop
	default:
		log.Errorf("HTTP server failed: %v", err)
		emergencyShutdown()
		return
	}
}

// ServeHTTP builds the context and passes onto the real handler
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Create the context
	ctx, err := NewContext(req)
	if err != nil {
		log.Errorf("HTTP failed to create context: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ctx.Close()

	// Run the handler, grab the error, and report it
	log.Tracef("HTTP[%v] %v %v %q", req.RemoteAddr, req.Proto, req.Method, req.RequestURI)
	err = h(w, req, ctx)
	if err != nil {
		log.Errorf("HTTP error handling %q: %v", req.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func emergencyShutdown() {
	// Shutdown Inbucket
	select {
	case _ = <-globalShutdown:
	default:
		close(globalShutdown)
	}
}
