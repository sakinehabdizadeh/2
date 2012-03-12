//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// This file implements the sum of Quantities.
// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	"mumax/gpu"
)

type SumUpdater struct {
	sum     *Quant
	parents []*Quant
	weight  []float64
}

func NewSumUpdater(sum *Quant) Updater {
	return &SumUpdater{sum, nil, nil}
}

func (u *SumUpdater) Update() {
	u.zero()
	u.addTerms()
}

func (u *SumUpdater) zero() {
	u.sum.array.Zero()
	if !u.sum.IsSpaceDependent() {
		for i := range u.sum.multiplier {
			u.sum.multiplier[i] = 0
		}
	}
}

func (u *SumUpdater) addTerms() {
	// TODO: optimize for 0,1,2 or more parents
	sum := u.sum
	parents := u.parents
	if sum.IsSpaceDependent() {
		for i := range parents {
			parent := parents[i]
			weight := u.weight[i]
			for c := 0; c < sum.NComp(); c++ {
				parComp := parent.array.Component(c)
				parMul := parent.multiplier[c]
				sumMul := sum.multiplier[c]
				sumComp := sum.array.Component(c)
				if !sumComp.IsNil() {
					weight := float32((weight * parMul) / sumMul)
					if weight != 0 {
						gpu.Madd(sumComp, sumComp, parComp, weight) // divide by sum's multiplier!
					}
				} else { // target is value
					sum.multiplier[c] += parent.multiplier[c]
				}
			}
		}
	} else {
		for p, parent := range parents {
			for c := range sum.multiplier {
				sum.multiplier[c] += parent.multiplier[c] * u.weight[p]
			}
		}
	}
}

// Adds a parent to the sum, i.e., its value*weight will be added to the sum
func (u *SumUpdater) MAddParent(name string, weight float64) {
	e := GetEngine()

	// TODO: we should check if not yet added
	Debug("MaddParent", u.sum.Name(), name, weight)

	parent := e.Quant(name)
	sum := u.sum
	if weight == 1 && parent.unit != sum.unit {
		panic(InputErr("sum: mismatched units: " + sum.FullName() + " <-> " + parent.FullName()))
	}
	u.parents = append(u.parents, parent)
	u.weight = append(u.weight, weight)
	e.Depends(sum.Name(), name)
}

// Add parent with weight 1.
func (u *SumUpdater) AddParent(name string) {
	u.MAddParent(name, 1)
}
