package luahost_test

import (
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/extension/luahost"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

// LuaInit holds useful test globals.
const LuaInit = `
	local logger = require("logger")

	async = false
	test_ok = true

	-- Sends marks tests as failed instead of erroring when enabled.
	function assert_async(value, message)
		if not value then
			if async then
				logger.error(message, {from = "assert_async"})
				test_ok = false
			else
				error(message)
			end
		end
	end

	-- Verifies plain values and list-style tables.
	function assert_eq(got, want)
		if type(got) == "table" and type(want) == "table" then
			assert_async(#got == #want, string.format("got %d elements, wanted %d", #got, #want))

			for i, gotv in ipairs(got) do
				local wantv = want[i]
				assert_eq(gotv, wantv, "got[%d] = %q, wanted %q", gotv, wantv)
			end

			return
		end

		assert_async(got == want, string.format("got %q, wanted %q", got, want))
	end

	-- Verifies string want contains string got.
	function assert_contains(got, want)
		assert_async(string.find(got, want),
			string.format("got %q, wanted it to contain %q", got, want))
	end
`

var consoleLogger = zerolog.New(zerolog.NewConsoleWriter())

func TestEmptyScript(t *testing.T) {
	script := ""
	extHost := extension.NewHost()

	_, err := luahost.NewFromReader(consoleLogger, extHost, strings.NewReader(script), "test.lua")
	require.NoError(t, err)
}

func TestLogger(t *testing.T) {
	script := `
		local logger = require("logger")
		logger.info("_test log entry_", {})
	`

	extHost := extension.NewHost()
	output := &strings.Builder{}
	logger := zerolog.New(output)

	_, err := luahost.NewFromReader(logger, extHost, strings.NewReader(script), "test.lua")
	require.NoError(t, err)

	assert.Contains(t, output.String(), "_test log entry_")
}

func TestAfterMessageDeleted(t *testing.T) {
	// Register lua event listener, setup notify channel.
	script := `
		async = true

		function inbucket.after.message_deleted(msg)
			-- Full message bindings tested elsewhere.
			assert_eq(msg.mailbox, "mb1")
			assert_eq(msg.id, "id1")
			notify:send(test_ok)
		end
	`
	extHost := extension.NewHost()
	luaHost, err := luahost.NewFromReader(consoleLogger, extHost, strings.NewReader(LuaInit+script), "test.lua")
	require.NoError(t, err)
	notify := luaHost.CreateChannel("notify")

	// Send event, check channel response is true.
	msg := &event.MessageMetadata{
		Mailbox: "mb1",
		ID:      "id1",
		From:    &mail.Address{Name: "name1", Address: "addr1"},
		To:      []*mail.Address{{Name: "name2", Address: "addr2"}},
		Date:    time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC),
		Subject: "subj1",
		Size:    42,
	}
	extHost.Events.AfterMessageDeleted.Emit(msg)
	assertNotified(t, notify)
}

func TestAfterMessageStored(t *testing.T) {
	// Register lua event listener, setup notify channel.
	script := `
		async = true

		function inbucket.after.message_stored(msg)
			-- Full message bindings tested elsewhere.
			assert_eq(msg.mailbox, "mb1")
			assert_eq(msg.id, "id1")
			notify:send(test_ok)
		end
	`
	extHost := extension.NewHost()
	luaHost, err := luahost.NewFromReader(consoleLogger, extHost, strings.NewReader(LuaInit+script), "test.lua")
	require.NoError(t, err)
	notify := luaHost.CreateChannel("notify")

	// Send event, check channel response is true.
	msg := &event.MessageMetadata{
		Mailbox: "mb1",
		ID:      "id1",
		From:    &mail.Address{Name: "name1", Address: "addr1"},
		To:      []*mail.Address{{Name: "name2", Address: "addr2"}},
		Date:    time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC),
		Subject: "subj1",
		Size:    42,
	}
	extHost.Events.AfterMessageStored.Emit(msg)
	assertNotified(t, notify)
}

