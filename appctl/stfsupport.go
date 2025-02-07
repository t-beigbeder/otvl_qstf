package appctl

import (
	"fmt"
)

func DeclareStfsNewFunction(cat *FunctionCatalog) error {
	return cat.DeclareFunction(
		FunctionDesc{Name: FNameNewFunction}, nil,
		func(in any) any {
			res := NewFunctionRespMsg{}
			in, ok := in.(NewFunctionReqMsg)
			if !ok {
				res.Error = fmt.Sprintf("NewFunction: invalid input type %T", in)
				return res
			}
			
			res.Error = "not yet implemented"
			return res
		})
}
