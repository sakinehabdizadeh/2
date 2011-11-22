//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// This file implements the average of a quantity.
// Author: Arne Vansteenkiste

import (
	"mumax/gpu"
)

type AverageUpdater struct {
	in, out *Quant
	reduce  gpu.Reductor
}

func NewAverageUpdater(in, out *Quant) Updater {
	checkKinds(in, FIELD, MASK)
	avg := new(AverageUpdater)
	avg.in = in
	avg.out = out
	avg.reduce.Init(1, GetEngine().GridSize())
	return avg
}

func (this *AverageUpdater) Update() {
	var sum float32 = 666

	if this.in.nComp == 1 {
		sum = this.reduce.Sum(this.in.Array())
		this.out.SetScalar(float64(sum) * this.in.multiplier[0] / float64(GetEngine().NCell()))
	} else {
		for c := 0; c < this.in.nComp; c++ {
			sum := this.reduce.Sum(&(this.in.Array().Comp[c]))
			this.out.SetComponent(c, float64(sum)*this.in.multiplier[c]/float64(GetEngine().NCell()))
		}
	}

}
