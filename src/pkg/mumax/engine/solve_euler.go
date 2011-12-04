//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	"mumax/gpu"
	"fmt"
)

// Euler solver
type EulerSolver struct {

}

func NewEuler() *EulerSolver {
	return &EulerSolver{}
}

func (s *EulerSolver) Step() {
	e := GetEngine()
	equation := e.equation

	// First update all inputs
	for i := range equation {
		Assert(equation[i].kind == EQN_PDE1)
		equation[i].input[0].Update()
	}

	// get dt here to avoid updates later on.
	dt := engine.dt.Scalar()

	// Then step all outputs (without intermediate updates!)
	// and invalidate them.
	for i := range equation {
		y := equation[i].output[0]
		dy := equation[i].input[0]
		dyMul := dy.multiplier
		checkUniform(dyMul)
		gpu.Madd(y.Array(), y.Array(), dy.Array(), float32(dt*dyMul[0])) // TODO: faster MAdd
		y.Invalidate()
	}

	// Advance time
	e.time.SetScalar(e.time.Scalar() + dt)
	e.step.SetScalar(e.step.Scalar() + 1) // advance time step
}

//DEBUG
func checkUniform(array []float64) {
	for _, v := range array {
		if v != array[0] {
			panic(Bug(fmt.Sprint("should be all equal:", array)))
		}
	}
}
