package gst

import (
	"github.com/reugn/go-streams"
	"github.com/reugn/go-streams/flow"
)

func Take(outlet streams.Outlet, qty int) streams.Flow {
	outTaken := flow.NewPassThrough()
	go func() {
		i := 0
		for element := range outlet.Out() {
			if i < qty {
				outTaken.In() <- element
				i++
			}
			if i >= qty {
				break
			}
		}
		close(outTaken.In())
	}()
	return outTaken
}
