package smtp

import (
	"container/list"
	"context"
	"expvar"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/metric"
	"github.com/jhillyerd/inbucket/pkg/policy"
	"github.com/rs/zerolog/log"
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

	metric.AddTickerFunc(func() {
		expReceivedHist.Set(metric.Push(deliveredHist, expReceivedTotal))
		expConnectsHist.Set(metric.Push(connectsHist, expConnectsTotal))
		expErrorsHist.Set(metric.Push(errorsHist, expErrorsTotal))
		expWarnsHist.Set(metric.Push(warnsHist, expWarnsTotal))
	})
}

// Server holds the configuration and state of our SMTP server
type Server struct {
	// Configuration
	host            string
	domain          string
	domainNoStore   string
	maxRecips       int
	maxMessageBytes int
	storeMessages   bool
	timeout         time.Duration

	// Dependencies
	apolicy        *policy.Addressing // Address policy.
	globalShutdown chan bool          // Shuts down Inbucket.
	manager        message.Manager    // Used to deliver messages.

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
	cfg config.SMTP,
	globalShutdown chan bool,
	manager message.Manager,
	apolicy *policy.Addressing,
) *Server {
	return &Server{
		host:            cfg.Addr,
		domain:          cfg.Domain,
		domainNoStore:   strings.ToLower(cfg.DomainNoStore),
		maxRecips:       cfg.MaxRecipients,
		timeout:         cfg.Timeout,
		maxMessageBytes: cfg.MaxMessageBytes,
		storeMessages:   cfg.StoreMessages,
		globalShutdown:  globalShutdown,
		manager:         manager,
		apolicy:         apolicy,
		waitgroup:       new(sync.WaitGroup),
	}
}

// Start the listener and handle incoming connections.
func (s *Server) Start(ctx context.Context) {
	slog := log.With().Str("module", "smtp").Str("phase", "startup").Logger()
	addr, err := net.ResolveTCPAddr("tcp4", s.host)
	if err != nil {
		slog.Error().Err(err).Msg("Failed to build tcp4 address")
		s.emergencyShutdown()
		return
	}
	slog.Info().Str("addr", addr.String()).Msg("SMTP listening on tcp4")
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		slog.Error().Err(err).Msg("Failed to start tcp4 listener")
		s.emergencyShutdown()
		return
	}
	if !s.storeMessages {
		slog.Info().Msg("Load test mode active, messages will not be stored")
	} else if s.domainNoStore != "" {
		slog.Info().Msgf("Messages sent to domain '%v' will be discarded", s.domainNoStore)
	}
	// Listener go routine.
	go s.serve(ctx)
	// Wait for shutdown.
	<-ctx.Done()
	slog = log.With().Str("module", "smtp").Str("phase", "shutdown").Logger()
	slog.Debug().Msg("SMTP shutdown requested, connections will be drained")
	// Closing the listener will cause the serve() go routine to exit.
	if err := s.listener.Close(); err != nil {
		slog.Error().Err(err).Msg("Failed to close SMTP listener")
	}
}

// serve is the listen/accept loop.
func (s *Server) serve(ctx context.Context) {
	// Handle incoming connections.
	var tempDelay time.Duration
	for sessionID := 1; ; sessionID++ {
		if conn, err := s.listener.Accept(); err != nil {
			// There was an error accepting the connection.
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				// Temporary error, sleep for a bit and try again.
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Error().Str("module", "smtp").Err(err).
					Msgf("SMTP accept error; retrying in %v", tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				// Permanent error.
				select {
				case <-ctx.Done():
					// SMTP is shutting down.
					return
				default:
					// Something went wrong.
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
	// Shutdown Inbucket.
	select {
	case <-s.globalShutdown:
	default:
		close(s.globalShutdown)
	}
}

// Drain causes the caller to block until all active SMTP sessions have finished
func (s *Server) Drain() {
	// Wait for sessions to close.
	s.waitgroup.Wait()
	log.Debug().Str("module", "smtp").Str("phase", "shutdown").Msg("SMTP connections have drained")
}
