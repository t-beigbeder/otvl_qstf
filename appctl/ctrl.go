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
	CmdAddIStream       = "AddIStream"
	CmdAddOStream       = "AddOStream"
	CmdRunSyncFunction  = "RunSyncFunction"
	FNameNewFunction    = "/stf/NewFunction"
	FNameFuncAddIStream = "/stf/FuncAddIStream"
	FnameFuncAddOStream = "/stf/FuncAddOStream"
	FnameFuncRun        = "/stf/FuncRun"
	FnameFuncStart      = "/stf/FuncStart"
	FnameFuncWait       = "/stf/FuncWait"
	FnameFuncTerminate  = "/stf/FuncTerminate"
	MaxReqSize          = 256
	MaxRspSize          = 256
)

type CtrlReqMsg struct {
	Command  string `json:"command"`
	StreamId string `json:"streamId,omitempty"`
	FName    string `json:"fName,omitempty"`
}

type CtrlRspMsg struct {
	Error string `json:"error,omitempty"`
}

type NewFunctionReqMsg struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type NewFunctionRespMsg struct {
	Error string       `json:"error,omitempty"`
	Desc  FunctionDesc `json:"desc,omitempty"`
}
