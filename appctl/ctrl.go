package appctl

import "github.com/t-beigbeder/otvl_qstf/stf"

const (
	CmdAddIStream       = "AddIStream"
	CmdAddOStream       = "AddOStream"
	CmdRunSyncFunction  = "RunSyncFunction"
	CmdRunRemoteCommand = "RunRemoteCommand"
	MaxReqSize          = 256
	MaxRspSize          = 256
)

type CtrlReqMsg struct {
	Command       string          `json:"command"`
	FunctionId    string          `json:"functionId,omitempty"`
	RemoteCommand stf.CommandSpec `json:"remoteCommand,omitempty"`
}

type CtrlRspMsg struct {
	Error string `json:"error,omitempty"`
}
