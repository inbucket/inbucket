package smtpd

import (
	"container/list"
	"expvar"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
)

// Server holds the configuration and state of our SMTP server
type Server struct {
	domain          string
	domainNoStore   string
	maxRecips       int
	maxIdleSeconds  int
	maxMessageBytes int
	dataStore       DataStore
	storeMessages   bool
	listener        net.Listener

	// globalShutdown is the signal Inbucket needs to shut down
	globalShutdown chan bool

	// localShutdown indicates this component has completed shutting down
	localShutdown chan bool

	// waitgroup tracks individual sessions
	waitgroup *sync.WaitGroup
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
func NewServer(cfg config.SMTPConfig, ds DataStore, globalShutdown chan bool) *Server {
	return &Server{
		dataStore:       ds,
		domain:          cfg.Domain,
		maxRecips:       cfg.MaxRecipients,
		maxIdleSeconds:  cfg.MaxIdleSeconds,
		maxMessageBytes: cfg.MaxMessageBytes,
		storeMessages:   cfg.StoreMessages,
		domainNoStore:   strings.ToLower(cfg.DomainNoStore),
		waitgroup:       new(sync.WaitGroup),
		globalShutdown:  globalShutdown,
		localShutdown:   make(chan bool),
	}
}

// Start the listener and handle incoming connections
func (s *Server) Start() {
	cfg := config.GetSMTPConfig()
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%v:%v",
		cfg.IP4address, cfg.IP4port))
	if err != nil {
		log.Errorf("Failed to build tcp4 address: %v", err)
		// serve() never called, so we do local shutdown here
		close(s.localShutdown)
		s.emergencyShutdown()
		return
	}

	log.Infof("SMTP listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Errorf("SMTP failed to start tcp4 listener: %v", err)
		// serve() never called, so we do local shutdown here
		close(s.localShutdown)
		s.emergencyShutdown()
		return
	}

	if !s.storeMessages {
		log.Infof("Load test mode active, messages will not be stored")
	} else if s.domainNoStore != "" {
		log.Infof("Messages sent to domain '%v' will be discarded", s.domainNoStore)
	}

	// Start retention scanner
	StartRetentionScanner(s.dataStore, s.globalShutdown)

	// Listener go routine
	go s.serve()

	// Wait for shutdown
	select {
	case _ = <-s.globalShutdown:
		log.Tracef("SMTP shutdown requested, connections will be drained")
	}

	// Closing the listener will cause the serve() go routine to exit
	if err := s.listener.Close(); err != nil {
		log.Errorf("Failed to close SMTP listener: %v", err)
	}
}

// serve is the listen/accept loop
func (s *Server) serve() {
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
				case _ = <-s.globalShutdown:
					close(s.localShutdown)
					return
				default:
					close(s.localShutdown)
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
	// Wait for listener to exit
	select {
	case _ = <-s.localShutdown:
	}
	// Wait for sessions to close
	s.waitgroup.Wait()
	log.Tracef("SMTP connections have drained")
	RetentionJoin()
}

// When the provided Ticker ticks, we update our metrics history
func metricsTicker(t *time.Ticker) {
	ok := true
	for ok {
		_, ok = <-t.C
		expReceivedHist.Set(pushMetric(deliveredHist, expReceivedTotal))
		expConnectsHist.Set(pushMetric(connectsHist, expConnectsTotal))
		expErrorsHist.Set(pushMetric(errorsHist, expErrorsTotal))
		expWarnsHist.Set(pushMetric(warnsHist, expWarnsTotal))
		expRetentionDeletesHist.Set(pushMetric(retentionDeletesHist, expRetentionDeletesTotal))
		expRetainedHist.Set(pushMetric(retainedHist, expRetainedCurrent))
	}
}

// pushMetric adds the metric to the end of the list and returns a comma separated string of the
// previous 61 entries.  We return 61 instead of 60 (an hour) because the chart on the client
// tracks deltas between these values - there is nothing to compare the first value against.
func pushMetric(history *list.List, ev expvar.Var) string {
	history.PushBack(ev.String())
	if history.Len() > 61 {
		history.Remove(history.Front())
	}
	return JoinStringList(history)
}

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

	t := time.NewTicker(time.Minute)
	go metricsTicker(t)
}
