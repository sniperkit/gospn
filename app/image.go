package app

import (
	"fmt"
	"github.com/RenatoGeh/gospn/common"
	"github.com/RenatoGeh/gospn/io"
	"github.com/RenatoGeh/gospn/learn"
	"github.com/RenatoGeh/gospn/learn/parameters"
	//"github.com/RenatoGeh/gospn/learn/poon"
	"github.com/RenatoGeh/gospn/learn/dennis"
	//"github.com/RenatoGeh/gospn/learn/gens"
	"github.com/RenatoGeh/gospn/spn"
	"github.com/RenatoGeh/gospn/sys"
	"github.com/RenatoGeh/gospn/test"
	"github.com/RenatoGeh/gospn/utils"
	"math"
	"runtime"
	"sync"
)

/* This file contains application functions for SPNs involving images (e.g. classification and
 * completion).
 */

// ImgBatchClassify applies ImgClassify [iterations] times, printing the results of the
// classifications.
func ImgBatchClassify(lf learn.LearnFunc, dataset string, p float64, rseed int64, clusters, iterations int) {
	fmt.Printf("Running cross-validation test with p = %.2f%%, random seed = %d and kclusters = %d "+
		"on the dataset = %s.\n", 100.0*p, rseed, clusters, dataset)
	fmt.Printf("Iterations to run: %d\n\n", iterations)
	in := io.GetDataPath(dataset)

	corrects, total := 0, 0
	for i := 0; i < iterations; i++ {
		fmt.Printf("+-----------------------------------------------+\n")
		fmt.Printf("|================ Iteration %d ==================|\n", i+1)
		fmt.Printf("+-----------------------------------------------+\n")
		c, t := ImgClassify(lf, utils.StringConcat(in, "/all.data"), p, rseed)
		corrects, total = corrects+c, total+t
		fmt.Printf("+-----------------------------------------------+\n")
		fmt.Printf("|============= End of Iteration %d =============|\n", i+1)
		fmt.Printf("+-----------------------------------------------+\n")
		//io.DrawGraphTools(utils.StringConcat(out, "/all.py"), s)
	}
	fmt.Printf("---------------------------------\n")
	fmt.Printf(">>>>>>>>> Final Results <<<<<<<<<\n")
	fmt.Printf("  Correct classifications: %d/%d\n", corrects, total)
	fmt.Printf("  Percentage of correct hits: %.2f%%\n", 100.0*(float64(corrects)/float64(total)))
	fmt.Printf("---------------------------------\n")
}

func getValue(st *spn.Storer, itk int, S spn.SPN, I spn.VarSet) float64 {
	spn.StoreInference(S, I, itk, st)
	p, _ := st.Single(itk, S)
	st.Reset(itk)
	return p
}

func testClassify(S spn.SPN, val spn.VarSet, storage *spn.Storer, itk int) {
	//fmt.Println("Starting test...")
	//fmt.Println("Calling StoreInference...")
	spn.StoreInference(S, val, itk, storage)
	//itab, _ := storage.Table(itk)
	q := &common.Queue{}
	//fmt.Println("Running BFS...")
	test.DoBFS(S, func(s spn.SPN) bool {
		//v, _ := itab.Single(s)
		v, _ := storage.Single(itk, s)
		u := s.Value(val)
		if v != u {
			fmt.Printf("Different values: v=%.5f vs u=%.5f\n", v, u)
		}
		return true
	}, q)
	//fmt.Println("Resetting...")
	storage.Reset(itk)
	//fmt.Println("Done.")
}

