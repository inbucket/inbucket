package luahost

import (
	"net/mail"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestMailAddressGetters(t *testing.T) {
	want := &mail.Address{
		Name:    "Roberto I",
		Address: "ri@example.com",
	}
	script := `
		assert(addr, "addr should not be nil")

		want = "Roberto I"
		got = addr.name
		assert(got == want, string.format("got name %q, want %q", got, want))

		want = "ri@example.com"
		got = addr.address
		assert(got == want, string.format("got address %q, want %q", got, want))
	`

	ls := lua.NewState()
	registerMailAddressType(ls)
	ls.SetGlobal("addr", wrapMailAddress(ls, want))
	require.NoError(t, ls.DoString(script))
}

func TestMailAddressSetters(t *testing.T) {
	want := &mail.Address{
		Name:    "Roberto I",
		Address: "ri@example.com",
	}
	script := `
		assert(addr, "addr should not be nil")

		addr.name = "Roberto I"
		addr.address = "ri@example.com"
	`

	got := &mail.Address{}
	ls := lua.NewState()
	registerMailAddressType(ls)
	ls.SetGlobal("addr", wrapMailAddress(ls, got))
	require.NoError(t, ls.DoString(script))

	assert.Equal(t, want, got)
}
