package luahost

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestInbucketAfterFuncs(t *testing.T) {
	// This Script registers each function and calls it.  No effort is made to use the arguments
	// that Inbucket expects, this is only to validate the inbucket.after data structure getters
	// and setters.
	script := `
		assert(inbucket, "inbucket should not be nil")
		assert(inbucket.after, "inbucket.after should not be nil")

		local fns = { "message_deleted", "message_stored" }

		-- Verify functions start off nil.
		for i, name in ipairs(fns) do
			assert(inbucket.after[name] == nil, "after." .. name .. " should be nil")
		end

		-- Test function to track func calls made, ensures no crossed wires.
		local calls = {}
		function makeTestFunc(create_name)
			return function(call_name)
				calls[create_name] = call_name
			end
		end

		-- Set after functions, verify not nil, and call them.
		for i, name in ipairs(fns) do
			inbucket.after[name] = makeTestFunc(name)
			assert(inbucket.after[name], "after." .. name .. " should not be nil")
		end

		-- Call each function.  Separate loop to verify final state in 'calls'.
		for i, name in ipairs(fns) do
			inbucket.after[name](name)
		end

		-- Verify functions were called.
		for i, name in ipairs(fns) do
			assert(calls[name], "after." .. name .. " should have been called")
			assert(calls[name] == name,
				string.format("after.%s was called with incorrect argument %s", name, calls[name]))
		end
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	require.NoError(t, ls.DoString(script))
}

func TestInbucketBeforeFuncs(t *testing.T) {
	// This Script registers each function and calls it.  No effort is made to use the arguments
	// that Inbucket expects, this is only to validate the inbucket.before data structure getters
	// and setters.
	script := `
		assert(inbucket, "inbucket should not be nil")
		assert(inbucket.before, "inbucket.before should not be nil")

		local fns = { "mail_from_accepted", "message_stored", "rcpt_to_accepted" }

		-- Verify functions start off nil.
		for i, name in ipairs(fns) do
			assert(inbucket.before[name] == nil, "before." .. name .. " should be nil")
		end

		-- Test function to track func calls made, ensures no crossed wires.
		local calls = {}
		function makeTestFunc(create_name)
			return function(call_name)
				calls[create_name] = call_name
			end
		end

		-- Set before functions, verify not nil, and call them.
		for i, name in ipairs(fns) do
			inbucket.before[name] = makeTestFunc(name)
			assert(inbucket.before[name], "before." .. name .. " should not be nil")
		end

		-- Call each function.  Separate loop to verify final state in 'calls'.
		for i, name in ipairs(fns) do
			inbucket.before[name](name)
		end

		-- Verify functions were called.
		for i, name in ipairs(fns) do
			assert(calls[name], "before." .. name .. " should have been called")
			assert(calls[name] == name,
				string.format("before.%s was called with incorrect argument %s", name, calls[name]))
		end
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	require.NoError(t, ls.DoString(script))
}
