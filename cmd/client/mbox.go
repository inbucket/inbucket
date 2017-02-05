package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/jhillyerd/inbucket/rest/client"
)

type mboxCmd struct {
	mailbox string
	delete  bool
}

func (*mboxCmd) Name() string {
	return "mbox"
}

func (*mboxCmd) Synopsis() string {
	return "output mailbox in mbox format"
}

func (*mboxCmd) Usage() string {
	return `mbox [options] <mailbox>:
	output mailbox in mbox format
`
}

func (m *mboxCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&m.delete, "delete", false, "delete messages after output")
}

func (m *mboxCmd) Execute(
	_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	mailbox := f.Arg(0)
	if mailbox == "" {
		return usage("mailbox required")
	}
	// Setup rest client
	c, err := client.New(baseURL())
	if err != nil {
		return fatal("Couldn't build client", err)
	}
	// Get list
	headers, err := c.ListMailbox(mailbox)
	if err != nil {
		return fatal("List REST call failed", err)
	}
	for _, h := range headers {
		source, err := h.GetSource()
		if err != nil {
			return fatal("Source REST call failed", err)
		}
		fmt.Printf("From %s\n", h.From)
		// TODO Escape "From " in message bodies with >
		source.WriteTo(os.Stdout)
		fmt.Println()
		if m.delete {
			err = h.Delete()
			if err != nil {
				return fatal("Delete REST call failed", err)
			}
		}
	}

	return subcommands.ExitSuccess
}
