package appctl

import (
	"encoding/json"
	"testing"
)

func TestNewFunctionCatalog(t *testing.T) {
	catalog := NewFunctionCatalog()
	_ = catalog
}

func TestWrapFunction1(t *testing.T) {
	wrapped := func(req any) any {
		ereq := req.(*NewFunctionReqMsg)
		return NewFunctionRespMsg{Error: "no", Desc: FunctionDesc{Name: ereq.Name}}
	}
	wrapper := func(wrapped func(any) any, jin []byte, it any) any {
		_ = json.Unmarshal(jin, it)
		out := wrapped(it)
		return out
	}
	jw, _ := json.Marshal(NewFunctionReqMsg{Name: "foo"})
	z := wrapper(wrapped, jw, &NewFunctionReqMsg{})
	_ = z
}
