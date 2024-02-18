package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
	"github.com/inbucket/inbucket/v3/pkg/rest/client"
)

type mboxCmd struct {
	delete bool
}

func (*mboxCmd) Name() string {
	return "mbox"
}

func (*mboxCmd) Synopsis() string {
	return "output mailbox in mbox format"
}

func (*mboxCmd) Usage() string {
	return `mbox [flags] <mailbox>:
	output mailbox in mbox format
`
}

func (m *mboxCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&m.delete, "delete", false, "delete messages after output")
}

func (m *mboxCmd) Execute(
	ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	mailbox := f.Arg(0)
	if mailbox == "" {
		return usage("mailbox required")
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
	err = outputMbox(ctx, headers)
	if err != nil {
		return fatal("Error", err)
	}

	// Optionally, delete retrieved messages
	if m.delete {
		for _, h := range headers {
			err = h.DeleteWithContext(ctx)
			if err != nil {
				return fatal("Delete REST call failed", err)
			}
		}
	}

	return subcommands.ExitSuccess
}

// outputMbox renders messages in mbox format.
// It is also used by match subcommand.
func outputMbox(ctx context.Context, headers []*client.MessageHeader) error {
	for _, h := range headers {
		source, err := h.GetSourceWithContext(ctx)
		if err != nil {
			return fmt.Errorf("get source REST failed: %v", err)
		}

		fmt.Printf("From %s\n", h.From)
		// TODO Escape "From " in message bodies with >
		if _, err := source.WriteTo(os.Stdout); err != nil {
			return err
		}
		fmt.Println()
	}
	return nil
}
