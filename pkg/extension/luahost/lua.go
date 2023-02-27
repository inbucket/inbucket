package luahost

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/extension"
	"github.com/inbucket/inbucket/pkg/extension/event"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// Host of Lua extensions.
type Host struct {
	Functions  []string // Functions detected in lua script.
	extHost    *extension.Host
	pool       *statePool
	logContext zerolog.Context
}

// New constructs a new Lua Host, pre-compiling the source.
func New(conf config.Lua, extHost *extension.Host) (*Host, error) {
	scriptPath := conf.Path
	if scriptPath == "" {
		return nil, nil
	}

	logContext := log.With().Str("module", "lua")
	logger := logContext.Str("phase", "startup").Str("path", scriptPath).Logger()

	// Pre-load, parse, and compile script.
	if fi, err := os.Stat(scriptPath); err != nil {
		logger.Info().Msg("Script file not found")
		return nil, nil
	} else if fi.IsDir() {
		return nil, fmt.Errorf("Lua script %v is a directory", scriptPath)
	}

	logger.Info().Msg("Loading script")
	file, err := os.Open(scriptPath)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return NewFromReader(extHost, bufio.NewReader(file), scriptPath)
}

// NewFromReader constructs a new Lua Host, loading Lua source from the provided reader.
// The provided path is used in logging and error messages.
func NewFromReader(extHost *extension.Host, r io.Reader, path string) (*Host, error) {
	logContext := log.With().Str("module", "lua")
	logger := logContext.Str("phase", "startup").Str("path", path).Logger()

	// Pre-parse, and compile script.
	chunk, err := parse.Parse(r, path)
	if err != nil {
		return nil, err
	}
	proto, err := lua.Compile(chunk, path)
	if err != nil {
		return nil, err
	}

	// Build the pool and confirm LState is retrievable.
	pool := newStatePool(proto)
	h := &Host{extHost: extHost, pool: pool, logContext: logContext}
	if ls, err := pool.getState(); err == nil {
		h.wireFunctions(logger, ls)

		// State creation works, put it back.
		pool.putState(ls)
	} else {
		return nil, err
	}

	return h, nil
}

// CreateChannel creates a channel and places it into the named global variable
// in newly created LStates.
func (h *Host) CreateChannel(name string) chan lua.LValue {
	return h.pool.createChannel(name)
}

// Detects global lua event listener functions and wires them up.
func (h *Host) wireFunctions(logger zerolog.Logger, ls *lua.LState) {
	detectFn := func(name string) bool {
		lval := ls.GetGlobal(name)
		switch lval.Type() {
		case lua.LTFunction:
			logger.Debug().Msgf("Detected %q function", name)
			h.Functions = append(h.Functions, name)
			return true
		case lua.LTNil:
			logger.Debug().Msgf("Did not detect %q function", name)
		default:
			logger.Fatal().Msgf("Found global named %q, but was a %v instead of a function",
				name, lval.Type().String())
		}

		return false
	}

	ib, err := getInbucket(ls)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get inbucket global")
	}

	events := h.extHost.Events
	const listenerName string = "lua"

	if detectFn(afterMessageDeletedFnName) {
		events.AfterMessageDeleted.AddListener(listenerName, h.handleAfterMessageDeleted)
	}
	if detectFn(beforeMailAcceptedFnName) {
		events.BeforeMailAccepted.AddListener(listenerName, h.handleBeforeMailAccepted)
	}

	if ib.After.MessageStored.Type() == lua.LTFunction {
		events.AfterMessageStored.AddListener(listenerName, h.handleAfterMessageStored)
	}
}

func (h *Host) handleAfterMessageDeleted(msg event.MessageMetadata) {
	logger, ls, lfunc, ok := h.prepareFuncCall(afterMessageDeletedFnName)
	if !ok {
		return
	}
	defer h.pool.putState(ls)

	// Call lua function.
	logger.Debug().Msgf("Calling Lua function with %+v", msg)
	if err := ls.CallByParam(
		lua.P{Fn: lfunc, NRet: 0, Protect: true},
		wrapMessageMetadata(ls, &msg),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
	}
}

func (h *Host) handleAfterMessageStored(msg event.MessageMetadata) {
	logger, ls, ib, ok := h.prepareInbucketFuncCall("after.message_stored")
	if !ok {
		return
	}
	defer h.pool.putState(ls)

	// Call lua function.
	logger.Debug().Msgf("Calling Lua function with %+v", msg)
	if err := ls.CallByParam(
		lua.P{Fn: ib.After.MessageStored, NRet: 0, Protect: true},
		wrapMessageMetadata(ls, &msg),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
	}
}

func (h *Host) handleBeforeMailAccepted(addr event.AddressParts) *bool {
	logger, ls, lfunc, ok := h.prepareFuncCall(beforeMailAcceptedFnName)
	if !ok {
		return nil
	}
	defer h.pool.putState(ls)

	logger.Debug().Msgf("Calling Lua function with %+v", addr)
	if err := ls.CallByParam(
		lua.P{Fn: lfunc, NRet: 1, Protect: true},
		lua.LString(addr.Local),
		lua.LString(addr.Domain),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
		return nil
	}

	lval := ls.Get(1)
	ls.Pop(1)
	logger.Debug().Msgf("Lua function returned %q (%v)", lval, lval.Type().String())

	if lval.Type() == lua.LTNil {
		return nil
	}

	result := true
	if lua.LVIsFalse(lval) {
		result = false
	}

	return &result
}

// Common preparation for calling Lua functions.
func (h *Host) prepareFuncCall(funcName string) (logger zerolog.Logger, ls *lua.LState, lfunc lua.LValue, ok bool) {
	logger = h.logContext.Str("event", funcName).Logger()

	ls, err := h.pool.getState()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get Lua state instance from pool")
		return logger, nil, nil, false
	}

	lfunc = ls.GetGlobal(funcName)
	if lfunc.Type() != lua.LTFunction {
		logger.Error().Msgf("global %q is no longer a function", funcName)
		return logger, nil, nil, false
	}

	return logger, ls, lfunc, true
}

// Common preparation for calling Lua functions.
func (h *Host) prepareInbucketFuncCall(funcName string) (logger zerolog.Logger, ls *lua.LState, ib *Inbucket, ok bool) {
	logger = h.logContext.Str("event", funcName).Logger()

	ls, err := h.pool.getState()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get Lua state instance from pool")
		return logger, nil, nil, false
	}

	ib, err = getInbucket(ls)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to obtain Lua inbucket object")
		return logger, nil, nil, false
	}

	return logger, ls, ib, true
}
