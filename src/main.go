package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	io "github.com/RenatoGeh/gospn/src/io"
	learn "github.com/RenatoGeh/gospn/src/learn"
	spn "github.com/RenatoGeh/gospn/src/spn"
	utils "github.com/RenatoGeh/gospn/src/utils"
)

const dataset = "olivetti_nolabels"
const (
	width  int = 46
	height int = 56
)

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

	cmpl, _ = s.ArgMax(half)

	for k := range half {
		delete(cmpl, k)
	}

	return cmpl, half
}

func classify(filename string, p float64, rseed int64, kclusters int) (spn.SPN, int, int) {
	vars, train, test, lbls := io.ParsePartitionedData(filename, p, rseed)
	s := learn.Gens(vars, train, kclusters)

	lines, n := len(test), len(vars)
	nclass := vars[n-1].Categories

	//fmt.Println("Drawing the MPE state of each class instance:")
	//evclass := make(spn.VarSet)
	//for i := 0; i < nclass; i++ {
	//evclass[n-1] = i
	//mpe, _ := s.ArgMax(evclass)
	//filename := fmt.Sprintf("mpe_%d.pbm", i)
	//delete(mpe, n-1)
	//io.VarSetToPBM(filename, mpe, width, height)
	//fmt.Printf("Class %d drawn to %s.\n", i, filename)
	//}

	corrects := 0
	for i := 0; i < lines; i++ {
		imax, max, prs := -1, -1.0, make([]float64, nclass)
		pz := s.Value(test[i])
		fmt.Printf("Testing instance %d. Should be classified as %d.\n", i, lbls[i])
		for j := 0; j < nclass; j++ {
			test[i][n-1] = j
			px := s.Value(test[i])
			prs[j] = utils.AntiLog(px - pz)
			fmt.Printf("  Pr(X=%d|E) = antilog(%.10f) = %.10f\n", j, px-pz, prs[j])
			if prs[j] > max {
				max, imax = prs[j], j
			}
		}
		fmt.Printf("Instance %d should be classified as %d. SPN classified as %d.\n", i, lbls[i], imax)
		if imax == lbls[i] {
			corrects++
		} else {
			fmt.Printf("--------> INCORRECT! <--------\n")
		}
		delete(test[i], n-1)
	}

	fmt.Printf("\n========= Iteration Results ========\n")
	fmt.Printf("  Correct classifications: %d/%d\n", corrects, lines)
	fmt.Printf("  Percentage of correct hits: %.2f%%\n", 100.0*(float64(corrects)/float64(lines)))
	fmt.Println("======================================")

	reps := make([]map[int]int, nclass)
	for i := 0; i < lines; i++ {
		if reps[lbls[i]] == nil {
			reps[lbls[i]] = test[i]
		}
	}
	for i := 0; i < nclass; i++ {
		for _, v := range io.Orientations {
			fmt.Printf("Drawing %s completion for digit %d.\n", v, i)
			cmpl, half := halfImg(s, reps[i], v, width, height)
			io.ImgCmplToPPM(fmt.Sprintf("cmpl_%d-%s.ppm", i, v), half, cmpl, v, width, height)
		}
	}

	return s, corrects, lines
}

func imageCompletion(filename string, kclusters int) {
	fmt.Printf("Parsing data from [%s]...\n", filename)
	sc, data := io.ParseData(filename)
	ndata := len(data)

	var train []map[int]int
	for i := 0; i < ndata; i++ {
		chosen := data[i]
		for j := 0; j < ndata; j++ {
			if i != j {
				train = append(train, data[j])
			}
		}

		fmt.Printf("Training SPN with %d clusters against instance %d...\n", kclusters, i)
		s := learn.Gens(sc, train, kclusters)

		for _, v := range io.Orientations {
			fmt.Printf("Drawing %s image completion for instance %d.\n", v, i)
			cmpl, half := halfImg(s, chosen, v, width, height)
			io.ImgCmplToPGM(fmt.Sprintf("cmpl_%d-%s.pgm", i, v), half, cmpl, v, width, height)
		}
	}
}

func convertData() {
	cmn, _ := filepath.Abs("../data/" + dataset + "/")
	io.PGMFToData(cmn, "all.data")
}

func main() {
	p := 0.7
	kclusters := -1
	var rseed int64 = -1
	iterations := 1
	var err error

	if len(os.Args) > 4 {
		iterations, err = strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Printf("Argument invalid. Argument iterations must be an integer greater than zero.\n")
			return
		}
	}
	if len(os.Args) > 3 {
		kclusters, err = strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Printf("Argument invalid. Argument kcluster must be an integer.\n")
			return
		}
	}
	if len(os.Args) > 2 {
		rseed, err = strconv.ParseInt(os.Args[2], 10, 64)
		if err != nil {
			fmt.Printf("Argument invalid. Argument rseed must be a 64-bit integer.\n")
			return
		}
	}
	if len(os.Args) > 1 {
		p, err = strconv.ParseFloat(os.Args[1], 64)
		if err != nil || p < 0 || p >= 1 {
			if p == -1 {
				fmt.Printf("Converting dataset %s...", dataset)
				convertData()
				return
			}
			fmt.Printf("Argument invalid. Argument p must be a 64-bit float in the interval (0, 1).")
			return
		}
	}

	in, _ := filepath.Abs("../data/" + dataset + "/compiled")
	out, _ := filepath.Abs("../results/" + dataset + "/models")

	if p == 0 {
		fmt.Printf("Running image completion on dataset %s...\n", dataset)
		imageCompletion(utils.StringConcat(in, "/all.data"), kclusters)
		return
	}

	fmt.Printf("Running cross-validation test with p = %.2f%%, random seed = %d and kclusters = %d "+
		"on the dataset = %s.\n", 100.0*p, rseed, kclusters, dataset)
	fmt.Printf("Iterations to run: %d\n\n", iterations)

	corrects, total := 0, 0
	for i := 0; i < iterations; i++ {
		fmt.Printf("+-----------------------------------------------+\n")
		fmt.Printf("|================ Iteration %d ==================|\n", i+1)
		fmt.Printf("+-----------------------------------------------+\n")
		s, c, t := classify(utils.StringConcat(in, "/all.data"), p, rseed, kclusters)
		corrects, total = corrects+c, total+t
		fmt.Printf("+-----------------------------------------------+\n")
		fmt.Printf("|============= End of Iteration %d =============|\n", i+1)
		fmt.Printf("+-----------------------------------------------+\n")
		io.DrawGraphTools(utils.StringConcat(out, "/all.py"), s)
	}
	fmt.Printf("---------------------------------\n")
	fmt.Printf(">>>>>>>>> Final Results <<<<<<<<<\n")
	fmt.Printf("  Correct classifications: %d/%d\n", corrects, total)
	fmt.Printf("  Percentage of correct hits: %.2f%%\n", 100.0*(float64(corrects)/float64(total)))
	fmt.Printf("---------------------------------\n")
}