// ImgClassify takes an SPN S to be evaluated, a .data dataset filename to cross-evaluate, a p float
// as the percentage of the dataset to be set as train/test and rseed as the pseudo-random seed for
// data partitioning. ImgClassify returns two integers: the first is how many instances of test it
// correctly classified, and the second is the total number of instances in the test dataset.
func ImgClassify(lf learn.LearnFunc, filename string, p float64, rseed int64) (int, int) {
	vars, train, test, lbls := io.ParsePartitionedData(filename, p, rseed)
	S := lf(vars, train)
	lines, n := len(test), len(vars)
	nclass := vars[n-1].Categories

	corrects := 0
	st := spn.NewStorer()
	itk := st.NewTicket()
	sys.Println("Starting classification...")
	for i := 0; i < lines; i++ {
		imax, max, prs := -1, -1.0, make([]float64, nclass)
		pz := getValue(st, itk, S, test[i])
		sys.Printf("Testing instance %d. Should be classified as %d.\n", i, lbls[i])
		for j := 0; j < nclass; j++ {
			test[i][n-1] = j
			px := getValue(st, itk, S, test[i])
			prs[j] = utils.AntiLog(px - pz)
			sys.Printf("  Pr(X=%d|E) = antilog(%.10f) = %.10f - %.10f = %.10f\n", j, px, pz, px-pz, prs[j])
			if prs[j] > max {
				max, imax = prs[j], j
			}
		}
		sys.Printf("Instance %d should be classified as %d. SPN classified as %d.\n", i, lbls[i], imax)
		if imax == lbls[i] {
			corrects++
		} else {
			sys.Printf("--------> INCORRECT! <--------\n")
		}
		delete(test[i], n-1)
	}

	fmt.Printf("========= Iteration Results ========\n")
	fmt.Printf("  Correct classifications: %d/%d\n", corrects, lines)
	fmt.Printf("  Percentage of correct hits: %.2f%%\n", 100.0*(float64(corrects)/float64(lines)))
	fmt.Printf("  Train set size: %d\n", len(train))
	fmt.Printf("  Test set size: %d\n", len(test))
	fmt.Println("======================================")

	reps := make([]map[int]int, nclass)
	for i := 0; i < lines; i++ {
		if reps[lbls[i]] == nil {
			reps[lbls[i]] = test[i]
		}
	}

	return corrects, lines
}

func halfImg(s spn.SPN, set spn.VarSet, typ io.CmplType, w, h int) (spn.VarSet, spn.VarSet) {
	cmpl, half := make(spn.VarSet), make(spn.VarSet)
	var criteria func(int) bool

	switch typ {
	case io.Top:
		criteria = func(p int) bool {
			return p < w*(h/2)
		}
	case io.Bottom:
		criteria = func(p int) bool {
			return p >= w*(h/2)
		}
	case io.Left:
		criteria = func(p int) bool {
			return p%w < w/2
		}
	case io.Right:
		criteria = func(p int) bool {
			return p%w >= w/2
		}
	}

	for k, v := range set {
		if !criteria(k) {
			half[k] = v
		}
	}

	st := spn.NewStorer()
	_, tk, cmpl := spn.StoreMAP(s, half, -1, st)
	st.Delete(tk)

	for k := range half {
		delete(cmpl, k)
	}

	return cmpl, half
}

func randVarSet(s spn.SPN, sc map[int]*learn.Variable, n int) spn.VarSet {
	nsc := len(sc)
	vs := make(spn.VarSet)

	for i := 0; i < n; i++ {
		r := sys.RandIntn(nsc)
		id := sc[r]
		v := int(sys.RandNormFloat64()*(float64(id.Categories)/6) + float64(id.Categories/2))
		if v >= id.Categories {
			v = id.Categories - 1
		} else if v < 0 {
			v = 0
		}
		vs[id.Varid] = v
	}

	//mpe, _ := s.ArgMax(vs)
	st := spn.NewStorer()
	_, tk, mpe := spn.StoreMAP(s, vs, -1, st)
	st.Delete(tk)

	vs = nil
	return mpe
}

