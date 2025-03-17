package common

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

type I16Bs [2]byte

func NewI16Bs(ln uint16) I16Bs {
	lbs := I16Bs{}
	binary.BigEndian.PutUint16(lbs[:], ln)
	return lbs
}

func SetI16Bs(ln uint16, bs []byte) {
	binary.BigEndian.PutUint16(bs, ln)
}

func (lbs I16Bs) Get() uint16 {
	return binary.BigEndian.Uint16(lbs[:])
}

func (lbs I16Bs) String() string {
	return fmt.Sprintf("%d", lbs.Get())
}

func Bs2LBs(ebs []byte) []byte {
	if len(ebs) > math.MaxUint16 {
		panic(fmt.Sprintf("buffer too big for uint16 %d", len(ebs)))
	}
	bs := make([]byte, uint16(len(ebs))+2)
	ibs := NewI16Bs(uint16(len(ebs)))
	copy(bs, ibs[:])
	copy(bs[2:], ebs)
	return bs
}

func LBsReader(rr io.Reader) ([]byte, error) {
	lbs := I16Bs{}
	_, err := io.ReadFull(rr, lbs[:])
	if err != nil {
		return nil, err
	}
	ebs := make([]byte, lbs.Get())
	_, err = io.ReadFull(rr, ebs)
	if err != nil {
		return nil, err
	}
	return ebs, nil
}

func LStReader(rr io.Reader) (string, error) {
	ebs, err := LBsReader(rr)
	if err != nil {
		return "", err
	}
	return string(ebs), nil
}

type I32Bs [4]byte

func NewI32Bs(ln uint32) I32Bs {
	lbs := I32Bs{}
	binary.BigEndian.PutUint32(lbs[:], ln)
	return lbs
}

func SetI32Bs(ln uint32, bs []byte) {
	binary.BigEndian.PutUint32(bs, ln)
}

func (lbs I32Bs) Get() uint32 {
	return binary.BigEndian.Uint32(lbs[:])
}

func (lbs I32Bs) String() string {
	return fmt.Sprintf("%d", lbs.Get())
}

type I64Bs [8]byte

func NewI64Bs(ln uint64) I64Bs {
	lbs := I64Bs{}
	binary.BigEndian.PutUint64(lbs[:], ln)
	return lbs
}

func SetI64Bs(ln uint64, bs []byte) {
	binary.BigEndian.PutUint64(bs, ln)
}

func (lbs I64Bs) Get() uint64 {
	return binary.BigEndian.Uint64(lbs[:])
}

func (lbs I64Bs) String() string {
	return fmt.Sprintf("%d", lbs.Get())
}
