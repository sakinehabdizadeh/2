//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package gpu

// This file implements 3-dimensional arrays of N-vectors distributed over multiple GPUs.
// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	"mumax/host"
	cu "cuda/driver"
)


// A MuMax Array represents a 3-dimensional array of N-vectors.
//
// Layout example for a (3,4) vsplice on 2 GPUs:
// 	GPU0: X0 X1  Y0 Y1 Z0 Z1
// 	GPU1: X2 X3  Y2 Y3 Z2 Z3
type Array struct {
	pointer []cu.DevicePtr // Pointers to array parts on each GPU.
	//devId []int
	_size     [4]int // INTERNAL {components, size0, size1, size2}
	size4D    []int  // {components, size0, size1, size2}
	size3D    []int  // {size0, size1, size2}
	_partSize [3]int
	partSize  []int
	partLen4D int // total number of floats per GPU
	partLen3D int // total number of floats per GPU for one component
	Stream
	Comp []Array
}


// Initializes the array to hold a field with the number of components and given size.
// 	Init(3, 1000) // gives an array of 1000 3-vectors
// 	Init(1, 1000) // gives an array of 1000 scalars
// 	Init(6, 1000) // gives an array of 1000 6-vectors or symmetric tensors
func (a *Array) InitArray(components int, size3D []int) {
	a.initSize(components, size3D)

	devices := getDevices()
	Ndev := len(devices)

	a.pointer = make([]cu.DevicePtr, Ndev)
	//a.devId = make([]int, Ndev)
	a.Stream = NewStream()
	for i := range devices {
		setDevice(devices[i])
		//a.devId[i] = devices[i]
		a.pointer[i] = cu.MemAlloc(SIZEOF_FLOAT * int64(a.partLen4D))
	}

	a.Zero()

	// initialize component arrays
	a.Comp = make([]Array, components)

	for c := range a.Comp {
		a.Comp[c].initSize(1, size3D)
		a.Comp[c].pointer = make([]cu.DevicePtr, Ndev)
		a.Comp[c].Stream = NewStream()
		//a.Comp[c].devId = make([]int, Ndev) // could re-use parent array's devId here...
		a.Comp[c].Comp = nil

		for j := range a.Comp[c].pointer {
			setDevice(_useDevice[j])
			start := c * a.partLen3D
			a.Comp[c].pointer[j] = cu.DevicePtr(offset(uintptr(a.pointer[j]), start*SIZEOF_FLOAT))
			//a.Comp[c].devId[j] = a.devId[j]
		}
	}
}


func (a *Array) initSize(components int, size3D []int) {
	Ndev := len(getDevices())
	Assert(components > 0)
	Assert(len(size3D) == 3)
	length3D := Prod(size3D)
	a.partLen4D = components * length3D / Ndev
	a.partLen3D = length3D / Ndev

	Assert(size3D[1]%Ndev == 0)

	a._size[0] = components
	for i := range size3D {
		a._size[i+1] = size3D[i]
	}
	a.size4D = a._size[:]
	a.size3D = a._size[1:]
	// Slice along the J-direction
	a._partSize[0] = a.size3D[0]
	a._partSize[1] = a.size3D[1] / Ndev
	a._partSize[2] = a.size3D[2]

}

// Returns an array which holds a field with the number of components and given size.
func NewArray(components int, size3D []int) *Array {
	t := new(Array)
	t.InitArray(components, size3D)
	return t
}


// Frees the underlying storage and sets the size to zero.
func (v *Array) Free() {
	v.Stream.Destroy()
	v.Stream = nil

	for i := range v.pointer {
		setDevice(_useDevice[i])
		v.pointer[i].Free()
		v.pointer[i] = 0

	}

	for i := range v._size {
		v._size[i] = 0
	}
}


// Address of part of the array on device deviceId.
func (a *Array) DevicePtr(deviceId int) cu.DevicePtr {
	return a.pointer[deviceId]
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
	d := dst.pointer
	s := src.pointer
	Assert(len(d) == len(s)) // in principle redundant
	start := 0
	// copies run concurrently on the individual devices
	for i := range s {
		length := src.partLen4D //s[i].length // in principle redundant     ---------------------- ------
		Assert(length == dst.partLen4D)
		cu.MemcpyDtoDAsync(cu.DevicePtr(d[i]), cu.DevicePtr(s[i]), SIZEOF_FLOAT*int64(length), dst.Stream[i])
		start += length
	}
	// Synchronize with all copies
	dst.Stream.Sync()

}


// Copy from host array to device array.
func (dst *Array) CopyFromHost(srca *host.Array) {
	src := srca.Comp

	Assert(dst.NComp() == len(src))
	// we have to work component-wise because of the data layout on the devices
	for i := range src {
		//Assert(len(dst.Comp[i]) == len(src[i])) // TODO(a): redundant
		//dst.Comp[i].CopyFromHost(src[i])

		h := src[i]
		s := dst.Comp[i].pointer
		//Assert(len(h) == len(s)) // in principle redundant
		start := 0
		for i := range s {
			length := dst.partLen3D
			cu.MemcpyHtoD(cu.DevicePtr(s[i]), cu.HostPtr(&h[start]), SIZEOF_FLOAT*int64(length))
			start += length
		}
	}
}


// Copy from device array to host array.
func (src *Array) CopyToHost(dsta *host.Array) {
	dst := dsta.Comp

	Assert(src.NComp() == len(dst))
	for i := range dst {
		//Assert(len(src.Comp[i]) == len(dst[i])) // TODO(a): redundant
		//src.Comp[i].CopyToHost(dst[i])

		h := dst[i]
		s := src.Comp[i].pointer
		//Assert(len(h) == len(s)) // in principle redundant
		start := 0
		for i := range s {
			length := src.partLen3D //s[i].length
			cu.MemcpyDtoH(cu.HostPtr(&h[start]), cu.DevicePtr(s[i]), SIZEOF_FLOAT*int64(length))
			start += length
		}

	}

}


// DEBUG: Make a freshly allocated copy on the host.
func (src *Array) LocalCopy() *host.Array {
	dst := host.NewArray(src.NComp(), src.Size3D())
	src.CopyToHost(dst)
	return dst
}


func (a *Array) Zero() {
	slices := a.pointer
	for i := range slices {
		setDevice(_useDevice[i])
		cu.MemsetD32Async(slices[i], 0, int64(a.partLen4D), a.Stream[i])
	}
	a.Stream.Sync()
}

// Error message.
const MSG_ARRAY_SIZE_MISMATCH = "array size mismatch"

// Pointer arithmetic.
func offset(ptr uintptr, bytes int) uintptr {
	return ptr + uintptr(bytes)
}
