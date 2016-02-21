package smtpd

import (
	"container/list"
	"expvar"
	"sync"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
)

var retentionScanCompleted time.Time
var retentionScanCompletedMu sync.RWMutex

var expRetentionDeletesTotal = new(expvar.Int)
var expRetentionPeriod = new(expvar.Int)
var expRetainedCurrent = new(expvar.Int)

// History of certain stats
var retentionDeletesHist = list.New()
var retainedHist = list.New()

// History rendered as comma delimited string
var expRetentionDeletesHist = new(expvar.String)
var expRetainedHist = new(expvar.String)

// StartRetentionScanner launches a go-routine that scans for expired
// messages, following the configured interval
func StartRetentionScanner(ds DataStore) {
	cfg := config.GetDataStoreConfig()
	expRetentionPeriod.Set(int64(cfg.RetentionMinutes * 60))
	if cfg.RetentionMinutes > 0 {
		// Retention scanning enabled
		log.Infof("Retention configured for %v minutes", cfg.RetentionMinutes)
		go retentionScanner(ds, time.Duration(cfg.RetentionMinutes)*time.Minute,
			time.Duration(cfg.RetentionSleep)*time.Millisecond)
	} else {
		log.Infof("Retention scanner disabled")
	}
}

func retentionScanner(ds DataStore, maxAge time.Duration, sleep time.Duration) {
	start := time.Now()
	for {
		// Prevent scanner from running more than once a minute
		since := time.Since(start)
		if since < time.Minute {
			dur := time.Minute - since
			log.Tracef("Retention scanner sleeping for %v", dur)
			time.Sleep(dur)
		}
		start = time.Now()

		// Kickoff scan
		if err := doRetentionScan(ds, maxAge, sleep); err != nil {
			log.Errorf("Error during retention scan: %v", err)
		}
	}
}

// doRetentionScan does a single pass of all mailboxes looking for messages that can be purged
func doRetentionScan(ds DataStore, maxAge time.Duration, sleep time.Duration) error {
	log.Tracef("Starting retention scan")
	cutoff := time.Now().Add(-1 * maxAge)
	mboxes, err := ds.AllMailboxes()
	if err != nil {
		return err
	}

	retained := 0
	for _, mb := range mboxes {
		messages, err := mb.GetMessages()
		if err != nil {
			return err
		}
		for _, msg := range messages {
			if msg.Date().Before(cutoff) {
				log.Tracef("Purging expired message %v", msg.ID())
				err = msg.Delete()
				if err != nil {
					// Log but don't abort
					log.Errorf("Failed to purge message %v: %v", msg.ID(), err)
				} else {
					expRetentionDeletesTotal.Add(1)
				}
			} else {
				retained++
			}
		}
		// Sleep after completing a mailbox
		time.Sleep(sleep)
	}

	setRetentionScanCompleted(time.Now())
	expRetainedCurrent.Set(int64(retained))

	return nil
}

func setRetentionScanCompleted(t time.Time) {
	retentionScanCompletedMu.Lock()
	defer retentionScanCompletedMu.Unlock()

	retentionScanCompleted = t
}

func getRetentionScanCompleted() time.Time {
	retentionScanCompletedMu.RLock()
	defer retentionScanCompletedMu.RUnlock()

	return retentionScanCompleted
}

func secondsSinceRetentionScanCompleted() interface{} {
	return time.Since(getRetentionScanCompleted()) / time.Second
}

func init() {
	rm := expvar.NewMap("retention")
	rm.Set("SecondsSinceScanCompleted", expvar.Func(secondsSinceRetentionScanCompleted))
	rm.Set("DeletesHist", expRetentionDeletesHist)
	rm.Set("DeletesTotal", expRetentionDeletesTotal)
	rm.Set("Period", expRetentionPeriod)
	rm.Set("RetainedHist", expRetainedHist)
	rm.Set("RetainedCurrent", expRetainedCurrent)
}
