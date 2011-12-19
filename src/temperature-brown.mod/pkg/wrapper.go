//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package temperature_brown

// CGO wrappers for temperature.cu
// Author: Arne Vansteenkiste

//#include "libmumax2.h"
import "C"
import (
	"mumax/gpu"
	"unsafe"
)

func ScaleNoise(noise, alphaMask gpu.Array,
tempMask *gpu.Array, alphaKB2tempMul float32,
mSatMask *gpu.Array, mu0VgammaDtMsatMul float32,
stream gpu.Stream, Npart int) {

	C.temperature_scaleNoise(
		(**C.float)(unsafe.Pointer(&(noise.pointer[0]))),
		(**C.float)(unsafe.Pointer(&(alphaMask.pointer[0]))),
		(**C.float)(unsafe.Pointer(&(tempMask.pointer[0]))),
		(C.float)(alphaKB2tempMul),
		(**C.float)(unsafe.Pointer(&(mSatMask.pointer[0]))),
		(C.float)(mu0VgammaDtMsatMul),
		(*C.CUstream)(unsafe.Pointer(&(stream[0]))))
}
