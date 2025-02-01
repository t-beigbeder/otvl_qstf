package stf

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewFuncWrapperBasic(t *testing.T) {
	wbs, err := toJsonBytes("value for test")
	assert.NoError(t, err)
	in := bytes.NewReader(wbs)
	out := newBufWr()
	fcw, err := NewFuncWrapper(context.Background(),
		func(a any) any {
			return "response for " + a.(string)
		},
		in, out)
	assert.NoError(t, err)
	err = fcw.Run()
	assert.NoError(t, err)
	a, err := fromJsonBytes(out)
	assert.Equal(t, "response for value for test", a)
}

func TestNewFuncWrapperSlow(t *testing.T) {
	wbs, err := toJsonBytes("value for test")
	assert.NoError(t, err)
	in := bytes.NewReader(wbs)
	out := newBufWr()
	fcw, err := NewFuncWrapper(context.Background(),
		func(a any) any {
			time.Sleep(time.Millisecond * 200)
			return "response for " + a.(string)
		},
		in, out)
	assert.NoError(t, err)
	err = fcw.Run()
	assert.NoError(t, err)
	a, err := fromJsonBytes(out)
	assert.Equal(t, "response for value for test", a)
}
