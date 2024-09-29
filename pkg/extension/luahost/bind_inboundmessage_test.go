package luahost

import (
	"net/mail"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundMessageGetters(t *testing.T) {
	want := &event.InboundMessage{
		Mailboxes: []string{"mb1", "mb2"},
		From:      &mail.Address{Name: "name1", Address: "addr1"},
		To: []*mail.Address{
			{Name: "name2", Address: "addr2"},
			{Name: "name3", Address: "addr3"},
		},
		Subject: "subj1",
		Size:    42,
	}
	script := `
		assert(msg, "msg should not be nil")

		assert_eq(msg.mailboxes, {"mb1", "mb2"})
		assert_eq(msg.subject, "subj1")
		assert_eq(msg.size, 42, "msg.size")

		assert_eq(msg.from.name, "name1", "from.name")
		assert_eq(msg.from.address, "addr1", "from.address")

		assert_eq(#msg.to, 2, "#msg.to")
		assert_eq(msg.to[1].name, "name2", "to[1].name")
		assert_eq(msg.to[1].address, "addr2", "to[1].address")
		assert_eq(msg.to[2].name, "name3", "to[2].name")
		assert_eq(msg.to[2].address, "addr3", "to[2].address")
	`

	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, want))
	require.NoError(t, ls.DoString(script))
}

func TestInboundMessageSetters(t *testing.T) {
	want := &event.InboundMessage{
		Mailboxes: []string{"mb1", "mb2"},
		From:      &mail.Address{Name: "name1", Address: "addr1"},
		To: []*mail.Address{
			{Name: "name2", Address: "addr2"},
			{Name: "name3", Address: "addr3"},
		},
		Subject: "subj1",
	}
	script := `
		assert(msg, "msg should not be nil")

		msg.mailboxes = {"mb1", "mb2"}
		msg.subject = "subj1"
		msg.from = address.new("name1", "addr1")
		msg.to = { address.new("name2", "addr2"), address.new("name3", "addr3") }
	`

	got := &event.InboundMessage{}
	ls, _ := test.NewLuaState()
	registerInboundMessageType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapInboundMessage(ls, got))
	require.NoError(t, ls.DoString(script))

	assert.Equal(t, want, got)
}
