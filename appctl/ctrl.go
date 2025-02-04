package appctl

const (
	CmdAddIStream  = "AddIStream"
	CmdAddOStream  = "AddOStream"
	CmdRunFunction = "RunFunction"
	MaxReqSize     = 256
	MaxRspSize     = 256
)

type CtrlReqMsg struct {
	Command    string `json:"command"`
	FunctionId string `json:"functionId,omitempty"`
}

type CtrlRspMsg struct {
	Error string `json:"error,omitempty"`
}
