//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// Author: Mykola Dvornik, Arne Vansteenkiste

import (
	"fmt"
	"math"
	. "mumax/common"
	"mumax/gpu"
	"sort"
	"container/list"
)

// Naive Backward Euler solver
type BDFEulerAuto struct {
	ybuffer []*gpu.Array // current value of the quantity

	y0buffer []*gpu.Array // the value of quantity at the begining of the step
	y1buffer []*gpu.Array // the value of quantity after pedictor step

	dy0buffer  []*gpu.Array // the value of quantity derivative at the begining of the step
	dybuffer   []*gpu.Array // the buffer for quantity derivative 
	err        []*Quant     // error estimates for each equation
	maxAbsErr     []*Quant     // maximum absolute error per step for each equation
	maxRelErr     []*Quant     // maximum absolute error per step for each equation
	maxIterErr []*Quant     // error iterator error estimates for each equation
	maxIter    []*Quant     // maximum number of iterations per step
	newDt      []float64    // 
	diff       []gpu.Reductor
	err_list   []*list.List
	iterations *Quant
	badSteps   *Quant
	minDt      *Quant
	maxDt      *Quant
}

func (s *BDFEulerAuto) Step() {
	e := GetEngine()
	t0 := e.time.Scalar()
	
	s.badSteps.SetScalar(0)
	s.iterations.SetScalar(0)

	equation := e.equation
	// make sure that errors history is wiped for t0 = 0s!
	if t0 == 0.0 {
		for i := range equation {
			s.err_list[i].Init()
		}
	}
	// save everything in the begining
	for i := range equation {
		equation[i].input[0].Update()
		y := equation[i].output[0]
		dy := equation[i].input[0]
		s.y0buffer[i].CopyFromDevice(y.Array())   // save for later
		s.dy0buffer[i].CopyFromDevice(dy.Array()) // save for later
	}
	
	const maxTry = 3 // undo at most this many bad steps
	const headRoom = 0.8
	
	for i := range equation {
		
		// try to integrate maxTry times at most
		for try := 0; try < maxTry; try++ {
			// get dt here to avoid updates later on.
			dt := engine.dt.Scalar()
			iter := 0
			badStep := false
			badIterator := false
			
			y := equation[i].output[0]
			dy := equation[i].input[0]
			dyMul := dy.multiplier
			t_step := dt * dyMul[0]
			h := float32(t_step)
			
			// Do zero order approximation with forward Euler method
			// The zero-order approximation is used as a starting point for fixed-point iteration
			gpu.Madd(y.Array(), s.y0buffer[i], s.dy0buffer[i], h)
			y.Invalidate()
			iter = iter + 1
			s.iterations.SetScalar(s.iterations.Scalar() + 1)
			
			// Advance time and update all inputs 
			// Since implicit methods use derivative at right side
			e.time.SetScalar(t0 + dt)
			equation[i].input[0].Update()
			
			s.dybuffer[i].CopyFromDevice(dy.Array())
			
			// Do higher order approximation until converges    
			maxIterErr := s.maxIterErr[i].Scalar()
			maxIter := int(s.maxIter[i].Scalar())
			
			iter = 0
			err := 1e10
			m := 1e-38
			// Do predictor: BDF Euler
			// Store previous errors in the list to measure the convergence
			
			err_list := list.New()
			err_list.PushFront(err)
			err_list.PushFront(err)
			
			for err > maxIterErr {
				gpu.Madd(s.ybuffer[i], s.y0buffer[i], dy.Array(), h) 
				err = float64(s.diff[i].MaxDiff(y.Array(), s.ybuffer[i]))
				//Estimate convergence
				elem := err_list.Front()
				e := elem.Value.(float64)
				m = err / e
				//~ Debug("AM0 m:",m)
				if m > 0.99 {
					// If it is converged then badstep is not reported
					// If there is indeed a badstep then it will be reflected in global error
					//if err > maxIterErr {Debug("BDF AM0, TOL failed at:", err)}
					break
				}
				err_list.PushFront(err)
				err_list.Remove(err_list.Back())				
				iter = iter + 1
				s.iterations.SetScalar(s.iterations.Scalar() + 1)
				y.Array().CopyFromDevice(s.ybuffer[i])
				y.Invalidate()
				equation[i].input[0].Update()
				if iter > maxIter {
					badIterator = true
					break
				}
			}
			// If fixed-point iterator cannot converge, then panic
			if badIterator {
				panic(Bug(fmt.Sprintf("The BDF iterator cannot converge for %s! Please increase the maximum number of iterations and re-run!",y.Name())))
			}
			
			// Update and save the result for step predictor
			// equation[i].input[0].Update()
			
			s.y1buffer[i].CopyFromDevice(y.Array())
			
			// Do corrector: BDF trapezoidal
			iter = 0
			err = 1e10
			m = 1e-38
			err_list.Init()
			err_list.PushFront(err)
			err_list.PushFront(err)
			
			dy.Array().CopyFromDevice(s.dybuffer[i])
			for err > maxIterErr {
				gpu.Add(s.ybuffer[i], dy.Array(), s.dy0buffer[i])
				gpu.Madd(s.ybuffer[i], s.y0buffer[i], s.ybuffer[i], 0.5*h) 
				err = float64(s.diff[i].MaxDiff(y.Array(), s.ybuffer[i]))
				//Estimate convergence
				elem := err_list.Front()
				e := elem.Value.(float64)
				m = err / e
				//~ Debug("AM1 m:",m)
				if m > 0.99 {
					// If it is converged then badstep is not reported
					// If there is indeed a badstep then it will be reflected in global error
					//~ if err > maxIterErr {Debug("BDF AM1, TOL failed at:", err)}
					break
				}
				err_list.PushFront(err)
				err_list.Remove(err_list.Back())				
				iter = iter + 1
				s.iterations.SetScalar(s.iterations.Scalar() + 1)
				y.Array().CopyFromDevice(s.ybuffer[i])
				y.Invalidate()
				equation[i].input[0].Update()
				if iter > maxIter {
					badIterator = true
					break
				}
			}
			 
			if badIterator {
				// If fixed-point iterator cannot converge, then panic
				panic(Bug(fmt.Sprintf("The BDF iterator cannot converge for %s! Please increase the maximum number of iterations and re-run!",y.Name())))
			}
			
			abs_dy := 1.5 * float64(s.diff[i].MaxDiff(y.Array(), s.y1buffer[i])) // / 3.0
			max_y := float64(s.diff[i].MaxAbs(s.y1buffer[i]))
		
			s.err[i].SetScalar(abs_dy)
			
			maxAbsStepErr := s.maxAbsErr[i].Scalar()
			
			StepErr := abs_dy
			if abs_dy == 0.0 {
				// Highly unlikely, but possible situation
				StepErr = headRoom * maxAbsStepErr
			}
			
			maxRelStepErr := s.maxRelErr[i].Scalar() * max_y
			maxStepErr := maxAbsStepErr
			
			// if step is large then threshold then badstep is reported
			if StepErr > maxAbsStepErr || StepErr > maxRelStepErr {
				s.badSteps.SetScalar(s.badSteps.Scalar() + 1)
				badStep = true
				// We don't now which particular condition has triggered the badstep, so let us pick the smallest error threshold
				maxStepErr = math.Min(maxAbsStepErr, maxRelStepErr)
			}
			
			// Let us compare the current error to the error from previous steps
			errRatio := 0.0	
			if  s.err_list[i].Len() == 3  && !badStep {
			    //do softcore adjustment
			    elem := s.err_list[i].Front()
			    e   := abs_dy
			    ep1 := elem.Value.(float64)
			    ep2 := elem.Next().Value.(float64)
			    pre_f1 := math.Pow((ep1/e),0.075)
			    pre_f2 := math.Pow((ep1*ep1/(e*ep2)), 0.01)
			    pre := math.Pow((maxStepErr / e), 0.175)
			    errRatio = 0.96 * pre_f1 * pre * pre_f2
			} else if !badStep {
			    //do hardcore adjustment if there is not enough history and err < err0
			    errRatio = headRoom * math.Sqrt(maxStepErr / StepErr)
			} else {
			    //do hardcore adjustment if there is not enough history and (or) err > err0
			    errRatio = headRoom * math.Pow((maxStepErr / StepErr), 0.33333333333)
			}
			
			step_corr := math.Abs(errRatio)
			
			if step_corr > 1.5 {
				step_corr = 1.5
			}
			if step_corr < 0.1 {
				step_corr = 0.1
			}
			new_dt := dt * step_corr
			if new_dt < s.minDt.Scalar() {
				new_dt = s.minDt.Scalar()
			}
			if new_dt > s.maxDt.Scalar() {
				new_dt = s.maxDt.Scalar()
			}
			if !badStep || try == (maxTry - 1) {
				// Memorize only successful steps!
				s.err_list[i].PushFront(StepErr)
				if s.err_list[i].Len() == 4 {
					s.err_list[i].Remove(s.err_list[i].Back())
				}
			}
			s.newDt[i] = new_dt
			if !badStep || new_dt == s.minDt.Scalar() {
				break //give up
			}
		}
	}
	
	// Get new timestep
	sort.Float64s(s.newDt)
	nDt := s.newDt[0]
	engine.dt.SetScalar(nDt)
	// Advance step	
	e.step.SetScalar(e.step.Scalar() + 1) // advance time step
}

