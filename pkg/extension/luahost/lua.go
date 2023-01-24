package luahost

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/inbucket/inbucket/pkg/config"
	"github.com/inbucket/inbucket/pkg/extension"
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
