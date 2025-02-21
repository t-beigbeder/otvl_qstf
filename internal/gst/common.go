package gst

import (
	"fmt"
	"github.com/reugn/go-streams"
	"github.com/reugn/go-streams/flow"
)

func AsStringFlow() streams.Flow {
	tsf := func(v any) string {
		return fmt.Sprintf("%v", v)
	}
	return flow.NewMap(tsf, 1)
}
