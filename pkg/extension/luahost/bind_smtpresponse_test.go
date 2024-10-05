package luahost

import (
	"testing"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/inbucket/inbucket/v3/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMTPResponseConstructors(t *testing.T) {
	check := func(script string, want event.SMTPResponse) {
		t.Helper()
		ls, _ := test.NewLuaState()
		registerSMTPResponseType(ls)
		require.NoError(t, ls.DoString(script))

		got, err := unwrapSMTPResponse(ls.Get(-1))
		require.NoError(t, err)
		assert.Equal(t, &want, got)
	}

	check("return smtp.defer()", event.SMTPResponse{Action: event.ActionDefer})
	check("return smtp.allow()", event.SMTPResponse{Action: event.ActionAllow})

	// Verify deny() has default code & msg.
	check("return smtp.deny()", event.SMTPResponse{
		Action:    event.ActionDeny,
		ErrorCode: 550,
		ErrorMsg:  "Mail denied by policy",
	})

	// Verify defaults can be overridden.
	check("return smtp.deny(123, 'bacon')", event.SMTPResponse{
		Action:    event.ActionDeny,
		ErrorCode: 123,
		ErrorMsg:  "bacon",
	})
}
