package smtpd

import (
	"container/list"
	"expvar"
	"fmt"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"net"
	"time"
)

// Real server code starts here
type Server struct {
	domain          string
	maxRecips       int
	maxIdleSeconds  int
	maxMessageBytes int
	dataStore       *DataStore
}

// Raw stat collectors
var expConnectsTotal = new(expvar.Int)
var expConnectsCurrent = new(expvar.Int)
var expDeliveredTotal = new(expvar.Int)
var expErrorsTotal = new(expvar.Int)
var expWarnsTotal = new(expvar.Int)

// History of certain stats
var deliveredHist = list.New()
var connectsHist = list.New()
var errorsHist = list.New()
var warnsHist = list.New()

// History rendered as comma delim string
var expDeliveredHist = new(expvar.String)
var expConnectsHist = new(expvar.String)
var expErrorsHist = new(expvar.String)
var expWarnsHist = new(expvar.String)

// Init a new Server object
func New() *Server {
	ds := NewDataStore()
	// TODO Make more of these configurable
	return &Server{domain: config.GetSmtpConfig().Domain, maxRecips: 100, maxIdleSeconds: 300,
		dataStore: ds, maxMessageBytes: 2048000}
}

// Main listener loop
func (s *Server) Start() {
	cfg := config.GetSmtpConfig()
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%v:%v",
		cfg.Ip4address, cfg.Ip4port))
	if err != nil {
		log.Error("Failed to build tcp4 address: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	log.Info("SMTP listening on TCP4 %v", addr)
	ln, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Error("Failed to start tcp4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	for sid := 1; ; sid++ {
		if conn, err := ln.Accept(); err != nil {
			// TODO Implement a max error counter before shutdown?
			// or maybe attempt to restart smtpd
			panic(err)
		} else {
			expConnectsTotal.Add(1)
			go s.startSession(sid, conn)
		}
	}
}

// When the provided Ticker ticks, we update our metrics history
func metricsTicker(t *time.Ticker) {
	ok := true
	for ok {
		_, ok = <-t.C
		expDeliveredHist.Set(pushMetric(deliveredHist, expDeliveredTotal))
		expConnectsHist.Set(pushMetric(connectsHist, expConnectsTotal))
		expErrorsHist.Set(pushMetric(errorsHist, expErrorsTotal))
		expWarnsHist.Set(pushMetric(warnsHist, expWarnsTotal))
	}
}

// pushMetric adds the metric to the end of the list and returns a comma
// separated string of the previous 50 entries
func pushMetric(history *list.List, ev expvar.Var) string {
	history.PushBack(ev.String())
	if history.Len() > 50 {
		history.Remove(history.Front())
	}
	return JoinStringList(history)
}

func init() {
	m := expvar.NewMap("smtp")
	m.Set("ConnectsTotal", expConnectsTotal)
	m.Set("ConnectsHist", expConnectsHist)
	m.Set("ConnectsCurrent", expConnectsCurrent)
	m.Set("DeliveredTotal", expDeliveredTotal)
	m.Set("DeliveredHist", expDeliveredHist)
	m.Set("ErrorsTotal", expErrorsTotal)
	m.Set("ErrorsHist", expErrorsHist)
	m.Set("WarnsTotal", expWarnsTotal)
	m.Set("WarnsHist", expWarnsHist)

	t := time.NewTicker(time.Minute)
	go metricsTicker(t)
}
