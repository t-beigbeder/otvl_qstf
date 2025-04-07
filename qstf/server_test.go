package qstf

import "testing"

func TestX(t *testing.T) {

}

func TestSimpleUseCase(t *testing.T) {
	// StrSrc(args) -> Flow(client) -> QstSnk(func, args) -> (net)
	// (net) -> QstSrc(func,args) -> Flow(func) -> StdoutSnk

}

func TestBasicSideChannel(t *testing.T) {
	// Same as TestSimpleUseCase but args are passed on a side channel
}
