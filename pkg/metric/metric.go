package metric

import (
	"container/list"
	"expvar"
	"strings"
	"time"
)

// TickerFunc is the function signature accepted by AddTickerFunc, will be called once per minute.
type TickerFunc func()

var tickerFuncChan = make(chan TickerFunc)

func init() {
	go metricsTicker()
}

// AddTickerFunc adds a new function callback to the list of metrics TickerFuncs that get
// called each minute.
func AddTickerFunc(f TickerFunc) {
	tickerFuncChan <- f
}

// Push adds the metric to the end of the list and returns a comma separated string of the
// previous 61 entries.  We return 61 instead of 60 (an hour) because the chart on the client
// tracks deltas between these values - there is nothing to compare the first value against.
func Push(history *list.List, ev expvar.Var) string {
	history.PushBack(ev.String())
	if history.Len() > 61 {
		history.Remove(history.Front())
	}
	return joinStringList(history)
}

// metricsTicker calls the current list of TickerFuncs once per minute.
func metricsTicker() {
	funcs := make([]TickerFunc, 0)
	ticker := time.NewTicker(time.Minute)

	for {
		select {
		case <-ticker.C:
			for _, f := range funcs {
				f()
			}
		case f := <-tickerFuncChan:
			funcs = append(funcs, f)
		}
	}
}

// joinStringList joins a List containing strings by commas.
func joinStringList(listOfStrings *list.List) string {
	if listOfStrings.Len() == 0 {
		return ""
	}
	s := make([]string, 0, listOfStrings.Len())
	for e := listOfStrings.Front(); e != nil; e = e.Next() {
		s = append(s, e.Value.(string))
	}
	return strings.Join(s, ",")
}
