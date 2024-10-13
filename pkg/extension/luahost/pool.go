package luahost

import (
	"net/http"
	"sync"

	"github.com/cjoudrey/gluahttp"
	"github.com/cosmotek/loguago"
	json "github.com/inbucket/gopher-json"
	"github.com/rs/zerolog"
	lua "github.com/yuin/gopher-lua"
)

type statePool struct {
	sync.Mutex
	funcProto *lua.FunctionProto         // Compiled lua.
	states    []*lua.LState              // Pool of available LStates.
	channels  map[string]chan lua.LValue // Global interop channels.
	logger    zerolog.Logger             // Logger exported to Lua scripts.
}

func newStatePool(logger zerolog.Logger, funcProto *lua.FunctionProto) *statePool {
	return &statePool{
		funcProto: funcProto,
		channels:  make(map[string]chan lua.LValue),
		logger:    logger,
	}
}

// newState creates a new LState and configures it. Lock must be held.
func (lp *statePool) newState() (*lua.LState, error) {
	ls := lua.NewState()

	logger := loguago.NewLogger(lp.logger)

	// Load supplemental native modules.
	ls.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
	ls.PreloadModule("json", json.Loader)
	ls.PreloadModule("logger", logger.Loader)

	// Setup channels.
	for name, ch := range lp.channels {
		ls.SetGlobal(name, lua.LChannel(ch))
	}

	// Register custom types.
	registerInboundMessageType(ls)
	registerInbucketTypes(ls)
	registerMailAddressType(ls)
	registerMessageMetadataType(ls)
	registerSMTPResponseType(ls)
	registerSMTPSessionType(ls)

	// Run compiled script.
	ls.Push(ls.NewFunctionFromProto(lp.funcProto))
	if err := ls.PCall(0, lua.MultRet, nil); err != nil {
		return nil, err
	}

	return ls, nil
}

// getState returns a free LState, or creates a new one.
func (lp *statePool) getState() (*lua.LState, error) {
	lp.Lock()
	defer lp.Unlock()

	ln := len(lp.states)
	if ln == 0 {
		return lp.newState()
	}

	state := lp.states[ln-1]
	lp.states = lp.states[0 : ln-1]

	return state, nil
}

// putState returns the LState to the pool.
func (lp *statePool) putState(state *lua.LState) {
	if state.IsClosed() {
		return
	}

	// Clear stack.
	state.Pop(state.GetTop())

	lp.Lock()
	defer lp.Unlock()

	lp.states = append(lp.states, state)
}

// createChannel creates a new channel, which will become a global variable in
// newly created LStates.  We also destroy any pooled states.
//
// Warning: There may still be checked out LStates that will not have the value
// set, which could be put back into the pool.
func (lp *statePool) createChannel(name string) chan lua.LValue {
	lp.Lock()
	defer lp.Unlock()

	ch := make(chan lua.LValue, 10)
	lp.channels[name] = ch

	// Flush state pool.
	for _, s := range lp.states {
		s.Close()
	}
	lp.states = lp.states[:0]

	return ch
}
