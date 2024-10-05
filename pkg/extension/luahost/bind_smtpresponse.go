package luahost

import (
	"fmt"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const smtpResponseName = "smtp"

func registerSMTPResponseType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(smtpResponseName)
	ls.SetGlobal(smtpResponseName, mt)

	// Static attributes.
	ls.SetField(mt, "allow", ls.NewFunction(newSMTPResponse(event.ActionAllow)))
	ls.SetField(mt, "defer", ls.NewFunction(newSMTPResponse(event.ActionDefer)))
	ls.SetField(mt, "deny", ls.NewFunction(newSMTPResponse(event.ActionDeny)))
}

func newSMTPResponse(action int) func(*lua.LState) int {
	return func(ls *lua.LState) int {
		val := &event.SMTPResponse{Action: action}

		if action == event.ActionDeny {
			// Optionally accept error code and message.
			val.ErrorCode = ls.OptInt(1, 550)
			val.ErrorMsg = ls.OptString(2, "Mail denied by policy")
		}

		ud := wrapSMTPResponse(ls, val)
		ls.Push(ud)
		return 1
	}
}

func wrapSMTPResponse(ls *lua.LState, val *event.SMTPResponse) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(smtpResponseName))

	return ud
}

func unwrapSMTPResponse(lv lua.LValue) (*event.SMTPResponse, error) {
	if ud, ok := lv.(*lua.LUserData); ok {
		if v, ok := ud.Value.(*event.SMTPResponse); ok {
			return v, nil
		}
	}

	return nil, fmt.Errorf("expected SMTPResponse, got %q", lv.Type().String())
}
