package appctl

import (
	"fmt"
	"sync"
)

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
	FuncId  string `json:"funcId"`
	InPlLen int    `json:"inPlLen"`
}

type RunSyncFunctionRespMsg struct {
	Error    string `json:"error"`
	OutPlLen int    `json:"outPlLen"`
}

type NewFunctionReqMsg struct {
	FdName     string `json:"fdName"`
	FuncId     string `json:"funcId"`
	Terminable bool   `json:"terminable"`
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