func completeHalfExact(S spn.SPN, E spn.VarSet) spn.VarSet {
	st := spn.NewStorer()
	jtk, etk := st.NewTicket(), st.NewTicket()
	Y := make(spn.VarSet) // Result of argmax_X P(X|E), where Y = X \union E.
	for k, v := range E {
		Y[k] = v
	}
	J := make(spn.VarSet) // Joint X=x \union E.
	for k, v := range E {
		J[k] = v
	}
	w, h := sys.Width/2, sys.Height
	for x := w - 1; x >= 0; x-- {
		for y := h - 1; y >= 0; y-- {
			spn.StoreInference(S, Y, etk, st)
			pe, _ := st.Single(etk, S) // Probability of evidence: P(E).
			p := x + y*sys.Width
			imax, max := -1, math.Inf(-1)
			//sys.Printf("Pr(E) = %.10f\n", math.Exp(pe))
			//sys.Printf("Pixel p=%d probabilities:\n", p)
			for v := 0; v < sys.Max; v++ {
				J[p] = v
				spn.StoreInference(S, J, jtk, st)
				pj, _ := st.Single(jtk, S) // Joint probability P(X, E).
				pc := pj - pe              // Conditional probability P(X | E) = P(X, E)/P(E).
				//sys.Printf("  Pr(X=%d, E) = %.10f\n", v, math.Exp(pj))
				//sys.Printf("    Pr(X=%d | E) = %.10f\n", v, math.Exp(pc))
				if pc > max {
					max, imax = pc, v
				}
				st.Reset(jtk)
			}
			Y[p], J[p] = imax, imax
			sys.Printf("Completed pixel: (%d, %d)=%d with value %d.\n", x, y, p, imax)
		}
	}
	return Y
}

func ImgTest(filename string, m, g, r int, eta, eps float64) {
	//_, D, _ := io.ParseDataNL(filename)
	sc, D, _ := io.ParseDataNL(filename)
	P := parameters.New(true, false, 0.001, parameters.HardGD, eta, eps, 1, 0.4, 4)
	for i := 0; i < len(D); i++ {
		I := D[i]
		tD := make(spn.Dataset, len(D)-1)
		var q int
		for j := range D {
			if i != j {
				tD[q] = D[j]
				q++
			}
		}
		//S := spn.NormalizeSPN(learn.PoonStructure(D, m, r))
		//S := poon.LearnGD(tD, m, g, r, eta, eps)
		S := dennis.LearnGD(tD, sc, 1, m, g, 0.95, P, i)
		//S := gens.Learn(sc, tD, 2, sys.Pval, sys.Eps, sys.Mp)
		//spn.NormalizeSPN(S)
		for _, q := range io.Quadrants {
			cmpl, half := halfImg(S, I, q, sys.Width, sys.Height)
			io.ImgCmplToPGM(fmt.Sprintf("cmpl_%s_%d.pgm", q, i), half, cmpl, io.Left, sys.Width, sys.Height, sys.Max-1)
		}
	}
}

