package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/inbucket/inbucket/v3/pkg/rest/client"
)

type listCmd struct{}

func (*listCmd) Name() string {
	return "list"
}

func (*listCmd) Synopsis() string {
	return "list contents of mailbox"
}

func (*listCmd) Usage() string {
	return `list <mailbox>:
	list message IDs in mailbox
`
}

func (l *listCmd) SetFlags(f *flag.FlagSet) {}

func (l *listCmd) Execute(
	ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
	headers, err := c.ListMailboxWithContext(ctx, mailbox)
	if err != nil {
		return fatal("REST call failed", err)
	}
	for _, h := range headers {
		fmt.Println(h.ID)
	}

	return subcommands.ExitSuccess
}
