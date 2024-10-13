package luahost

import (
	"net/mail"
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestSMTPSessionGetters(t *testing.T) {
	want := &event.SMTPSession{
		From: &mail.Address{Name: "name1", Address: "addr1"},
		To: []*mail.Address{
			{Name: "name2", Address: "addr2"},
			{Name: "name3", Address: "addr3"},
		},
	}
	script := `
		assert(msg, "msg should not be nil")

		assert_eq(msg.from.name, "name1", "from.name")
		assert_eq(msg.from.address, "addr1", "from.address")

		assert_eq(#msg.to, 2, "#msg.to")
		assert_eq(msg.to[1].name, "name2", "to[1].name")
		assert_eq(msg.to[1].address, "addr2", "to[1].address")
		assert_eq(msg.to[2].name, "name3", "to[2].name")
		assert_eq(msg.to[2].address, "addr3", "to[2].address")
	`

	ls, _ := test.NewLuaState()
	registerSMTPSessionType(ls)
	registerMailAddressType(ls)
	ls.SetGlobal("msg", wrapSMTPSession(ls, want))
	require.NoError(t, ls.DoString(script))
}
