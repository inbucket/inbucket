package luahost

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/inbucket/inbucket/v3/pkg/config"
	"github.com/inbucket/inbucket/v3/pkg/extension"
	"github.com/inbucket/inbucket/v3/pkg/extension/event"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

// ErrNoScript signals that the Lua script file was not present.
var ErrNoScript error = errors.New("no script file present")

// Host of Lua extensions.
type Host struct {
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
		return nil, ErrNoScript
	} else if fi.IsDir() {
		return nil, fmt.Errorf("lua script %v is a directory", scriptPath)
	}

	logger.Info().Msg("Loading script")
	file, err := os.Open(scriptPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return NewFromReader(logContext.Logger(), extHost, bufio.NewReader(file), scriptPath)
}

// NewFromReader constructs a new Lua Host, loading Lua source from the provided reader.
// The provided path is used in logging and error messages.
func NewFromReader(logger zerolog.Logger, extHost *extension.Host, r io.Reader, path string) (*Host, error) {
	startLogger := logger.With().Str("phase", "startup").Str("path", path).Logger()

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
	pool := newStatePool(logger, proto)
	h := &Host{extHost: extHost, pool: pool, logContext: logger.With()}
	if ls, err := pool.getState(); err == nil {
		h.wireFunctions(startLogger, ls)

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
	ib, err := getInbucket(ls)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get inbucket global")
	}

	events := h.extHost.Events
	const listenerName string = "lua"

	if ib.After.MessageDeleted != nil {
		events.AfterMessageDeleted.AddListener(listenerName, h.handleAfterMessageDeleted)
	}
	if ib.After.MessageStored != nil {
		events.AfterMessageStored.AddListener(listenerName, h.handleAfterMessageStored)
	}
	if ib.Before.MailFromAccepted != nil {
		events.BeforeMailFromAccepted.AddListener(listenerName, h.handleBeforeMailFromAccepted)
	}
	if ib.Before.MessageStored != nil {
		events.BeforeMessageStored.AddListener(listenerName, h.handleBeforeMessageStored)
	}
	if ib.Before.RcptToAccepted != nil {
		events.BeforeRcptToAccepted.AddListener(listenerName, h.handleBeforeRcptToAccepted)
	}
}

func (h *Host) handleAfterMessageDeleted(msg event.MessageMetadata) {
	logger, ls, ib, ok := h.prepareInbucketFuncCall("after.message_deleted")
	if !ok {
		return
	}
	defer h.pool.putState(ls)

	// Call lua function.
	logger.Debug().Msgf("Calling Lua function with %+v", msg)
	if err := ls.CallByParam(
		lua.P{Fn: ib.After.MessageDeleted, NRet: 0, Protect: true},
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

func (h *Host) handleBeforeMailFromAccepted(session event.SMTPSession) *event.SMTPResponse {
	logger, ls, ib, ok := h.prepareInbucketFuncCall("before.mail_from_accepted")
	if !ok {
		return nil
	}
	defer h.pool.putState(ls)

	logger.Debug().Msgf("Calling Lua function with %+v", session)
	if err := ls.CallByParam(
		lua.P{Fn: ib.Before.MailFromAccepted, NRet: 1, Protect: true},
		wrapSMTPSession(ls, &session),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
		return nil
	}

	lval := ls.Get(-1)
	ls.Pop(1)
	logger.Debug().Msgf("Lua function returned %q (%v)", lval, lval.Type().String())

	result, err := unwrapSMTPResponse(lval)
	if err != nil {
		logger.Error().Err(err).Msg("Bad response from Lua Function")
	}

	return result
}

func (h *Host) handleBeforeRcptToAccepted(session event.SMTPSession) *event.SMTPResponse {
	logger, ls, ib, ok := h.prepareInbucketFuncCall("before.rcpt_to_accepted")
	if !ok {
		return nil
	}
	defer h.pool.putState(ls)

	logger.Debug().Msgf("Calling Lua function with %+v", session)
	if err := ls.CallByParam(
		lua.P{Fn: ib.Before.RcptToAccepted, NRet: 1, Protect: true},
		wrapSMTPSession(ls, &session),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
		return nil
	}

	lval := ls.Get(-1)
	ls.Pop(1)
	logger.Debug().Msgf("Lua function returned %q (%v)", lval, lval.Type().String())

	result, err := unwrapSMTPResponse(lval)
	if err != nil {
		logger.Error().Err(err).Msg("Bad response from Lua Function")
	}

	return result
}

func (h *Host) handleBeforeMessageStored(msg event.InboundMessage) *event.InboundMessage {
	logger, ls, ib, ok := h.prepareInbucketFuncCall("before.message_stored")
	if !ok {
		return nil
	}
	defer h.pool.putState(ls)

	logger.Debug().Msgf("Calling Lua function with %+v", msg)
	if err := ls.CallByParam(
		lua.P{Fn: ib.Before.MessageStored, NRet: 1, Protect: true},
		wrapInboundMessage(ls, &msg),
	); err != nil {
		logger.Error().Err(err).Msg("Failed to call Lua function")
		return nil
	}

	lval := ls.Get(-1)
	ls.Pop(1)
	logger.Debug().Msgf("Lua function returned %q (%v)", lval, lval.Type().String())

	if lua.LVIsFalse(lval) {
		return nil
	}

	result, err := unwrapInboundMessage(lval)
	if err != nil {
		logger.Error().Err(err).Msg("Bad response from Lua Function")
	}

	return result
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
