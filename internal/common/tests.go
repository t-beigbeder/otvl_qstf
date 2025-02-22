package common

import (
	"fmt"
	"slices"
)

func Sample(label string, hasEmpty bool) [][]byte {
	res := make([][]byte, 0)
	res = append(res, []byte(fmt.Sprintf("Start %s!", label)))
	res = append(res, []byte(fmt.Sprintf("Continue %s!", label)))
	res = append(res, []byte(fmt.Sprintf("Again %s!", label)))
	if hasEmpty {
		res = append(res, []byte{})
	}
	res = append(res, []byte(fmt.Sprintf("Last %s!", label)))
	res = append(res, []byte(fmt.Sprintf("End %s!", label)))
	return res
}

func LargeSample(label string) [][]byte {
	res := make([][]byte, 0)
	for i := 0; i < 10000; i++ {
		res = append(res, []byte(fmt.Sprintf("Continue %s#%d!", label, i)))
	}
	return res
}

func LBsSample(label string, hasEmpty bool) []byte {
	rs := slices.Concat(
		Bs2LBs([]byte(fmt.Sprintf("Start %s!", label))),
		Bs2LBs([]byte(fmt.Sprintf("Continue %s!", label))),
	)
	if hasEmpty {
		rs = slices.Concat(rs, Bs2LBs([]byte{}))
	}
	rs = slices.Concat(
		rs,
		Bs2LBs([]byte(fmt.Sprintf("Stop %s!", label))),
	)
	return rs
}

func BgLaunch(workload func()) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		workload()
	}()
	return done
}

func BgLaunchErr(workload func() error, pe *error) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		*pe = workload()
	}()
	return done
}
