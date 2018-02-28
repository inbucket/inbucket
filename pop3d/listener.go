package pop3d

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/datastore"
	"github.com/jhillyerd/inbucket/log"
)

// Server defines an instance of our POP3 server
type Server struct {
	host           string
	domain         string
	maxIdleSeconds int
	dataStore      datastore.DataStore
	listener       net.Listener
	globalShutdown chan bool
	waitgroup      *sync.WaitGroup
}

// New creates a new Server struct
func New(cfg config.POP3Config, shutdownChan chan bool, ds datastore.DataStore) *Server {
	return &Server{
		host:           fmt.Sprintf("%v:%v", cfg.IP4address, cfg.IP4port),
		domain:         cfg.Domain,
		dataStore:      ds,
		maxIdleSeconds: cfg.MaxIdleSeconds,
		globalShutdown: shutdownChan,
		waitgroup:      new(sync.WaitGroup),
	}
}

// Start the server and listen for connections
func (s *Server) Start(ctx context.Context) {
	addr, err := net.ResolveTCPAddr("tcp4", s.host)
	if err != nil {
		log.Errorf("POP3 Failed to build tcp4 address: %v", err)
		s.emergencyShutdown()
		return
	}

	log.Infof("POP3 listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Errorf("POP3 failed to start tcp4 listener: %v", err)
		s.emergencyShutdown()
		return
	}

	// Listener go routine
	go s.serve(ctx)

	// Wait for shutdown
	select {
	case _ = <-ctx.Done():
	}

	log.Tracef("POP3 shutdown requested, connections will be drained")
	// Closing the listener will cause the serve() go routine to exit
	if err := s.listener.Close(); err != nil {
		log.Errorf("Error closing POP3 listener: %v", err)
	}
}

// serve is the listen/accept loop
func (s *Server) serve(ctx context.Context) {
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
				log.Errorf("POP3 accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				// Permanent error
				select {
				case <-ctx.Done():
					// POP3 is shutting down
					return
				default:
					// Something went wrong
					s.emergencyShutdown()
					return
				}
			}
		} else {
			tempDelay = 0
			s.waitgroup.Add(1)
			go s.startSession(sid, conn)
		}
	}
}

func (s *Server) emergencyShutdown() {
	// Shutdown Inbucket
	select {
	case _ = <-s.globalShutdown:
	default:
		close(s.globalShutdown)
	}
}

// Drain causes the caller to block until all active POP3 sessions have finished
func (s *Server) Drain() {
	// Wait for sessions to close
	s.waitgroup.Wait()
	log.Tracef("POP3 connections have drained")
}