func TestBeforeMailAccepted(t *testing.T) {
	// Register lua event listener.
	script := `
		function inbucket.before.mail_accepted(localpart, domain)
			return localpart == "from" and domain == "test"
		end
	`
	extHost := extension.NewHost()
	_, err := luahost.NewFromReader(consoleLogger, extHost, strings.NewReader(script), "test.lua")
	require.NoError(t, err)

	// Send event to be accepted.
	addr := &event.AddressParts{Local: "from", Domain: "test"}
	got := extHost.Events.BeforeMailAccepted.Emit(addr)
	want := true
	require.NotNil(t, got, "Expected result from Emit()")
	if *got != want {
		t.Errorf("Got %v, wanted %v for addr %v", *got, want, addr)
	}

	// Send event to be denied.
	addr = &event.AddressParts{Local: "reject", Domain: "me"}
	got = extHost.Events.BeforeMailAccepted.Emit(addr)
	want = false
	require.NotNil(t, got, "Expected result from Emit()")
	if *got != want {
		t.Errorf("Got %v, wanted %v for addr %v", *got, want, addr)
	}
}

func TestBeforeMessageStored(t *testing.T) {
	// Event to send.
	msg := event.InboundMessage{
		Mailboxes: []string{"one", "two"},
		From:      mail.Address{Name: "From Name", Address: "from@example.com"},
		To: []mail.Address{
			{Name: "To1 Name", Address: "to1@example.com"},
			{Name: "To2 Name", Address: "to2@example.com"},
		},
		Subject: "inbound subj",
		Size:    42,
	}

	// Register lua event listener.
	script := `
		async = true

		function inbucket.before.message_stored(msg)
			-- Verify incoming values.
			assert_eq(msg.mailboxes, {"one", "two"})
			assert_eq(msg.from.name, "From Name")
			assert_eq(msg.from.address, "from@example.com")
			assert_eq(2, #msg.to)
			assert_eq(msg.to[1].name, "To1 Name")
			assert_eq(msg.to[1].address, "to1@example.com")
			assert_eq(msg.to[2].name, "To2 Name")
			assert_eq(msg.to[2].address, "to2@example.com")
			assert_eq(msg.subject, "inbound subj")
			assert_eq(msg.size, 42)
			notify:send(test_ok)

			-- Generate response.
			res = inbound_message.new()
			res.mailboxes = {"resone", "restwo"}
			res.from = address.new("Res From", "res@example.com")
			res.to = {
				address.new("To1 Res", "res1@example.com"),
				address.new("To2 Res", "res2@example.com"),
			}
			res.subject = "res subj"
			return res
		end
	`
	extHost := extension.NewHost()
	luaHost, err := luahost.NewFromReader(consoleLogger, extHost, strings.NewReader(LuaInit+script), "test.lua")
	require.NoError(t, err)
	notify := luaHost.CreateChannel("notify")

	// Send event to be accepted.
	got := extHost.Events.BeforeMessageStored.Emit(&msg)
	require.NotNil(t, got, "Expected result from Emit()")

	// Verify Lua assertions passed.
	assertNotified(t, notify)

	// Verify response values.
	want := &event.InboundMessage{
		Mailboxes: []string{"resone", "restwo"},
		From:      mail.Address{Name: "Res From", Address: "res@example.com"},
		To: []mail.Address{
			{Name: "To1 Res", Address: "res1@example.com"},
			{Name: "To2 Res", Address: "res2@example.com"},
		},
		Subject: "res subj",
		Size:    0,
	}
	assert.Equal(t, want, got, "Response InboundMessage did not match")
}

func assertNotified(t *testing.T, notify chan lua.LValue) {
	t.Helper()
	select {
	case reslv := <-notify:
		// Lua function received event.
		if lua.LVIsFalse(reslv) {
			t.Error("Lua responsed with false, wanted true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Lua did not respond to event within timeout")
	}
}
