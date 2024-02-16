package luahost

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

func makeEmptyPool() *statePool {
	source := strings.NewReader("-- Empty source")

	chunk, err := parse.Parse(source, "from string")
	if err != nil {
		panic(err)
	}

	proto, err := lua.Compile(chunk, "from string")
	if err != nil {
		panic(err)
	}

	return newStatePool(zerolog.Nop(), proto)
}

func TestPoolGetsDistinct(t *testing.T) {
	pool := makeEmptyPool()

	a, err := pool.getState()
	require.NoError(t, err)
	b, err := pool.getState()
	require.NoError(t, err)

	if a == b {
		t.Error("Got pool a == b, expected distinct pools")
	}
}

func TestPoolGrowsWithPuts(t *testing.T) {
	pool := makeEmptyPool()

	a, err := pool.getState()
	require.NoError(t, err)
	b, err := pool.getState()
	require.NoError(t, err)
	assert.Empty(t, pool.states, "Wanted pool to be empty")

	pool.putState(a)
	pool.putState(b)

	want := 2
	if got := len(pool.states); got != want {
		t.Errorf("len pool.states got %v, want %v", got, want)
	}
}

// Closed LStates should not be added to the pool.
func TestPoolPutDiscardsClosed(t *testing.T) {
	pool := makeEmptyPool()

	a, err := pool.getState()
	require.NoError(t, err)
	assert.Empty(t, pool.states, "Wanted pool to be empty")

	a.Close()
	pool.putState(a)
	assert.Empty(t, pool.states, "Wanted pool to remain empty")
}

func TestPoolPutClearsStack(t *testing.T) {
	pool := makeEmptyPool()

	ls, err := pool.getState()
	require.NoError(t, err)
	assert.Empty(t, pool.states, "Wanted pool to be empty")

	// Setup stack.
	ls.Push(lua.LNumber(4))
	ls.Push(lua.LString("bacon"))
	require.Equal(t, 2, ls.GetTop(), "Want stack to have two items")

	// Return and verify stack cleared.
	pool.putState(ls)
	assert.Len(t, pool.states, 1, "Wanted pool to have one item")
	require.Equal(t, 0, ls.GetTop(), "Want stack to be empty")
}

func TestPoolSetsChannels(t *testing.T) {
	pool := makeEmptyPool()
	pool.createChannel("test_chan")

	s, err := pool.getState()
	require.NoError(t, err)

	got := s.GetGlobal("test_chan")
	assert.Equal(t, lua.LTChannel, got.Type(),
		"Got global type %v, wanted LTChannel", got.Type().String())
}
