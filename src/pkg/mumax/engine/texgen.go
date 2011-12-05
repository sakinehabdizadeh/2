//  This file is part of MuMax, a high-performance micromagnetic simulator.
//  Copyright 2011  Arne Vansteenkiste and Ben Van de Wiele.
//  Use of this source code is governed by the GNU General Public License version 3
//  (as published by the Free Software Foundation) that can be found in the license.txt file.
//  Note that you are welcome to modify this code under the condition that you do not remove any 
//  copyright notices and prominently state that you modified it, giving a relevant date.

package engine

// This file automatically generates LaTeX documentation
// for all registered modules.

// Author: Arne Vansteenkiste

import (
	. "mumax/common"
	cu "cuda/driver"
	"mumax/gpu"
	"strings"
	"exec"
	"runtime"
	"os"
	"io"
	"fmt"
)

func TexGen() {
	initCUDA()
	gpu.InitMultiGPU([]int{0}, 0)

	out := OpenWRONLY("modules.tex")
	defer out.Close()

	for mod := range modules {
		moduleTexGen(out, mod)
	}
}

func initCUDA() {
	Debug("Initializing CUDA")
	runtime.LockOSThread()
	Debug("Locked OS Thread")
	cu.Init()
}

func moduleTexGen(out io.Writer, module string) {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Fprintln(os.Stderr, "texgen failed", module, err)
		}
	}()

	// init dummy engine
	engine = *new(Engine) // flush global engine with zero value	
	Init()
	engine.outputDir = "."
	api := &API{&engine}
	api.SetGridSize(4, 4, 4)          // dummy size 
	api.SetCellSize(1e-9, 1e-9, 1e-9) // dummy size 
	api.Load(module)

	// save physics graph
	graphbase := "modules/" + noslash(module)
	api.SaveGraph(graphbase + ".pdf")
	err := exec.Command("mv", graphbase+".dot.pdf", graphbase+".pdf").Run()
	CheckIO(err)

	// module subsection
	fmt.Fprintln(out, `\subsection{`+module+`}`)
	fmt.Fprintln(out, `\label{`+module+`}`)
	fmt.Fprintln(out, `\index{`+module+`}`)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `Load this module with \texttt{\textbf{load}("`+module+`")}`)
	fmt.Fprintln(out, `\subsubsection*{Module description}`)
	fmt.Fprintln(out, modules[module].Description, `\\`)

	// provided quantities
	if len(engine.quantity) > 3 {
		fmt.Fprintln(out, `\subsubsection*{Module quantities}`)
		fmt.Fprintln(out, `These quantities are provided by the module:\\`)
		fmt.Fprintln(out, `\smallskip`)
		fmt.Fprintln(out, `\begin{tabular}{llll}`)
		fmt.Fprintln(out, `name & unit & comp+kind & desc \\\hline`)
		for n, q := range engine.quantity {
			if n == "t" || n == "dt" || n == "step" {
				continue
			}
			name := texEsc(q.Name())
			fmt.Fprintln(out, `\texttt{`+name+`}\index{`+name+`}`, "&", q.Unit(), "&", q.NComp(), q.kind, "&", q.desc, `\\`)
		}
		fmt.Fprintln(out, `\end{tabular}\\`)
	}

	// graph
	fmt.Fprintln(out, `\subsubsection*{Module graph}`)
	fmt.Fprintln(out, `This is the dependency graph between this module's quantities:\\`)
	fmt.Fprintln(out, `\includegraphics[height=5cm, width=\textwidth, keepaspectratio=true]{`+graphbase+`}`)

	fmt.Fprintln(out)
	fmt.Fprintln(out)
}

func noslash(str string) string {
	const ALL = -1
	str = strings.Replace(str, "/", "-", ALL)
	return str
}

func texEsc(str string) string {
	const ALL = -1
	str = strings.Replace(str, "_", `\_`, ALL)
	return str
}