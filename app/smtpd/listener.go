package smtpd

import (
  "fmt"
  "net"
)

// Real server code starts here
type Server struct {
  domain string
  port int
  maxRecips int
  maxIdleSeconds int
}

// Init a new Server object
func New(domain string, port int) *Server {
  return &Server{domain: domain, port: port, maxRecips: 3, maxIdleSeconds: 10}
}

// Loggers
func (s *Server) trace(msg string, args ...interface {}) {
  fmt.Printf("[trace] %s\n", fmt.Sprintf(msg, args...)) 
}

func (s *Server) info(msg string, args ...interface {}) {
  fmt.Printf("[info ] %s\n", fmt.Sprintf(msg, args...)) 
}

func (s *Server) warn(msg string, args ...interface {}) {
  fmt.Printf("[warn ] %s\n", fmt.Sprintf(msg, args...)) 
}

func (s *Server) error(msg string, args ...interface {}) {
  fmt.Printf("[error] %s\n", fmt.Sprintf(msg, args...)) 
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

