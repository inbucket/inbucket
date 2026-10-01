package luahost

import (
	"net/textproto"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestMIMEHeaderGetters(t *testing.T) {
	spamdResult := "default: False [7.54 / 6.00];" +
		" DMARC_POLICY_REJECT(2.00)[example.com : SPF not aligned (relaxed), DKIM not aligned (relaxed),reject];" +
		" R_DKIM_REJECT(1.00)[example.org:s=sel];" +
		" FORGED_SENDER(0.30)[from@example.com,bounces+1=u1@send.example.org];" +
		" MIME_TRACE(0.00)[0:+,1:+,2:~];" +
		" ARC_NA(0.00)[]"
	want := textproto.MIMEHeader{
		"Subject":        {"subj1"},
		"X-Spam-Status":  {"Yes, score=7.54"},
		"X-Spamd-Result": {spamdResult, "default: False [0.00 / 6.00]"},
	}
	script := `
		assert(header, "header should not be nil")

		assert_eq(header["Subject"], "subj1", "header[Subject]")
		assert_eq(header["X-Spam-Status"], "Yes, score=7.54", "header[X-Spam-Status]")
		assert_eq(header["X-Spamd-Result"], spamd_result, "header[X-Spamd-Result]")
		assert_eq(header["x-spamd-result"], spamd_result, "header[x-spamd-result]")
		assert_eq(header["X-Missing"], nil, "header[X-Missing]")
	`

	ls, _ := test.NewLuaState()
	registerMIMEHeaderType(ls)
	ls.SetGlobal("header", wrapMIMEHeader(ls, want))
	ls.SetGlobal("spamd_result", lua.LString(spamdResult))
	require.NoError(t, ls.DoString(script))
}

func TestMIMEHeaderNilGetters(t *testing.T) {
	script := `
		assert(header, "header should not be nil")

		assert_eq(header["Subject"], nil, "header[Subject]")
	`

	ls, _ := test.NewLuaState()
	registerMIMEHeaderType(ls)
	ls.SetGlobal("header", wrapMIMEHeader(ls, nil))
	require.NoError(t, ls.DoString(script))
}

func TestMIMEHeaderReadOnly(t *testing.T) {
	ls, _ := test.NewLuaState()
	registerMIMEHeaderType(ls)
	ls.SetGlobal("header", wrapMIMEHeader(ls, textproto.MIMEHeader{}))

	err := ls.DoString(`header["X-Foo"] = "bar"`)
	require.Error(t, err, "assigning header[name] should fail")
	assert.Contains(t, err.Error(), `header "X-Foo" is read-only`)
}
