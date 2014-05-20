package pop3d

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
)

// Real server code starts here
type Server struct {
	domain         string
	maxIdleSeconds int
	dataStore      smtpd.DataStore
	listener       net.Listener
	shutdown       bool
	waitgroup      *sync.WaitGroup
}

// Init a new Server object
func New() *Server {
	// TODO is two filestores better/worse than sharing w/ smtpd?
	ds := smtpd.DefaultFileDataStore()
	cfg := config.GetPop3Config()
	return &Server{domain: cfg.Domain, dataStore: ds, maxIdleSeconds: cfg.MaxIdleSeconds,
		waitgroup: new(sync.WaitGroup)}
}

// Main listener loop
func (s *Server) Start() {
	cfg := config.GetPop3Config()
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%v:%v",
		cfg.Ip4address, cfg.Ip4port))
	if err != nil {
		log.LogError("POP3 Failed to build tcp4 address: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	log.LogInfo("POP3 listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		log.LogError("POP3 failed to start tcp4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	// Handle incoming connections
	var tempDelay time.Duration
	for sid := 1; ; sid++ {
		if conn, err := s.listener.Accept(); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				// Temporary error, sleep for a bit and try again
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.LogError("POP3 accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				if s.shutdown {
					log.LogTrace("POP3 listener shutting down on request")
					return
				}
				// TODO Implement a max error counter before shutdown?
				// or maybe attempt to restart POP3
				panic(err)
			}
		} else {
			tempDelay = 0
			s.waitgroup.Add(1)
			go s.startSession(sid, conn)
		}
	}
}

// Stop requests the POP3 server closes it's listener
func (s *Server) Stop() {
	log.LogTrace("POP3 shutdown requested, connections will be drained")
	s.shutdown = true
	s.listener.Close()
}

// Drain causes the caller to block until all active POP3 sessions have finished
func (s *Server) Drain() {
	s.waitgroup.Wait()
	log.LogTrace("POP3 connections drained")
}
