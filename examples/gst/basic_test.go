package gst

import (
	"context"
	"errors"
	"fmt"
	ext "github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
	"os"
	"testing"
)

func TestFileIO(t *testing.T) {
	const path = "examples/gst/testdata/mfw_sample.yaml"
	ext.NewFileSource(path).Via(flow.NewPassThrough()).To(ext.NewFileSink("/tmp/TestFileIO.txt"))
}

func TestContextBase(t *testing.T) {
	ctx, cancelCause := context.WithCancelCause(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "cancelCause was %v\n", ctx.Err())
		}
	}()
	cancelCause(errors.New("TestContextBase error is fine"))
	<-done
	fmt.Fprintf(os.Stderr, "TestContextBase is done\n")
}
