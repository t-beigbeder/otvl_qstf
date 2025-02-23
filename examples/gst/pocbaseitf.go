package gst

import (
	_ "github.com/reugn/go-streams"
	_ "github.com/reugn/go-streams/extension"
	_ "github.com/reugn/go-streams/flow"
	"io"
)

type Host interface {
	Connect(cnId string) (Connection, error)
	GetCn(cnId string) Connection
}

type Connection interface {
	OpenStream(streamId string) (Stream, error)
	GetStream(streamId string) Stream
}

type Stream interface {
	GetReader() io.Reader
	GetWriter() io.WriteCloser
}
