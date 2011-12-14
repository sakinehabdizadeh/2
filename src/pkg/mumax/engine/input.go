//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// Auhtor: Arne Vansteenkiste

import (
	. "mumax/common"
	"mumax/host"
	"path"
)

// Reads an array from a file.
func ReadFile(fname string) *host.Array {
	switch path.Ext(fname) {
	default:
		panic(InputErrF("Can not load file with extension ", path.Ext(fname)))
	}
	return nil // silence 6g	

}