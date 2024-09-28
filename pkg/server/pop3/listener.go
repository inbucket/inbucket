package pop3

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/storage"
	"github.com/rs/zerolog/log"
)

// Server defines an instance of the POP3 server.
type Server struct {
	config    config.POP3     // POP3 configuration.
	store     storage.Store   // Mail store.
	listener  net.Listener    // TCP listener.
	wg        *sync.WaitGroup // Waitgroup tracking sessions.
	notify    chan error      // Notify on fatal error.
	tlsConfig *tls.Config     // TLS encryption configuration.
	tlsState  *tls.ConnectionState
}

// NewServer creates a new, unstarted, POP3 server.
func NewServer(pop3Config config.POP3, store storage.Store) (*Server, error) {
	slog := log.With().Str("module", "pop3").Str("phase", "tls").Logger()
	tlsConfig := &tls.Config{}
	if pop3Config.TLSEnabled {
		var err error
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(pop3Config.TLSCert, pop3Config.TLSPrivKey)
		if err != nil {
			slog.Error().Msgf("Failed loading X509 KeyPair: %v", err)
			return nil, fmt.Errorf("failed to configure TLS; %v", err)
			// Do not silently turn off Security.
		}
		slog.Debug().Msg("TLS config available")
	} else {
		tlsConfig = nil
	}
	return &Server{
		config:    pop3Config,
		store:     store,
		wg:        new(sync.WaitGroup),
		notify:    make(chan error, 1),
		tlsConfig: tlsConfig,
	}, nil
}

// Start the server and listen for connections
func (s *Server) Start(ctx context.Context, readyFunc func()) {
	slog := log.With().Str("module", "pop3").Str("phase", "startup").Logger()
	addr, err := net.ResolveTCPAddr("tcp4", s.config.Addr)
	if err != nil {
		slog.Error().Err(err).Msg("Failed to build tcp4 address")
		s.notify <- err
		close(s.notify)
		return
	}
	slog.Info().Str("addr", addr.String()).Msg("POP3 listening on tcp4")
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		slog.Error().Err(err).Msg("Failed to start tcp4 listener")
		s.notify <- err
		close(s.notify)
		return
	}

	// Start listener go routine.
	go s.serve(ctx)
	readyFunc()

	// Wait for shutdown.
	<-ctx.Done()
	slog = log.With().Str("module", "pop3").Str("phase", "shutdown").Logger()
	slog.Debug().Msg("POP3 shutdown requested, connections will be drained")

	// Closing the listener will cause the serve() go routine to exit.
	if err := s.listener.Close(); err != nil {
		slog.Error().Err(err).Msg("Failed to close POP3 listener")
	}
}

// serve is the listen/accept loop.
func (s *Server) serve(ctx context.Context) {
	// Handle incoming connections.
	var tempDelay time.Duration
	for sid := 1; ; sid++ {
		if conn, err := s.listener.Accept(); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				// Timeout, sleep for a bit and try again.
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if maxDelay := 1 * time.Second; tempDelay > maxDelay {
					tempDelay = maxDelay
				}
				log.Error().Str("module", "pop3").Err(err).
					Msgf("POP3 accept timout; retrying in %v", tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				// Permanent error.
				select {
				case <-ctx.Done():
					// POP3 is shutting down.
					return
				default:
					// Something went wrong.
					s.notify <- err
					close(s.notify)
					return
				}
			}
		} else {
			tempDelay = 0
			s.wg.Add(1)
			go s.startSession(sid, conn)
		}
	}
}

// Drain causes the caller to block until all active POP3 sessions have finished
func (s *Server) Drain() {
	// Wait for sessions to close
	log.Debug().Str("module", "pop3").Str("phase", "shutdown").Msg("waiting for connections to complete.")
	s.wg.Wait()
	log.Debug().Str("module", "pop3").Str("phase", "shutdown").Msg("POP3 connections have drained")
}

// Notify allows the running POP3 server to be monitored for a fatal error.
func (s *Server) Notify() <-chan error {
	return s.notify
}
