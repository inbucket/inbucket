package smtpd

import (
	"github.com/jhillyerd/inbucket/log"
	"time"
)

// retentionScan does a single pass of all mailboxes looking for messages that can be purged
func retentionScan(ds DataStore, maxAge time.Duration, sleep time.Duration) error {
	log.Trace("Starting retention scan")
	cutoff := time.Now().Add(-1 * maxAge)
	mboxes, err := ds.AllMailboxes()
	if err != nil {
		return err
	}

	for _, mb := range mboxes {
		messages, err := mb.GetMessages()
		if err != nil {
			return err
		}
		for _, msg := range messages {
			if msg.Date().Before(cutoff) {
				log.Trace("Purging expired message %v", msg.Id())
				err = msg.Delete()
				if err != nil {
					// Log but don't abort
					log.Error("Failed to purge message %v: %v", msg.Id(), err)
				}
			}
		}
		// Sleep after completing a mailbox
		time.Sleep(sleep)
	}

	return nil
}
