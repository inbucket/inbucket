package luahost_test

import (
	"strings"
	"testing"

	"github.com/inbucket/inbucket/pkg/extension"
	"github.com/inbucket/inbucket/pkg/extension/luahost"
	"github.com/stretchr/testify/require"
)

func TestEmptyScript(t *testing.T) {
	script := ""
	extHost := extension.NewHost()

	_, err := luahost.NewFromReader(extHost, strings.NewReader(script), "test.lua")
	require.NoError(t, err)
}