func (s *BDFEulerAuto) Dependencies() (children, parents []string) {
	children = []string{"dt", "bdf_iterations", "t", "step", "badsteps"}
	parents = []string{"dt", "mindt", "maxdt"}
	for i := range s.err {
		parents = append(parents, s.maxAbsErr[i].Name())
		parents = append(parents, s.maxRelErr[i].Name())
		parents = append(parents, s.maxIter[i].Name())
		parents = append(parents, s.maxIterErr[i].Name())
	}
	return
}

// Register this module
func init() {
	RegisterModule("solver/am01", "Adaptive Adams-Moulton 0+1 solver", LoadBDFEulerAuto)
}

func LoadBDFEulerAuto(e *Engine) {
	s := new(BDFEulerAuto)
    
	// Minimum/maximum time step
	s.minDt = e.AddNewQuant("mindt", SCALAR, VALUE, Unit("s"), "Minimum time step")
	s.minDt.SetScalar(1e-38)
	s.minDt.SetVerifier(Positive)
	s.maxDt = e.AddNewQuant("maxdt", SCALAR, VALUE, Unit("s"), "Maximum time step")
	s.maxDt.SetVerifier(Positive)
	s.maxDt.SetScalar(1e38)

	s.iterations = e.AddNewQuant("bdf_iterations", SCALAR, VALUE, Unit(""), "Number of iterations per step")
	s.badSteps = e.AddNewQuant("badsteps", SCALAR, VALUE, Unit(""), "Number of time steps that had to be re-done")

	equation := e.equation
	s.ybuffer = make([]*gpu.Array, len(equation))
	s.y0buffer = make([]*gpu.Array, len(equation))
	s.y1buffer = make([]*gpu.Array, len(equation))
	s.dy0buffer = make([]*gpu.Array, len(equation))
	s.dybuffer = make([]*gpu.Array, len(equation))

	s.err = make([]*Quant, len(equation))
	s.err_list = make([]*list.List, len(equation))
	
	for i := range equation {
	    s.err_list[i] = list.New()
	}
	
	s.maxAbsErr = make([]*Quant, len(equation))
	s.maxRelErr = make([]*Quant, len(equation))
	s.maxIterErr = make([]*Quant, len(equation))
	s.maxIter = make([]*Quant, len(equation))
	s.diff = make([]gpu.Reductor, len(equation))
	s.newDt = make([]float64, len(equation))

	for i := range equation {

		eqn := &(equation[i])
		Assert(eqn.kind == EQN_PDE1)
		out := eqn.output[0]
		unit := out.Unit()
		s.err[i] = e.AddNewQuant(out.Name()+"_error", SCALAR, VALUE, unit, "Error/step estimate for "+out.Name())
		s.maxAbsErr[i] = e.AddNewQuant(out.Name()+"_maxAbsError", SCALAR, VALUE, unit, "Maximum absolute error per step for "+out.Name())
		s.maxAbsErr[i].SetScalar(1e-5)
		s.maxRelErr[i] = e.AddNewQuant(out.Name()+"_maxRelError", SCALAR, VALUE, unit, "Maximum relative error per step for "+out.Name())
		s.maxRelErr[i].SetScalar(1e-5)
		s.maxIterErr[i] = e.AddNewQuant(out.Name()+"_maxIterError", SCALAR, VALUE, unit, "The maximum error of iterator"+out.Name())
		s.maxIterErr[i].SetScalar(1e-8)
		s.maxIter[i] = e.AddNewQuant(out.Name()+"_maxIterations", SCALAR, VALUE, unit, "Maximum number of evaluations per step"+out.Name())
		s.maxIterErr[i].SetScalar(1000)
		s.diff[i].Init(out.Array().NComp(), out.Array().Size3D())

		s.maxAbsErr[i].SetVerifier(Positive)
		s.maxRelErr[i].SetVerifier(Positive)
		s.maxIterErr[i].SetVerifier(Positive)
		s.maxIter[i].SetVerifier(Positive)

		// TODO: recycle?

		y := equation[i].output[0]
		s.ybuffer[i] = Pool.Get(y.NComp(), y.Size3D())
		s.y0buffer[i] = Pool.Get(y.NComp(), y.Size3D())
		s.y1buffer[i] = Pool.Get(y.NComp(), y.Size3D())
		s.dy0buffer[i] = Pool.Get(y.NComp(), y.Size3D())
		s.dybuffer[i] = Pool.Get(y.NComp(), y.Size3D())

	}
	e.SetSolver(s)
}
