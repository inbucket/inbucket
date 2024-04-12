package test

import (
	"strings"
	"testing"
	"time"

	"github.com/cosmotek/loguago"
	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
)

// LuaInit holds useful test globals.
const LuaInit = `
	local logger = require("logger")

	async = false
	test_ok = true

	-- With async: marks tests as failed via test_ok, logs error.
	-- Without async: erroring when tests fail.
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

// NewLuaState creates a new Lua LState initialized with logging and the test helpers in `LuaInit`.
//
// Returns a pointer to the created LState and a string builder to hold the log output.
func NewLuaState() (*lua.LState, *strings.Builder) {
	output := &strings.Builder{}
	logger := loguago.NewLogger(zerolog.New(output))

	ls := lua.NewState()
	ls.PreloadModule("logger", logger.Loader)
	if err := ls.DoString(LuaInit); err != nil {
		panic(err)
	}

	return ls, output
}

// AssertNotified requires a truthy LValue on the notify channel.
func AssertNotified(t *testing.T, notify chan lua.LValue) {
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
