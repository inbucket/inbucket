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

// Server defines an instance of our POP3 server
type Server struct {
	domain         string
	maxIdleSeconds int
	dataStore      smtpd.DataStore
	listener       net.Listener
	globalShutdown chan bool
	localShutdown  chan bool
	waitgroup      *sync.WaitGroup
}

// New creates a new Server struct
func New(shutdownChan chan bool) *Server {
	// Get a new instance of the the FileDataStore - the locking and counting
	// mechanisms are both global variables in the smtpd package.  If that
	// changes in the future, this should be modified to use the same DataStore
	// instance.
	ds := smtpd.DefaultFileDataStore()
	cfg := config.GetPOP3Config()
	return &Server{
		domain:         cfg.Domain,
		dataStore:      ds,
		maxIdleSeconds: cfg.MaxIdleSeconds,
		globalShutdown: shutdownChan,
		localShutdown:  make(chan bool),
		waitgroup:      new(sync.WaitGroup),
	}
}

// Start the server and listen for connections
func (s *Server) Start() {
	cfg := config.GetPOP3Config()
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%v:%v",
		cfg.IP4address, cfg.IP4port))
	if err != nil {
		log.Errorf("POP3 Failed to build tcp4 address: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	log.Infof("POP3 listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Errorf("POP3 failed to start tcp4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	// Listener go routine
	go s.serve()

	// Wait for shutdown
	select {
	case _ = <-s.globalShutdown:
	}

	log.Tracef("POP3 shutdown requested, connections will be drained")
	// Closing the listener will cause the serve() go routine to exit
	if err := s.listener.Close(); err != nil {
		log.Errorf("Error closing POP3 listener: %v", err)
	}
}

// serve is the listen/accept loop
func (s *Server) serve() {
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
				case _ = <-s.globalShutdown:
					close(s.localShutdown)
					return
				default:
					// TODO Implement a max error counter before shutdown?
					// or maybe attempt to restart smtpd
					panic(err)
				}
			}
		} else {
			tempDelay = 0
			s.waitgroup.Add(1)
			go s.startSession(sid, conn)
		}
	}
}

// Drain causes the caller to block until all active POP3 sessions have finished
func (s *Server) Drain() {
	// Wait for listener to exit
	select {
	case _ = <-s.localShutdown:
	}
	// Wait for sessions to close
	s.waitgroup.Wait()
	log.Tracef("POP3 connections have drained")
}
