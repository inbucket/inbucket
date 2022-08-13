package storage

import (
	"container/list"
	"context"
	"expvar"
	"time"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	scanCompletedMillis = new(expvar.Int)

	// History counters
	expRetentionDeletesTotal = new(expvar.Int)
	expRetentionPeriod       = new(expvar.Int)
	expRetainedCurrent       = new(expvar.Int)
	expRetainedSize          = new(expvar.Int)

	// History of certain stats
	retentionDeletesHist = list.New()
	retainedHist         = list.New()
	sizeHist             = list.New()

	// History rendered as comma delimited string
	expRetentionDeletesHist = new(expvar.String)
	expRetainedHist         = new(expvar.String)
	expSizeHist             = new(expvar.String)
)

func init() {
	rm := expvar.NewMap("retention")
	rm.Set("ScanCompletedMillis", scanCompletedMillis)
	rm.Set("DeletesHist", expRetentionDeletesHist)
	rm.Set("DeletesTotal", expRetentionDeletesTotal)
	rm.Set("Period", expRetentionPeriod)
	rm.Set("RetainedHist", expRetainedHist)
	rm.Set("RetainedCurrent", expRetainedCurrent)
	rm.Set("RetainedSize", expRetainedSize)
	rm.Set("SizeHist", expSizeHist)

	metric.AddTickerFunc(func() {
		expRetentionDeletesHist.Set(metric.Push(retentionDeletesHist, expRetentionDeletesTotal))
		expRetainedHist.Set(metric.Push(retainedHist, expRetainedCurrent))
		expSizeHist.Set(metric.Push(sizeHist, expRetainedSize))
	})
}

// RetentionScanner looks for messages older than the configured retention period and deletes them.
type RetentionScanner struct {
	retentionShutdown chan bool // Closed after the scanner has shut down
	ds                Store
	retentionPeriod   time.Duration
	retentionSleep    time.Duration
}

// NewRetentionScanner configures a new RententionScanner.
func NewRetentionScanner(
	cfg config.Storage,
	ds Store,
) *RetentionScanner {
	rs := &RetentionScanner{
		retentionShutdown: make(chan bool),
		ds:                ds,
		retentionPeriod:   cfg.RetentionPeriod,
		retentionSleep:    cfg.RetentionSleep,
	}
	// expRetentionPeriod is displayed on the status page
	expRetentionPeriod.Set(int64(cfg.RetentionPeriod / time.Second))
	return rs
}

// Start up the retention scanner if retention period > 0
func (rs *RetentionScanner) Start(ctx context.Context) {
	slog := log.With().Str("module", "storage").Logger()

	if rs.retentionPeriod <= 0 {
		slog.Info().Str("phase", "startup").Msg("Retention scanner disabled")
		close(rs.retentionShutdown)
		return
	}
	slog.Info().Str("phase", "startup").Msgf("Retention configured for %v", rs.retentionPeriod)

	start := time.Now()
retentionLoop:
	for {
		// Prevent scanner from starting more than once a minute
		since := time.Since(start)
		if since < time.Minute {
			dur := time.Minute - since
			slog.Debug().Msgf("Retention scanner sleeping for %v", dur)
			select {
			case <-ctx.Done():
				break retentionLoop
			case <-time.After(dur):
			}
		}
		// Kickoff scan
		start = time.Now()
		if err := rs.DoScan(ctx); err != nil {
			slog.Error().Err(err).Msg("Error during retention scan")
		}
		// Check for global shutdown
		select {
		case <-ctx.Done():
			break retentionLoop
		default:
		}
	}
	slog.Debug().Str("phase", "shutdown").Msg("Retention scanner shut down")
	close(rs.retentionShutdown)
}

// DoScan does a single pass of all mailboxes looking for messages that can be purged.
func (rs *RetentionScanner) DoScan(ctx context.Context) error {
	slog := log.With().Str("module", "storage").Logger()
	slog.Debug().Msg("Starting retention scan")
	cutoff := time.Now().Add(-1 * rs.retentionPeriod)

	// Loop over all mailboxes.
	retained := 0
	storeSize := int64(0)
	err := rs.ds.VisitMailboxes(func(messages []Message) bool {
		for _, msg := range messages {
			if msg.Date().Before(cutoff) {
				slog.Debug().Str("mailbox", msg.Mailbox()).
					Msgf("Purging expired message %v", msg.ID())
				if err := rs.ds.RemoveMessage(msg.Mailbox(), msg.ID()); err != nil {
					slog.Error().Str("mailbox", msg.Mailbox()).Err(err).
						Msgf("Failed to purge message %v", msg.ID())
				} else {
					expRetentionDeletesTotal.Add(1)
				}
			} else {
				retained++
				storeSize += msg.Size()
			}
		}
		select {
		case <-ctx.Done():
			slog.Debug().Str("phase", "shutdown").Msg("Retention scan aborted due to shutdown")
			return false
		case <-time.After(rs.retentionSleep):
			// Reduce disk thrashing
		}
		return true
	})
	if err != nil {
		return err
	}

	// Update metrics
	scanCompletedMillis.Set(time.Now().UnixNano() / 1000000)
	expRetainedCurrent.Set(int64(retained))
	expRetainedSize.Set(storeSize)

	return nil
}

// Join does not return until the retention scanner has shut down.
func (rs *RetentionScanner) Join() {
	if rs.retentionShutdown != nil {
		<-rs.retentionShutdown
	}
}
