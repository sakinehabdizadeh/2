//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package gpu

// This file implements 3-dimensional arrays of N-vectors on the GPU
// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	"mumax/host"
	cu "cuda/driver"
)


// A MuMax Array represents a 3-dimensional array of N-vectors.
// TODO: get components as array (slice in J direction), get device part as array.
type Array struct {
	splice vSplice // Underlying multi-GPU storage
	_size  [4]int  // INTERNAL {components, size0, size1, size2}
	size4D []int   // {components, size0, size1, size2}
	size3D []int   // {size0, size1, size2}
	//length int
}


// Initializes the array to hold a field with the number of components and given size.
func (t *Array) Init(components int, size3D []int) {
	Assert(len(size3D) == 3)
	Assert(t.splice.IsNil()) // should not be initialized already
	//t.length = Prod(size3D)
	length := Prod(size3D)
	t.splice.Init(components, length)
	t._size[0] = components
	for i := range size3D {
		t._size[i+1] = size3D[i]
	}
	t.size4D = t._size[:]
	t.size3D = t._size[1:]
	t.Zero()
}


// Returns an array which holds a field with the number of components and given size.
func NewArray(components int, size3D []int) *Array {
	t := new(Array)
	t.Init(components, size3D)
	return t
}


// Frees the underlying storage and sets the size to zero.
func (t *Array) Free() {
	t.splice.Free()
	for i := range t._size {
		t._size[i] = 0
	}
	//t.length = 0
}


// Address of part of the array on device deviceId.
func (a *Array) DevicePtr(deviceId int) cu.DevicePtr {
	return a.splice.list.slice[deviceId].array
}

// True if unallocated/freed.
func (a *Array) IsNil() bool {
	return a.splice.IsNil()
}

// Total number of elements
func (a *Array) Len() int {
	return a._size[0] * a._size[1] * a._size[2] * a._size[3]
}

// Number of components (1: scalar, 3: vector, ...).
func (a *Array) NComp() int {
	return a._size[0]
}

// Size of the vector field
func (a *Array) Size3D() []int {
	return a.size3D
}
func (dst *Array) CopyFromDevice(src *Array) {
	// test for equal size
	for i, d := range dst._size {
		if d != src._size[i] {
			panic(MSG_ARRAY_SIZE_MISMATCH)
		}
	}
	(&(dst.splice)).CopyFromDevice(&(src.splice))
}


// Copy from host array to device array.
func (dst *Array) CopyFromHost(src *host.Array) {
	(&(dst.splice)).CopyFromHost(src.Comp)
}


// Copy from device array to host array.
func (src *Array) CopyToHost(dst *host.Array) {
	(&(src.splice)).CopyToHost(dst.Comp)
}


// DEBUG: Make a freshly allocated copy on the host.
func (src *Array) LocalCopy() *host.Array {
	dst := host.NewArray(src.NComp(), src.Size3D())
	src.CopyToHost(dst)
	return dst
}


func (a *Array) Zero() {
	slices := a.splice.list.slice
	for i := range slices {
		assureContextId(slices[i].devId)
		cu.MemsetD32Async(slices[i].array, 0, int64(slices[i].length), slices[i].stream)
	}
	for i := range slices {
		slices[i].stream.Synchronize()
	}
}

// Error message.
const MSG_ARRAY_SIZE_MISMATCH = "array size mismatch"
