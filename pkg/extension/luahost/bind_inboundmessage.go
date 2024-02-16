package luahost

import (
	"fmt"
	"net/mail"

	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	lua "github.com/yuin/gopher-lua"
)

const inboundMessageName = "inbound_message"

func registerInboundMessageType(ls *lua.LState) {
	mt := ls.NewTypeMetatable(inboundMessageName)
	ls.SetGlobal(inboundMessageName, mt)

	// Static attributes.
	ls.SetField(mt, "new", ls.NewFunction(newInboundMessage))

	// Methods.
	ls.SetField(mt, "__index", ls.NewFunction(inboundMessageIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inboundMessageNewIndex))
}

func newInboundMessage(ls *lua.LState) int {
	val := &event.InboundMessage{}
	ud := wrapInboundMessage(ls, val)
	ls.Push(ud)

	return 1
}

func wrapInboundMessage(ls *lua.LState, val *event.InboundMessage) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inboundMessageName))

	return ud
}

// Checks there is an InboundMessage at stack position `pos`, else throws Lua error.
func checkInboundMessage(ls *lua.LState, pos int) *event.InboundMessage {
	ud := ls.CheckUserData(pos)
	if v, ok := ud.Value.(*event.InboundMessage); ok {
		return v
	}
	ls.ArgError(pos, inboundMessageName+" expected")
	return nil
}

func unwrapInboundMessage(lv lua.LValue) (*event.InboundMessage, error) {
	if ud, ok := lv.(*lua.LUserData); ok {
		if v, ok := ud.Value.(*event.InboundMessage); ok {
			return v, nil
		}
	}

	return nil, fmt.Errorf("expected InboundMessage, got %q", lv.Type().String())
}

// Gets a field value from InboundMessage user object.  This emulates a Lua table,
// allowing `msg.subject` instead of a Lua object syntax of `msg:subject()`.
func inboundMessageIndex(ls *lua.LState) int {
	m := checkInboundMessage(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "mailboxes":
		lt := &lua.LTable{}
		for _, v := range m.Mailboxes {
			lt.Append(lua.LString(v))
		}
		ls.Push(lt)
	case "from":
		ls.Push(wrapMailAddress(ls, m.From))
	case "to":
		lt := &lua.LTable{}
		for _, v := range m.To {
			addr := v
			lt.Append(wrapMailAddress(ls, addr))
		}
		ls.Push(lt)
	case "subject":
		ls.Push(lua.LString(m.Subject))
	case "size":
		ls.Push(lua.LNumber(m.Size))
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// Sets a field value on InboundMessage user object.  This emulates a Lua table,
// allowing `msg.subject = x` instead of a Lua object syntax of `msg:subject(x)`.
func inboundMessageNewIndex(ls *lua.LState) int {
	m := checkInboundMessage(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "mailboxes":
		lt := ls.CheckTable(3)
		mailboxes := make([]string, 0, 16)
		lt.ForEach(func(k, lv lua.LValue) {
			if mb, ok := lv.(lua.LString); ok {
				mailboxes = append(mailboxes, string(mb))
			}
		})
		m.Mailboxes = mailboxes
	case "from":
		m.From = checkMailAddress(ls, 3)
	case "to":
		lt := ls.CheckTable(3)
		to := make([]*mail.Address, 0, 16)
		lt.ForEach(func(k, lv lua.LValue) {
			if ud, ok := lv.(*lua.LUserData); ok {
				// TODO should fail if wrong type + test.
				if entry, ok := unwrapMailAddress(ud); ok {
					to = append(to, entry)
				}
			}
		})
		m.To = to
	case "subject":
		m.Subject = ls.CheckString(3)
	case "size":
		ls.RaiseError("size is read-only")
	default:
		ls.RaiseError("invalid index %q", index)
	}

	return 0
}
