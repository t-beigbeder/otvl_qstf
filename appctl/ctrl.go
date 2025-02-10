package appctl

import (
	"encoding/binary"
	"fmt"
	"sync"
)

type RidBs [8]byte

func NewRidBs(rid uint64) RidBs {
	rbs := RidBs{}
	binary.BigEndian.PutUint64(rbs[:], rid)
	return rbs
}

func SetRidBs(rid uint64, bs []byte) {
	binary.BigEndian.PutUint64(bs, rid)
}

func (rbs RidBs) Get() uint64 {
	return binary.BigEndian.Uint64(rbs[:])
}

func (rbs RidBs) String() string {
	return fmt.Sprintf("%08x", rbs.Get())
}

type LenBs [4]byte

func NewLenBs(rid uint32) LenBs {
	lbs := LenBs{}
	binary.BigEndian.PutUint32(lbs[:], rid)
	return lbs
}

func SetLenBs(ln uint32, bs []byte) {
	binary.BigEndian.PutUint32(bs, ln)
}

func (lbs LenBs) Get() uint32 {
	return binary.BigEndian.Uint32(lbs[:])
}

func (lbs LenBs) String() string {
	return fmt.Sprintf("%d", lbs.Get())
}

type RspData struct {
	Rid     uint64
	Err     error
	Rsp     any
	Payload any
}

var (
	idmux  sync.Mutex
	counts map[string]int
)

func init() {
	counts = map[string]int{}
}

func NextId(prefix string) string {
	idmux.Lock()
	defer idmux.Unlock()
	_, ok := counts[prefix]
	if !ok {
		counts[prefix] = 0
	}
	counts[prefix]++
	return fmt.Sprintf("%s#%d", prefix, counts[prefix])
}

const (
	CmdAddIStream      = "AddIStream"
	CmdAddOStream      = "AddOStream"
	CmdRunSyncFunction = "RunSyncFunction"
	CmdNewFunction     = "NewFunction"
	CmdFuncAddIStream  = "FuncAddIStream"
	CmdFuncAddOStream  = "FuncAddOStream"
	CmdFuncOper        = "FuncOper"
	MaxReqSize         = 256
	MaxRspSize         = 256
)

type AddStreamReqMsg struct {
	StreamId string `json:"streamId"`
}

type RunSyncFunctionReqMsg struct {
	FdName string `json:"fdName"`
	FuncId string `json:"funcId"`
}

type NewFunctionReqMsg struct {
	FdName string `json:"fdName"`
	FuncId string `json:"funcId"`
}

type NewFunctionRespMsg struct {
	Error string       `json:"error"`
	Desc  FunctionDesc `json:"desc"`
}

type FuncAddStreamReqMsg struct {
	FuncId   string `json:"funcId"`
	StreamId string `json:"streamId"`
	Discrete bool   `json:"discrete"`
	MaxLen   int    `json:"maxLen"`
	MaxNb    int    `json:"maxNb"`
	BSize    int    `json:"bSize"`
}

type FuncOperReqMsg struct {
	Oper   string `json:"oper"`
	FuncId string `json:"funcId"`
}

type RespMsg struct {
	Error string `json:"error"`
}

type ReqDesc struct {
	Req func() any
	Rsp func() any
}

func GetReqDesc(cmd string) *ReqDesc {
	reqDescs := map[string]ReqDesc{
		CmdAddIStream:      {func() any { return &AddStreamReqMsg{} }, func() any { return &RespMsg{} }},
		CmdAddOStream:      {func() any { return &AddStreamReqMsg{} }, func() any { return &RespMsg{} }},
		CmdRunSyncFunction: {func() any { return &RunSyncFunctionReqMsg{} }, func() any { return &RespMsg{} }},
		CmdNewFunction:     {func() any { return &NewFunctionReqMsg{} }, func() any { return &NewFunctionRespMsg{} }},
		CmdFuncAddIStream:  {func() any { return &FuncAddStreamReqMsg{} }, func() any { return &RespMsg{} }},
		CmdFuncAddOStream:  {func() any { return &FuncAddStreamReqMsg{} }, func() any { return &RespMsg{} }},
		CmdFuncOper:        {func() any { return &FuncOperReqMsg{} }, func() any { return &RespMsg{} }},
	}
	rd, ok := reqDescs[cmd]
	if !ok {
		return nil
	}
	return &rd
}
