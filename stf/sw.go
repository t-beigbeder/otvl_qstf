package stf

import (
	"encoding/binary"
	"golang.org/x/sync/semaphore"
	"io"
)

type defaultStartWaiter struct {
	fc      *function
	subDone *semaphore.Weighted
}

func (sw *defaultStartWaiter) startInSub(i int) {
	defer sw.subDone.Release(1)
	is := sw.fc.inStreams[i]
	var (
		bs, bsln []byte
		bln      uint32
		hasLn    bool
		err      error
	)
	if is.isRaw {
		bs = make([]byte, 128)
	} else {
		bsln = make([]byte, 4)
	}
	for {
		read := 0
		select {
		case <-sw.fc.ctx.Done():
			is.err = is.rc.Close()
			return
		default:
		}
		if is.isRaw {
			read, err = is.rc.Read(bs)
			if read != 0 && is.bset != nil {
				if cErr := is.bset(bs); cErr != nil {
					is.err = cErr
				}
			}
		} else {
			if !hasLn {
				read, err = io.ReadFull(is.rc, bsln)
				if err == nil || err == io.EOF {
					hasLn = true
					bln = binary.BigEndian.Uint32(bsln)
					bs = make([]byte, bln)
				}
			} else {
				read, err = io.ReadFull(is.rc, bs)
				if err == nil || err == io.EOF {
					if is.bset != nil {
						if cErr := is.bset(bs); cErr != nil {
							is.err = cErr
						}
					} else if is.unmarshaller != nil && is.aset != nil {
						var v any
						if cErr := is.unmarshaller(bs, &v); cErr != nil {
							is.err = cErr
						} else if cErr = is.aset(v); cErr != nil {
							is.err = cErr
						}
					}
				}
				if err == nil || err == io.EOF {
					hasLn = false
				}
			}
		}
		is.read += read
		if err != nil {
			if err != io.EOF {
				is.err = err
			}
			return
		}
	}
}

func (sw *defaultStartWaiter) startOutSub(i int) {
	defer sw.subDone.Release(1)
	os := sw.fc.outStreams[i]
	var (
		bs        []byte
		err, cErr error
	)
	for {
		written := 0
		select {
		case <-sw.fc.ctx.Done():
			return
		default:
		}
		if os.isRaw {
			if bs, cErr = os.bget(); cErr != nil {
				os.err = err
			} else if written, cErr = os.wc.Write(bs); cErr != nil {
				os.err = cErr
			}
			if err != nil {
				return
			}
		} else {
			if os.bget != nil {
				if bs, cErr = os.bget(); cErr != nil {
					os.err = err
				}
			} else if os.aget != nil && os.marshaller != nil {
				var v any
				if v, cErr = os.aget(); cErr != nil {
					os.err = cErr
				} else {
					if bs, cErr = os.marshaller(v); cErr != nil {
						os.err = cErr
					}
				}
			}
			if cErr == nil {
				bsln := make([]byte, 4)
				binary.BigEndian.PutUint32(bsln, uint32(len(bs)))
				if written, cErr = os.wc.Write(bsln); cErr != nil {
					os.err = cErr
				} else if written, cErr = os.wc.Write(bs); cErr != nil {
					os.err = cErr
				} else {
					written += 4
				}
			}
		}
		os.written += written
		if err != nil {
			os.err = err
			return
		}
	}
}

func (sw *defaultStartWaiter) Start() error {
	subNb := int64(len(sw.fc.inStreams) + len(sw.fc.outStreams))
	sw.subDone = semaphore.NewWeighted(subNb)
	if err := sw.subDone.Acquire(sw.fc.ctx, subNb); err != nil {
		return err
	}
	for i, _ := range sw.fc.inStreams {
		go sw.startInSub(i)
	}
	for i, _ := range sw.fc.outStreams {
		go sw.startOutSub(i)
	}
	return nil
}

func (sw *defaultStartWaiter) Wait() error {
	//TODO implement me
	panic("implement me")
}

var _ StartWaiter = &defaultStartWaiter{}

func NewDefaultStartWaiter(fc *function) StartWaiter {
	return &defaultStartWaiter{fc: fc}
}
