//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package gpu

// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	"testing"
	"fmt"
)

func TestDuplicateFunc(test *testing.T) {
	// test fails if there is no panic
	defer func() {
		err := recover()
		if err == nil {
			test.Fail()
		} else {
			Log(err)
		}
	}()

	LoadModule(GetExecDir() + "testmodule.ptx")
	LoadModule(GetExecDir() + "testmodule2.ptx") // bad, contains duplicate function
}


func TestModule(test *testing.T) {
	Global("testmodule", "testMemset")
	//
	//	size := []int{4, 8, 128}
	//	dev := NewArray(1, size)
	//
	//	for i := range c.DeviceClosure {
	//		c.DeviceClosure[i].SetArg(0, float32(42))
	//		c.DeviceClosure[i].SetArg(1, dev.DevicePtr(i))
	//		c.DeviceClosure[i].SetArg(2, dev.splice.list.slice[i].Len())
	//		c.DeviceClosure[i].BlockDim[0] = 128
	//		c.DeviceClosure[i].GridDim[0] = DivUp(Prod(size)/DeviceCount(), 128)
	//	}
	//
	//	c.Call()
	//
	//	host := dev.LocalCopy()
	//	for _, h := range host.List {
	//		if h != 42 {
	//			fmt.Println(host.Array)
	//			test.Fail()
	//			break
	//		}
	//	}
}


func TestPTXParse(test *testing.T) {
	fmt.Println(parsePTXArgTypes(GetExecDir() + "testmodule.ptx"))
}