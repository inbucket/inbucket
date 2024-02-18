package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/mail"
	"os"
	"time"

	"github.com/google/subcommands"
	"github.com/inbucket/inbucket/v3/pkg/rest/client"
)

type matchCmd struct {
	output  string
	outFunc func(ctx context.Context, headers []*client.MessageHeader) error
	delete  bool
	// match criteria
	from    regexFlag
	subject regexFlag
	to      regexFlag
	maxAge  time.Duration
}

func (*matchCmd) Name() string {
	return "match"
}

func (*matchCmd) Synopsis() string {
	return "output messages matching criteria"
}

func (*matchCmd) Usage() string {
	return `match [flags] <mailbox>:
	output messages matching all specified criteria
	exit status will be 1 if no matches were found, otherwise 0
`
}

func (m *matchCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&m.output, "output", "id", "output format: id, json, or mbox")
	f.BoolVar(&m.delete, "delete", false, "delete matched messages after output")
	f.Var(&m.from, "from", "From header matching regexp (address, not name)")
	f.Var(&m.subject, "subject", "Subject header matching regexp")
	f.Var(&m.to, "to", "To header matching regexp (must match 1+ to address)")
	f.DurationVar(
		&m.maxAge, "maxage", 0,
		"Matches must have been received in this time frame (ex: \"10s\", \"5m\")")
}

func (m *matchCmd) Execute(
	ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	mailbox := f.Arg(0)
	if mailbox == "" {
		return usage("mailbox required")
	}

	// Select output function
	switch m.output {
	case "id":
		m.outFunc = outputID
	case "json":
		m.outFunc = outputJSON
	case "mbox":
		m.outFunc = outputMbox
	default:
		return usage("unknown output type: " + m.output)
	}

	// Setup REST client
	c, err := client.New(baseURL())
	if err != nil {
		return fatal("Couldn't build client", err)
	}

	// Get list
	headers, err := c.ListMailboxWithContext(ctx, mailbox)
	if err != nil {
		return fatal("List REST call failed", err)
	}

	// Find matches
	matches := make([]*client.MessageHeader, 0, len(headers))
	for _, h := range headers {
		if m.match(h) {
			matches = append(matches, h)
		}
	}

	// Return error status if no matches
	if len(matches) == 0 {
		return subcommands.ExitFailure
	}

	// Output matches
	err = m.outFunc(ctx, matches)
	if err != nil {
		return fatal("Error", err)
	}

	// Optionally, delete matches
	if m.delete {
		for _, h := range matches {
			err = h.DeleteWithContext(ctx)
			if err != nil {
				return fatal("Delete REST call failed", err)
			}
		}
	}

	return subcommands.ExitSuccess
}

// match returns true if header matches all defined criteria
func (m *matchCmd) match(header *client.MessageHeader) bool {
	if m.maxAge > 0 {
		if time.Since(header.Date) > m.maxAge {
			return false
		}
	}
	if m.subject.Defined() {
		if !m.subject.MatchString(header.Subject) {
			return false
		}
	}
	if m.from.Defined() {
		from := header.From
		addr, err := mail.ParseAddress(from)
		if err == nil {
			// Parsed successfully
			from = addr.Address
		}
		if !m.from.MatchString(from) {
			return false
		}
	}
	if m.to.Defined() {
		match := false
		for _, to := range header.To {
			addr, err := mail.ParseAddress(to)
			if err == nil {
				// Parsed successfully
				to = addr.Address
			}
			if m.to.MatchString(to) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func outputID(_ context.Context, headers []*client.MessageHeader) error {
	for _, h := range headers {
		fmt.Println(h.ID)
	}
	return nil
}

func outputJSON(_ context.Context, headers []*client.MessageHeader) error {
	jsonEncoder := json.NewEncoder(os.Stdout)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "  ")
	return jsonEncoder.Encode(headers)
}
