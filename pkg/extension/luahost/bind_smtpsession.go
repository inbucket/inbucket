package luahost

import (
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const smtpSessionName = "smtp_session"

func registerSMTPSessionType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(smtpSessionName)
	ls.SetGlobal(smtpSessionName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newSMTPSession))

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(smtpSessionIndex))
}

func newSMTPSession(ls *lua.LState) int {
	val := &event.SMTPSession{}
	ud := wrapSMTPSession(ls, val)
	ls.Push(ud)

	return 1
}

func wrapSMTPSession(ls *lua.LState, val *event.SMTPSession) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(smtpSessionName))

	return ud
}

// Checks there is an SMTPSession at stack position `pos`, else throws Lua error.
func checkSMTPSession(ls *lua.LState, pos int) *event.SMTPSession {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(*event.SMTPSession); ok {
		return v
	}
	ls.ArgError(pos, smtpSessionName+" expected")
	return nil
}

// Gets a field value from SMTPSession user object.  This emulates a Lua table,
// allowing `msg.subject` instead of a Lua object syntax of `msg:subject()`.
func smtpSessionIndex(ls *lua.LState) int {
	session := checkSMTPSession(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "from":
		ls.Push(wrapMailAddress(ls, session.From))
	case "to":
		lt := &lua.LTable{}
		for _, v := range session.To {
			addr := v
			lt.Append(wrapMailAddress(ls, addr))
		}
		ls.Push(lt)
	case "remote_addr":
		ls.Push(lua.LString(session.RemoteAddr))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}
