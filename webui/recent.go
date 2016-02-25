package webui

import (
	"github.com/jhillyerd/inbucket/httpd"
)

const (
	// maximum mailboxes to remember
	maxRemembered = 8
	// session value key; referenced in templates, do not change
	mailboxKey = "recentMailboxes"
)

// RememberMailbox manages the list of recently accessed mailboxes stored in the session
func RememberMailbox(ctx *httpd.Context, mailbox string) {
	recent := RecentMailboxes(ctx)
	newRecent := make([]string, 1, maxRemembered)
	newRecent[0] = mailbox

	for _, recBox := range recent {
		// Insert until newRecent is full, but don't repeat the new mailbox
		if len(newRecent) < maxRemembered && mailbox != recBox {
			newRecent = append(newRecent, recBox)
		}
	}

	ctx.Session.Values[mailboxKey] = newRecent
}

// RecentMailboxes returns a slice of the most recently accessed mailboxes
func RecentMailboxes(ctx *httpd.Context) []string {
	val := ctx.Session.Values[mailboxKey]
	recent, _ := val.([]string)
	return recent
}
