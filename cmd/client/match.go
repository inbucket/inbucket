package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/jhillyerd/inbucket/rest/client"
)

type matchCmd struct {
	mailbox string
	output  string
	outFunc func(headers []*client.MessageHeader) error
	delete  bool
	from    regexFlag
	subject regexFlag
	to      regexFlag
}

func (*matchCmd) Name() string {
	return "match"
}

func (*matchCmd) Synopsis() string {
	return "output messages matching criteria"
}

func (*matchCmd) Usage() string {
	return `match [options] <mailbox>:
	output messages matching all specified criteria
	exit status will be 1 if no matches were found, otherwise 0
`
}

func (m *matchCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&m.output, "output", "id", "output format: id, json, or mbox")
	f.BoolVar(&m.delete, "delete", false, "delete matched messages after output")
	f.Var(&m.from, "from", "From header matching regexp")
	f.Var(&m.subject, "subject", "Subject header matching regexp")
	f.Var(&m.to, "to", "To header matching regexp (must match one)")
}

func (m *matchCmd) Execute(
	_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
	headers, err := c.ListMailbox(mailbox)
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
	err = m.outFunc(matches)
	if err != nil {
		return fatal("Error", err)
	}
	if m.delete {
		// Delete matches
		for _, h := range matches {
			err = h.Delete()
			if err != nil {
				return fatal("Delete REST call failed", err)
			}
		}
	}
	return subcommands.ExitSuccess
}

// match returns true if header matches all defined criteria
func (m *matchCmd) match(header *client.MessageHeader) bool {
	if m.subject.Defined() {
		if !m.subject.MatchString(header.Subject) {
			return false
		}
	}
	if m.from.Defined() {
		if !m.from.MatchString(header.From) {
			return false
		}
	}
	if m.to.Defined() {
		match := false
		for _, to := range header.To {
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

func outputID(headers []*client.MessageHeader) error {
	for _, h := range headers {
		fmt.Println(h.ID)
	}
	return nil
}

func outputJSON(headers []*client.MessageHeader) error {
	jsonEncoder := json.NewEncoder(os.Stdout)
	jsonEncoder.SetEscapeHTML(false)
	jsonEncoder.SetIndent("", "  ")
	return jsonEncoder.Encode(headers)
}
