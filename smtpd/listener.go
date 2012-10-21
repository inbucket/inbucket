package smtpd

import (
	"fmt"
	"github.com/jhillyerd/inbucket"
	"net"
)

// Real server code starts here
type Server struct {
	domain          string
	maxRecips       int
	maxIdleSeconds  int
	maxMessageBytes int
	dataStore       *inbucket.DataStore
}

// Init a new Server object
func New() *Server {
	ds := inbucket.NewDataStore()
	// TODO Make more of these configurable
	return &Server{domain: inbucket.GetSmtpConfig().Domain, maxRecips: 100, maxIdleSeconds: 300,
		dataStore: ds, maxMessageBytes: 2048000}
}

// Main listener loop
func (s *Server) Start() {
	cfg := inbucket.GetSmtpConfig()
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%v:%v",
		cfg.Ip4address, cfg.Ip4port))
	if err != nil {
		inbucket.Error("Failed to build tcp4 address: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	inbucket.Info("Listening on TCP4 %v", addr)
	ln, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		inbucket.Error("Failed to start tcp4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	for sid := 1; ; sid++ {
		if conn, err := ln.Accept(); err != nil {
			// TODO Implement a max error counter before shutdown?
			// or maybe attempt to restart smtpd
			panic(err)
		} else {
			go s.startSession(sid, conn)
		}
	}
}
