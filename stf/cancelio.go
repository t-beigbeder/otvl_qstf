package stf

import (
	"context"
	"io"
)

type cancelableReadCloser struct {
	ctx context.Context
	rc  io.ReadCloser
}

func (crc *cancelableReadCloser) Read(p []byte) (int, error) {
	select {
	case <-crc.ctx.Done():
		return 0, crc.ctx.Err()
	default:
		return crc.rc.Read(p)
	}
}

func (crc *cancelableReadCloser) Close() error {
	return crc.rc.Close()
}

func newCancelableReadCloser(ctx context.Context, rc io.ReadCloser) io.ReadCloser {
	return &cancelableReadCloser{ctx: ctx, rc: rc}
}

type cancelableWriteCloser struct {
	ctx context.Context
	wc  io.WriteCloser
}

func (cwc *cancelableWriteCloser) Write(p []byte) (int, error) {
	select {
	case <-cwc.ctx.Done():
		return 0, cwc.ctx.Err()
	default:
		return cwc.wc.Write(p)
	}
}

func (cwc *cancelableWriteCloser) Close() error {
	return cwc.wc.Close()
}

func newCancelableWriteCloser(ctx context.Context, wc io.WriteCloser) io.WriteCloser {
	return &cancelableWriteCloser{ctx: ctx, wc: wc}
}
