package luahost

import (
	lua "github.com/yuin/gopher-lua"
)

const inbucketConfigName = "inbucket_config"

// InbucketConfig holds Inbucket settings that a Lua script may control.
//
// Values are captured once, from the first LState after the script runs.  The
// script is re-run for each additional pooled LState, but its later config
// settings are ignored: scripts must set config deterministically at the top
// level.
type InbucketConfig struct {
	// DecodeHeaders names additional message headers to decode; the default
	// set (Date, Subject, Sender, From, To, CC, BCC) is always decoded.
	// Decoded headers are exposed to scripts via the `inbound_message.header`
	// field.  Only headers listed here and in the default set are decoded,
	// keeping the cost proportional to what the script asked for.
	DecodeHeaders []string
}

func registerInbucketConfigType(ls *lua.LState) {
	// inbucket.config type.
	mt := ls.NewTypeMetatable(inbucketConfigName)
	ls.SetField(mt, "__index", ls.NewFunction(inbucketConfigIndex))
	ls.SetField(mt, "__newindex", ls.NewFunction(inbucketConfigNewIndex))
}

func wrapInbucketConfig(ls *lua.LState, val *InbucketConfig) *lua.LUserData {
	ud := ls.NewUserData()
	ud.Value = val
	ls.SetMetatable(ud, ls.GetTypeMetatable(inbucketConfigName))

	return ud
}

func checkInbucketConfig(ls *lua.LState, pos int) *InbucketConfig {
	ud := ls.CheckUserData(pos)
	if val, ok := ud.Value.(*InbucketConfig); ok {
		return val
	}
	ls.ArgError(pos, inbucketConfigName+" expected")
	return nil
}

// inbucket.config getter.
func inbucketConfigIndex(ls *lua.LState) int {
	c := checkInbucketConfig(ls, 1)
	field := ls.CheckString(2)

	// Push the requested field's value onto the stack.
	switch field {
	case "decode_headers":
		if len(c.DecodeHeaders) == 0 {
			ls.Push(lua.LNil)
			return 1
		}
		lt := &lua.LTable{}
		for _, v := range c.DecodeHeaders {
			lt.Append(lua.LString(v))
		}
		ls.Push(lt)
	default:
		// Unknown field.
		ls.Push(lua.LNil)
	}

	return 1
}

// inbucket.config setter.
func inbucketConfigNewIndex(ls *lua.LState) int {
	c := checkInbucketConfig(ls, 1)
	index := ls.CheckString(2)

	switch index {
	case "decode_headers":
		lt := ls.CheckTable(3)
		headers := make([]string, 0, lt.Len())
		for i := 1; i <= lt.Len(); i++ {
			value := lt.RawGetInt(i)
			if value.Type() != lua.LTString {
				ls.RaiseError("decode_headers[%d] must be a string, got %s", i, value.Type().String())
			}
			headers = append(headers, value.String())
		}
		c.DecodeHeaders = headers
	default:
		ls.RaiseError("invalid inbucket.config index %q", index)
	}

	return 0
}
