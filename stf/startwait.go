package stf

import (
	"encoding/binary"
	"errors"
	"golang.org/x/sync/semaphore"
	"io"
)

type defaultStartWaiter struct {
	fc      *function
	subDone *semaphore.Weighted
}

func readRaw(is istream) error {
	bs := make([]byte, is.bSize)
	read, err := is.rc.Read(bs)
	if read != 0 && is.bset != nil {
		if cErr := is.bset(bs); cErr != nil {
			err = cErr
		}
	}
	is.read += read
	return err
}

func readSetter(is istream) error {
	bsln := make([]byte, 4)
	read, err := io.ReadFull(is.rc, bsln)
	is.read += read
	if err == nil {
		bln := binary.BigEndian.Uint32(bsln)
		bs := make([]byte, bln)
		read, err = io.ReadFull(is.rc, bs)
		is.read += read
		if err == nil {
			if is.bset != nil {
				err = is.bset(bs)
			}
			if is.unmarshaller != nil && is.aset != nil {
				var v any
				if err = is.unmarshaller(bs, &v); err == nil {
					err = is.aset(v)
				}
			}
		}
	}
	return err
}

func (sw *defaultStartWaiter) startInSub(i int) {
	defer sw.subDone.Release(1)
	is := sw.fc.inStreams[i]
	for {
		var err error
		if is.isRaw {
			err = readRaw(is)
		} else {
			err = readSetter(is)
		}
		if err != nil {
			if err != io.EOF {
				is.err = err
			}
			return
		}
		if !is.isRaw && is.setOnce {
			return
		}
	}
}

func writeRaw(os ostream) error {
	bs, err := os.bget()
	if err != nil {
		return err
	}
	written := 0
	written, err = os.wc.Write(bs)
	os.written += written
	return err
}

func writeGetter(os ostream) error {
	var (
		bs  []byte
		v   any
		err error
	)
	if os.bget != nil {
		bs, err = os.bget()
	} else if os.aget != nil && os.marshaller != nil {
		if v, err = os.aget(); err == nil {
			bs, err = os.marshaller(v)
		}
	} else {
		err = errors.New("output stream has no getter")
	}
	if err != nil {
		return err
	}
	bsln := make([]byte, 4)
	binary.BigEndian.PutUint32(bsln, uint32(len(bs)))
	if _, err = os.wc.Write(bsln); err != nil {
		return err
	}
	written := 0
	written, err = os.wc.Write(bs)
	os.written += written + 4
	return err
}

func (sw *defaultStartWaiter) startOutSub(i int) {
	defer sw.subDone.Release(1)
	os := sw.fc.outStreams[i]
	for {
		var err error
		if os.isRaw {
			err = writeRaw(os)
		} else {
			err = writeGetter(os)
		}
		if err != nil {
			os.err = err
			return
		}
		if !os.isRaw && os.getOnce {
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
	subNb := int64(len(sw.fc.inStreams) + len(sw.fc.outStreams))
	if err := sw.subDone.Acquire(sw.fc.ctx, subNb); err != nil {
		return err
	}
	return nil
}

var _ StartWaiter = &defaultStartWaiter{}

func NewDefaultStartWaiter(fc *function) StartWaiter {
	return &defaultStartWaiter{fc: fc}
}
