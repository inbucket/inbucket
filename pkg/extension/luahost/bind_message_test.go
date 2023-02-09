package luahost

import (
	"net/mail"
	"testing"
	"time"

	"github.com/inbucket/inbucket/pkg/extension/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestMessageMetadataGetters(t *testing.T) {
	want := &event.MessageMetadata{
		Mailbox: "mb1",
		ID:      "id1",
		From:    &mail.Address{Name: "name1", Address: "addr1"},
		To:      []*mail.Address{{Name: "name2", Address: "addr2"}},
		Date:    time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC),
		Subject: "subj1",
		Size:    42,
	}
	script := `
		assert(msg, "msg should not be nil")

		function assert_eq(got, want)
			assert(got == want, string.format("got name %q, wanted %q", got, want))
		end

		assert_eq(msg.mailbox, "mb1")
		assert_eq(msg.id, "id1")
		assert_eq(msg.subject, "subj1")
		assert_eq(msg.size, 42)

		assert_eq(msg.from.name, "name1")
		assert_eq(msg.from.address, "addr1")

		assert_eq(table.getn(msg.to), 1)
		assert_eq(msg.to[1].name, "name2")
		assert_eq(msg.to[1].address, "addr2")

		assert_eq(msg.date, 981173106)
	`

	ls := lua.NewState()
	registerMessageMetadataType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapMessageMetadata(ls, want))
	require.NoError(t, ls.DoString(script))
}

func TestMessageMetadataSetters(t *testing.T) {
	want := &event.MessageMetadata{
		Mailbox: "mb1",
		ID:      "id1",
		From:    &mail.Address{Name: "name1", Address: "addr1"},
		To:      []*mail.Address{{Name: "name2", Address: "addr2"}},
		Date:    time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC),
		Subject: "subj1",
		Size:    42,
	}
	script := `
		assert(msg, "msg should not be nil")

		msg.mailbox = "mb1"
		msg.id = "id1"
		msg.subject = "subj1"
		msg.size = 42

		msg.from = address.new("name1", "addr1")
		msg.to = { address.new("name2", "addr2") }

		msg.date = 981173106
	`

	got := &event.MessageMetadata{}
	ls := lua.NewState()
	registerMessageMetadataType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapMessageMetadata(ls, got))
	require.NoError(t, ls.DoString(script))

	// Timezones will cause a naive comparison to fail.
	assert.Equal(t, want.Date.Unix(), got.Date.Unix())
	now := time.Now()
	want.Date = now
	got.Date = now

	assert.Equal(t, want, got)
}
