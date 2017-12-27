package smtpd

import (
	"container/list"
	"context"
	"expvar"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/datastore"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/msghub"
)

func init() {
	m := expvar.NewMap("smtp")
	m.Set("ConnectsTotal", expConnectsTotal)
	m.Set("ConnectsHist", expConnectsHist)
	m.Set("ConnectsCurrent", expConnectsCurrent)
	m.Set("ReceivedTotal", expReceivedTotal)
	m.Set("ReceivedHist", expReceivedHist)
	m.Set("ErrorsTotal", expErrorsTotal)
	m.Set("ErrorsHist", expErrorsHist)
	m.Set("WarnsTotal", expWarnsTotal)
	m.Set("WarnsHist", expWarnsHist)

	log.AddTickerFunc(func() {
		expReceivedHist.Set(log.PushMetric(deliveredHist, expReceivedTotal))
		expConnectsHist.Set(log.PushMetric(connectsHist, expConnectsTotal))
		expErrorsHist.Set(log.PushMetric(errorsHist, expErrorsTotal))
		expWarnsHist.Set(log.PushMetric(warnsHist, expWarnsTotal))
		expRetentionDeletesHist.Set(log.PushMetric(retentionDeletesHist, expRetentionDeletesTotal))
		expRetainedHist.Set(log.PushMetric(retainedHist, expRetainedCurrent))
	})
}

// Server holds the configuration and state of our SMTP server
type Server struct {
	// Configuration
	host            string
	domain          string
	domainNoStore   string
	maxRecips       int
	maxIdleSeconds  int
	maxMessageBytes int
	storeMessages   bool

	// Dependencies
	dataStore        datastore.DataStore // Mailbox/message store
	globalShutdown   chan bool           // Shuts down Inbucket
	msgHub           *msghub.Hub         // Pub/sub for message info
	retentionScanner *RetentionScanner   // Deletes expired messages

	// State
	listener  net.Listener    // Incoming network connections
	waitgroup *sync.WaitGroup // Waitgroup tracks individual sessions
}

var (
	// Raw stat collectors
	expConnectsTotal   = new(expvar.Int)
	expConnectsCurrent = new(expvar.Int)
	expReceivedTotal   = new(expvar.Int)
	expErrorsTotal     = new(expvar.Int)
	expWarnsTotal      = new(expvar.Int)

	// History of certain stats
	deliveredHist = list.New()
	connectsHist  = list.New()
	errorsHist    = list.New()
	warnsHist     = list.New()

	// History rendered as comma delim string
	expReceivedHist = new(expvar.String)
	expConnectsHist = new(expvar.String)
	expErrorsHist   = new(expvar.String)
	expWarnsHist    = new(expvar.String)
)

// NewServer creates a new Server instance with the specificed config
func NewServer(
	cfg config.SMTPConfig,
	globalShutdown chan bool,
	ds datastore.DataStore,
	msgHub *msghub.Hub) *Server {
	return &Server{
		host:             fmt.Sprintf("%v:%v", cfg.IP4address, cfg.IP4port),
		domain:           cfg.Domain,
		domainNoStore:    strings.ToLower(cfg.DomainNoStore),
		maxRecips:        cfg.MaxRecipients,
		maxIdleSeconds:   cfg.MaxIdleSeconds,
		maxMessageBytes:  cfg.MaxMessageBytes,
		storeMessages:    cfg.StoreMessages,
		globalShutdown:   globalShutdown,
		dataStore:        ds,
		msgHub:           msgHub,
		retentionScanner: NewRetentionScanner(ds, globalShutdown),
		waitgroup:        new(sync.WaitGroup),
	}
}

// Start the listener and handle incoming connections
func (s *Server) Start(ctx context.Context) {
	addr, err := net.ResolveTCPAddr("tcp4", s.host)
	if err != nil {
		log.Errorf("Failed to build tcp4 address: %v", err)
		s.emergencyShutdown()
		return
	}

	log.Infof("SMTP listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Errorf("SMTP failed to start tcp4 listener: %v", err)
		s.emergencyShutdown()
		return
	}

	if !s.storeMessages {
		log.Infof("Load test mode active, messages will not be stored")
	} else if s.domainNoStore != "" {
		log.Infof("Messages sent to domain '%v' will be discarded", s.domainNoStore)
	}

	// Start retention scanner
	s.retentionScanner.Start()

	// Listener go routine
	go s.serve(ctx)

	// Wait for shutdown
	select {
	case <-ctx.Done():
		log.Tracef("SMTP shutdown requested, connections will be drained")
	}

	// Closing the listener will cause the serve() go routine to exit
	if err := s.listener.Close(); err != nil {
		log.Errorf("Failed to close SMTP listener: %v", err)
	}
}

// serve is the listen/accept loop
func (s *Server) serve(ctx context.Context) {
	// Handle incoming connections
	var tempDelay time.Duration
	for sessionID := 1; ; sessionID++ {
		if conn, err := s.listener.Accept(); err != nil {
			// There was an error accepting the connection
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
				log.Errorf("SMTP accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				// Permanent error
				select {
				case <-ctx.Done():
					// SMTP is shutting down
					return
				default:
					// Something went wrong
					s.emergencyShutdown()
					return
				}
			}
		} else {
			tempDelay = 0
			expConnectsTotal.Add(1)
			s.waitgroup.Add(1)
			go s.startSession(sessionID, conn)
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

// Drain causes the caller to block until all active SMTP sessions have finished
func (s *Server) Drain() {
	// Wait for sessions to close
	s.waitgroup.Wait()
	log.Tracef("SMTP connections have drained")
	s.retentionScanner.Join()
}
