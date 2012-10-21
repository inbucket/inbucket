/*
	The web package contains all the code to provide Inbucket's web GUI
*/
package web

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jhillyerd/inbucket"
	"net/http"
	"time"
)

/*
type WebServer struct {
	thing string
}

// NewServer() returns a new web.Server instance
func NewWebServer() *Server {
	return &WebServer{}
}
*/

var Router *mux.Router

func setupRoutes(cfg inbucket.WebConfig) {
	r := mux.NewRouter()
	Router = r
	inbucket.Info("Theme templates mapped to '%v'", cfg.TemplatesDir)
	inbucket.Info("Theme static content mapped to '%v'", cfg.PublicDir)

	// Static content
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/",
		http.FileServer(http.Dir(cfg.PublicDir))))
}

// Start() the web server
func Start() {
	cfg := inbucket.GetWebConfig()
	setupRoutes(cfg)
	addr := fmt.Sprintf("%v:%v", cfg.Ip4address, cfg.Ip4port)
	inbucket.Info("HTTP listening on TCP4 %v", addr)

	s := &http.Server{
		Addr:         addr,
		Handler:      Router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	err := s.ListenAndServe()
	if err != nil {
		inbucket.Error("HTTP server failed: %v", err)
	}
}
