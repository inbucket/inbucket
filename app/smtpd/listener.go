package smtpd

import (
	"fmt"
	"github.com/jhillyerd/inbucket/app/inbucket"
	"github.com/robfig/revel"
	"net"
)

// Real server code starts here
type Server struct {
	domain         string
	port           int
	maxRecips      int
	maxIdleSeconds int
	dataStore      *inbucket.DataStore
}

// Init a new Server object
func New(domain string, port int) *Server {
	ds := inbucket.NewDataStore()
	return &Server{domain: domain, port: port, maxRecips: 100, maxIdleSeconds: 60,
		dataStore: ds}
}

// Loggers
func (s *Server) trace(msg string, args ...interface{}) {
	rev.TRACE.Printf(msg, args...)
}

func (s *Server) info(msg string, args ...interface{}) {
	rev.INFO.Printf(msg, args...)
}

func (s *Server) warn(msg string, args ...interface{}) {
	rev.WARN.Printf(msg, args...)
}

func (s *Server) error(msg string, args ...interface{}) {
	rev.ERROR.Printf(msg, args...)
}

// Main listener loop
func (s *Server) Start() {
	s.trace("Server Start() called")
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", s.port))
	if err != nil {
		panic(err)
	}

	for sid := 1; ; sid++ {
		if conn, err := ln.Accept(); err != nil {
			panic(err)
		} else {
			go s.startSession(sid, conn)
		}
	}
}
