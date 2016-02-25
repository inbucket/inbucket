// Package httpd provides the plumbing for Inbucket's web GUI and RESTful API
package httpd

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/goods/httpbuf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
)

// Handler is a function type that handles an HTTP request in Inbucket
type Handler func(http.ResponseWriter, *http.Request, *Context) error

var (
	// DataStore is where all the mailboxes and messages live
	DataStore smtpd.DataStore

	// Router is shared between httpd, webui and rest packages. It sends
	// incoming requests to the correct handler function
	Router = mux.NewRouter()

	webConfig    config.WebConfig
	listener     net.Listener
	sessionStore sessions.Store
	shutdown     bool
)

// Initialize sets up things for unit tests or the Start() method
func Initialize(cfg config.WebConfig, ds smtpd.DataStore) {
	webConfig = cfg
	setupRoutes(cfg)

	// NewContext() will use this DataStore for the web handlers
	DataStore = ds

	// TODO Make configurable
	sessionStore = sessions.NewCookieStore([]byte("something-very-secret"))
}

func setupRoutes(cfg config.WebConfig) {
	log.Infof("HTTP templates mapped to %q", cfg.TemplateDir)
	log.Infof("HTTP static content mapped to %q", cfg.PublicDir)

	// Static content
	Router.PathPrefix("/public/").Handler(http.StripPrefix("/public/",
		http.FileServer(http.Dir(cfg.PublicDir))))

	// Register w/ HTTP
	http.Handle("/", Router)
}

// Start begins listening for HTTP requests
func Start() {
	addr := fmt.Sprintf("%v:%v", webConfig.IP4address, webConfig.IP4port)
	server := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// We don't use ListenAndServe because it lacks a way to close the listener
	log.Infof("HTTP listening on TCP4 %v", addr)
	var err error
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		log.Errorf("HTTP failed to start TCP4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	err = server.Serve(listener)
	if shutdown {
		log.Tracef("HTTP server shutting down on request")
	} else if err != nil {
		log.Errorf("HTTP server failed: %v", err)
	}
}

// Stop shuts down the HTTP server
func Stop() {
	log.Tracef("HTTP shutdown requested")
	shutdown = true
	if listener != nil {
		if err := listener.Close(); err != nil {
			log.Errorf("Error closing HTTP listener: %v", err)
		}
	} else {
		log.Errorf("HTTP listener was nil during shutdown")
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
	buf := new(httpbuf.Buffer)
	log.Tracef("HTTP[%v] %v %v %q", req.RemoteAddr, req.Proto, req.Method, req.RequestURI)
	err = h(buf, req, ctx)
	if err != nil {
		log.Errorf("HTTP error handling %q: %v", req.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save the session
	if err = ctx.Session.Save(req, buf); err != nil {
		log.Errorf("HTTP failed to save session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply the buffered response to the writer
	if _, err = buf.Apply(w); err != nil {
		log.Errorf("HTTP failed to write response: %v", err)
	}
}