func ImgTestParallel(filename string, m, g, r int, eta, eps float64, concurrents int) {
	//sc, D, lbls := io.ParseDataNL(filename)
	sc, D, _ := io.ParseDataNL(filename)
	ndata := len(D)
	P := parameters.New(true, false, 0.01, parameters.HardGD, eta, eps, 1, 0.1, 1)

	sys.Random.Shuffle(len(D), func(i, j int) {
		D[i], D[j] = D[j], D[i]
	})

	// Concurrency control.
	var wg sync.WaitGroup
	var nprocs int
	if concurrents <= 0 {
		nprocs = runtime.NumCPU()
	} else {
		nprocs = concurrents
	}
	nrun := 0
	cond := sync.NewCond(&sync.Mutex{})
	cpmutex := &sync.Mutex{}

	for i := 0; i < ndata; i++ {
		cond.L.Lock()
		for nrun >= nprocs {
			cond.Wait()
		}
		nrun++
		cond.L.Unlock()
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var train []map[int]int
			var ldata []map[int]int
			lsc := make(map[int]*learn.Variable)

			cpmutex.Lock()
			for k, v := range sc {
				lsc[k] = v
			}
			for j := 0; j < ndata; j++ {
				ldata = append(ldata, make(map[int]int))
				for k, v := range D[j] {
					ldata[j][k] = v
				}
			}
			cpmutex.Unlock()

			I := ldata[id]
			for j := 0; j < ndata; j++ {
				//if id != j && lbls[j] != lbls[id] {
				if id != j {
					train = append(train, ldata[j])
				}
			}

			sys.Printf("Starting job %d\n", id)
			S := dennis.LearnGD(train, lsc, 1, m, g, 0.95, P, id)
			for _, q := range io.Quadrants {
				cmpl, half := halfImg(S, I, q, sys.Width, sys.Height)
				io.ImgCmplToPGM(fmt.Sprintf("cmpl_%s_%d.pgm", q, i), half, cmpl, q, sys.Width, sys.Height, sys.Max-1)
			}
			sys.Printf("Ending job %d\n", id)

			parameters.Unbind(S)
			S, ldata, lsc, train = nil, nil, nil, nil
			sys.Free()

			cond.L.Lock()
			nrun--
			cond.L.Unlock()
			cond.Signal()
		}(i)
	}
	wg.Wait()
}

// ImgCompletion takes a LearnFunc, a dataset filename and the number of concurrent threads and
// runs an image completion job on the dataset.
func ImgCompletion(lf learn.LearnFunc, filename string, concurrents int) {
	fmt.Printf("Parsing data from [%s]...\n", filename)
	sc, data, lbls := io.ParseDataNL(filename)
	ndata := len(data)

	// Concurrency control.
	var wg sync.WaitGroup
	var nprocs int
	if concurrents <= 0 {
		nprocs = runtime.NumCPU()
	} else {
		nprocs = concurrents
	}
	nrun := 0
	cond := sync.NewCond(&sync.Mutex{})
	cpmutex := &sync.Mutex{}

	for i := 0; i < ndata; i++ {
		cond.L.Lock()
		for nrun >= nprocs {
			cond.Wait()
		}
		nrun++
		cond.L.Unlock()
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var train []map[int]int
			var ldata []map[int]int
			lsc := make(map[int]*learn.Variable)

			cpmutex.Lock()
			for k, v := range sc {
				lsc[k] = v
			}
			for j := 0; j < ndata; j++ {
				ldata = append(ldata, make(map[int]int))
				for k, v := range data[j] {
					ldata[j][k] = v
				}
			}
			cpmutex.Unlock()

			chosen := ldata[id]
			for j := 0; j < ndata; j++ {
				if id != j && lbls[j] != lbls[id] {
					train = append(train, ldata[j])
				}
			}

			fmt.Printf("P-%d: Training SPN against instance %d...\n", id, id)
			s := lf(lsc, train)

			for _, v := range io.Orientations {
				fmt.Printf("P-%d: Drawing %s image completion for instance %d.\n", id, v, id)
				cmpl, half := halfImg(s, chosen, v, sys.Width, sys.Height)
				io.ImgCmplToPGM(fmt.Sprintf("cmpl_%d-%s.pgm", id, v), half, cmpl, v, sys.Width,
					sys.Height, sys.Max-1)
				cmpl, half = nil, nil
			}
			//fmt.Printf("P-%d: Drawing MPE image for instance %d.\n", id, id)
			//io.VarSetToPGM(fmt.Sprintf("mpe_cmpl_%d.pgm", id), randVarSet(s, lsc, 100),
			//sys.Width, sys.Height, sys.Max-1)

			// Force garbage collection.
			s = nil
			train = nil
			lsc = nil
			ldata = nil
			sys.Free()

			cond.L.Lock()
			nrun--
			cond.L.Unlock()
			cond.Signal()
		}(i)
	}
	wg.Wait()
}
