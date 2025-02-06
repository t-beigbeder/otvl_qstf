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
	CmdFuncRun         = "FuncRun"
	CmdFuncStart       = "FuncStart"
	CmdFuncWait        = "FuncWait"
	CmdFuncTerminate   = "FuncTerminate"
	MaxReqSize         = 256
	MaxRspSize         = 256
)

type CtrlReqMsg struct {
	Command    string `json:"command"`
	StreamId   string `json:"streamId,omitempty"`
	FunctionId string `json:"functionId,omitempty"`
}

type CtrlRspMsg struct {
	Error string `json:"error,omitempty"`
}
