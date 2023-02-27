package luahost

import (
	"testing"

	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestInbucketAfterFuncs(t *testing.T) {
	script := `
		assert(inbucket, "inbucket should not be nil")
		assert(inbucket.after, "inbucket.after should not be nil")

		local fns = { "message_stored" }

		-- Verify functions start off nil.
		for i, name in ipairs(fns) do
			assert(inbucket.after[name] == nil, "after." .. name .. " should be nil")
		end

		-- Test function to track func calls made.
		local calls = {}
		local testfn = function(name)
			calls[name] = true
		end

		-- Set after functions, verify not nil, and call them.
		for i, name in ipairs(fns) do
			inbucket.after[name] = testfn
			assert(inbucket.after[name], "after." .. name .. " should not be nil")
			inbucket.after[name](name)
		end

		-- Verify functions were called.
		for i, name in ipairs(fns) do
			assert(calls[name], "after." .. name .. " should have been called")
		end
	`

	ls := lua.NewState()
	registerInbucketTypes(ls)
	require.NoError(t, ls.DoString(script))
}
