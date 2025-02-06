package appctl

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNextId(t *testing.T) {
	id := NextId("")
	require.Equal(t, "#1", id)
	id = NextId("")
	require.Equal(t, "#2", id)
	id = NextId("in")
	require.Equal(t, "in#1", id)
	id = NextId("")
	require.Equal(t, "#3", id)
	id = NextId("in")
	require.Equal(t, "in#2", id)
}
