package luahost

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInbucketConfigDecodeHeaders(t *testing.T) {
	script := `
		assert(inbucket, "inbucket should not be nil")
		assert(inbucket.config, "inbucket.config should not be nil")

		-- Verify decode_headers starts off nil.
		assert(inbucket.config.decode_headers == nil,
			"decode_headers should be nil")

		-- Set and read back.
		inbucket.config.decode_headers = {
			"X-Spam-Status",
			"Authentication-Results",
		}
		assert_eq(inbucket.config.decode_headers,
			{ "X-Spam-Status", "Authentication-Results" })

		-- Unknown fields read as nil.
		assert(inbucket.config.bogus == nil, "unknown config field should be nil")
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	require.NoError(t, ls.DoString(script))

	ib, err := getInbucket(ls)
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"X-Spam-Status", "Authentication-Results"},
		ib.Config.DecodeHeaders)
}

func TestInbucketConfigDecodeHeadersEmpty(t *testing.T) {
	script := `
		inbucket.config.decode_headers = {}
		assert(inbucket.config.decode_headers == nil,
			"empty decode_headers should read as nil")
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	require.NoError(t, ls.DoString(script))

	ib, err := getInbucket(ls)
	require.NoError(t, err)
	assert.Empty(t, ib.Config.DecodeHeaders)
}

func TestInbucketConfigInvalidIndex(t *testing.T) {
	script := `
		inbucket.config.bogus = true
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	err := ls.DoString(script)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid inbucket.config index")
}

func TestInbucketConfigDecodeHeadersInvalidEntry(t *testing.T) {
	script := `
		inbucket.config.decode_headers = { "X-Spam-Status", 42 }
	`

	ls, _ := test.NewLuaState()
	registerInbucketTypes(ls)
	err := ls.DoString(script)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `decode_headers[2] must be a string`)
}
