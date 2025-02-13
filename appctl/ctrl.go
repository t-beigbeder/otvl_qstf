package appctl

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/t-beigbeder/otvl_qstf/internal/bfio"
	"gopkg.in/yaml.v3"
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

func NewLenBs(ln uint32) LenBs {
	lbs := LenBs{}
	binary.BigEndian.PutUint32(lbs[:], ln)
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
	CmdCloseStream     = "CloseStream"
	CmdGetFDesc        = "GetFDesc"
	CmdRunSyncFunction = "RunSyncFunction"
	CmdNewFunction     = "NewFunction"
	CmdFuncAddIStream  = "FuncAddIStream"
	CmdFuncAddOStream  = "FuncAddOStream"
	CmdFuncOper        = "FuncOper"
	MaxReqSize         = 256
	MaxRspSize         = 1024
	MaxInPlSize        = bfio.MaxBufWriterSize / 2
	MaxOutPlSize       = bfio.MaxBufWriterSize / 2
)

type StfQErr quic.ApplicationErrorCode

const (
	QServerCloseNoError quic.ApplicationErrorCode = iota
	QServerInitError
	QServerProtoError
	QClientCloseNoError
	QClientInitError
	QClientProtoError
)

const (
	QServerStreamNoError quic.StreamErrorCode = iota
	QServerStreamProtoError
)

type AddStreamReqMsg struct {
	StreamId string `json:"streamId"`
}

type CloseStreamReqMsg struct {
	StreamId string `json:"streamId"`
	IsIn     bool   `json:"isIn"`
}

type GetFDescReqMsg struct {
	FdName string `json:"fdName"`
}

type GetFDescRespMsg struct {
	Error string       `json:"error"`
	Desc  FunctionDesc `json:"desc"`
}

type RunSyncFunctionReqMsg struct {
	FdName string `json:"fdName"`
	FuncId string `json:"funcId"`
}

type RunSyncFunctionRespMsg GetFDescRespMsg

type NewFunctionReqMsg struct {
	FdName string `json:"fdName"`
	FuncId string `json:"funcId"`
}

type NewFunctionRespMsg GetFDescRespMsg

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
		CmdCloseStream:     {func() any { return &CloseStreamReqMsg{} }, func() any { return &RespMsg{} }},
		CmdGetFDesc:        {func() any { return &GetFDescReqMsg{} }, func() any { return &GetFDescRespMsg{} }},
		CmdRunSyncFunction: {func() any { return &RunSyncFunctionReqMsg{} }, func() any { return &RunSyncFunctionRespMsg{} }},
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

func MarshallToWrapped(mt Marshaller, rqPl any) ([]byte, error) {
	var ms func(any) ([]byte, error)
	switch mt {
	case MarshalNone:
		bs, ok := rqPl.([]byte)
		if !ok {
			return nil, fmt.Errorf("MarshallToWrapped: expected []byte, got %T", rqPl)
		}
		return bs, nil
	case MarshalJson:
		ms = json.Marshal
	case MarshalYaml:
		ms = yaml.Marshal
	default:
		return nil, fmt.Errorf("MarshallToWrapped: unsupported marshaller: %v", mt)
	}
	return ms(rqPl)
}

func UnmarshallFromWrapped(mt Marshaller, rsPl any, v any) (any, error) {
	var ums func(data []byte, v any) error
	switch mt {
	case MarshalNone:
		return v, nil
	case MarshalJson:
		ums = json.Unmarshal
	case MarshalYaml:
		ums = yaml.Unmarshal
	default:
		return nil, fmt.Errorf("UnmarshallFromWrapped: unsupported marshaller: %v", mt)
	}
	plbs, ok := rsPl.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected output payload type: %T", rsPl)
	}
	err := ums(plbs, v)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
