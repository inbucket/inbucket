package luahost

import (
	"net/textproto"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderGetters(t *testing.T) {
	header := textproto.MIMEHeader{
		"Subject":                {"subj1"},
		"From":                   {"from@example.com"},
		"X-Spam-Status":          {"Yes", "second value ignored"},
		"X-Exotic+Key.Keep":      {"exotic1"},
		"X-Registered-But-Unset": {},
	}
	script := `
		assert(header, "header should not be nil")

		-- Subject matches msg.subject.
		assert_eq(header.subject, "subj1")
		assert_eq(header.Subject, "subj1")
		assert_eq(header["SUBJECT"], "subj1")

		-- Registered extra header, case-insensitive.
		assert_eq(header["X-Spam-Status"], "Yes")
		assert_eq(header["x-spam-status"], "Yes")

		-- First value wins when set more than once.
		assert_eq(header.from, "from@example.com")

		-- Keys enmime's canonicalization rules preserve exactly.
		assert_eq(header["X-Exotic+Key.Keep"], "exotic1")
		assert_eq(header["x-exotic+key.keep"], "exotic1")

		-- Registered but unset, and unregistered headers, are nil.
		assert(header["X-Registered-But-Unset"] == nil,
			"unset header should be nil")
		assert(header["X-Not-Registered"] == nil,
			"unregistered header should be nil")
	`

	ls, _ := test.NewLuaState()
	registerHeaderType(ls)
	ls.SetGlobal("header", wrapHeader(ls, header))
	require.NoError(t, ls.DoString(script))
}

func TestHeaderReadOnly(t *testing.T) {
	header := textproto.MIMEHeader{"Subject": {"subj1"}}
	script := `
		header["Subject"] = "rewritten"
	`

	ls, _ := test.NewLuaState()
	registerHeaderType(ls)
	ls.SetGlobal("header", wrapHeader(ls, header))
	err := ls.DoString(script)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `header "Subject" is read-only`)

	// Value was not modified.
	assert.Equal(t, []string{"subj1"}, header["Subject"])
}
