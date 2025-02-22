package common

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBgLaunchErr(t *testing.T) {
	e := errors.New("")
	done := BgLaunchErr(
		func() error {
			return fmt.Errorf(t.Name())
		}, &e)
	<-done
	require.Equal(t, e.Error(), t.Name())
}
