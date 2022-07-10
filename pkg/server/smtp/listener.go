package smtp

import (
	"container/list"
	"context"
	"crypto/tls"
	"expvar"
	"net"
	"sync"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/message"
	"github.com/inbucket/inbucket/pkg/metric"
	"github.com/inbucket/inbucket/pkg/policy"
	"github.com/rs/zerolog/log"
)

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

// Server holds the configuration and state of our SMTP server.
type Server struct {
	config         config.SMTP        // SMTP configuration.
	addrPolicy     *policy.Addressing // Address policy.
	globalShutdown chan bool          // Shuts down Inbucket.
	manager        message.Manager    // Used to deliver messages.
	listener       net.Listener       // Incoming network connections.
	wg             *sync.WaitGroup    // Waitgroup tracks individual sessions.
	tlsConfig      *tls.Config        // TLS encryption configuration.
	notify         chan error         // Notify on fatal error.
}

// NewServer creates a new, unstarted, SMTP server instance with the specificed config.
func NewServer(
	smtpConfig config.SMTP,
	globalShutdown chan bool,
	manager message.Manager,
	apolicy *policy.Addressing,
) *Server {
	slog := log.With().Str("module", "smtp").Str("phase", "tls").Logger()
	tlsConfig := &tls.Config{}
	if smtpConfig.TLSEnabled {
		var err error
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(smtpConfig.TLSCert, smtpConfig.TLSPrivKey)
		if err != nil {
			slog.Error().Msgf("Failed loading X509 KeyPair: %v", err)
			slog.Error().Msg("Disabling STARTTLS support")
			smtpConfig.TLSEnabled = false
		} else {
			slog.Debug().Msg("STARTTLS feature available")
		}
	}

	return &Server{
		config:         smtpConfig,
		globalShutdown: globalShutdown,
		manager:        manager,
		addrPolicy:     apolicy,
		wg:             new(sync.WaitGroup),
		tlsConfig:      tlsConfig,
		notify:         make(chan error, 1),
	}
}

// Start the listener and handle incoming connections.
func (s *Server) Start(ctx context.Context) {
	slog := log.With().Str("module", "smtp").Str("phase", "startup").Logger()
	addr, err := net.ResolveTCPAddr("tcp4", s.config.Addr)
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
					s.notify <- err
					close(s.notify)
					s.emergencyShutdown()
					return
				}
			}
		} else {
			tempDelay = 0
			expConnectsTotal.Add(1)
			s.wg.Add(1)
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
	s.wg.Wait()
	log.Debug().Str("module", "smtp").Str("phase", "shutdown").Msg("SMTP connections have drained")
}

// Notify allows the running SMTP server to be monitored for a fatal error.
func (s *Server) Notify() <-chan error {
	return s.notify
}
