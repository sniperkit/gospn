package indep

// Chi-Square Independence Test

import (
	//"fmt"
	"gonum.org/v1/gonum/stat/distuv"
	"math"
)

// ChiSquare returns the cumulative distribution function at point chi, that is:
// 	Pr(X^2 <= chi)
// Where X^2 is the chi-square distribution X^2(df), with df being the degree of freedom.
func ChiSquare(chi float64, df int) float64 {
	// GoNum version.
	if df > 20000 {
		// When degrees of freedom is very high, the Chi-Squared distribution approximates a Gaussian.
		g := distuv.Normal{float64(df), float64(2 * df), nil}
		return g.CDF(chi)
	}
	cs := distuv.ChiSquared{float64(df), nil}
	return cs.CDF(chi)
}

const eps = 1e-12

// Lower incomplete gamma.
func lgamma(x, s float64, regularized bool) float64 {
	if x == 0 {
		return 0
	}
	if x < 0 || s <= 0 {
		return math.NaN()
	}

	if x > 1.1 && x > s {
		if regularized {
			return 1.0 - ugamma(x, s, regularized)
		}
		return math.Gamma(s) - ugamma(x, s, regularized)
	}

	var ft float64
	r := s
	c := 1.0
	pws := 1.0

	if regularized {
		logg, _ := math.Lgamma(s)
		ft = s*math.Log(x) - x - logg
	} else {
		ft = s*math.Log(x) - x
	}
	ft = math.Exp(ft)
	for c/pws > eps {
		r++
		c *= x / r
		pws += c
	}
	return pws * ft / s
}

// Upper incomplete gamma.
func ugamma(x, s float64, regularized bool) float64 {
	if x <= 1.1 || x <= s {
		if regularized {
			return 1 - lgamma(x, s, regularized)
		}
		return math.Gamma(s) - lgamma(x, s, regularized)
	}

	f := 1.0 + x - s
	C := f
	D := 0.0
	var a, b, chg float64

	for i := 1; i < 10000; i++ {
		a = float64(i) * (s - float64(i))
		b = float64(i<<1) + 1.0 + x - s
		D = b + a*D
		C = b + a/C
		D = 1.0 / D
		chg = C * D
		f *= chg
		if math.Abs(chg-1) < eps {
			break
		}
	}
	if regularized {
		logg, _ := math.Lgamma(s)
		return math.Exp(s*math.Log(x) - x - logg - math.Log(f))
	}
	return math.Exp(s*math.Log(x) - x - math.Log(f))
}

// Chisquare returns the p-value of Pr(X^2 > cv).
// Compare this value to the significance level assumed. If chisquare < sigval, then we cannot
// accept the null hypothesis and thus the two variables are dependent.
//
// Thanks to Jacob F. W. for a tutorial on chi-square distributions.
// Source: http://www.codeproject.com/Articles/432194/How-to-Calculate-the-Chi-Squared-P-Value
func Chisquare(df int, cv float64) float64 {
	//fmt.Println("Running chi-square...")
	if cv < 0 || df < 1 {
		return 0.0
	}

	k := float64(df) / 2.0
	x := cv / 2.0

	//if df == 1 {
	//return math.Exp(-x/2.0) / (math.Sqrt2 * math.SqrtPi * math.Sqrt(x))
	//return (math.Pow(x, (k/2.0)-1.0) * math.Exp(-x/2.0)) / (math.Pow(2, k/2.0) * math.Gamma(k/2.0))
	//return lgamma(k/2.0, x/2.0, false) / math.Gamma(k/2.0)

	//} else if df == 2 {
	if df == 2 {
		return math.Exp(-x)
	}

	//fmt.Println("Computing incomplete lower gamma function...")
	pval := lgamma(x, k, false)

	if math.IsNaN(pval) || math.IsInf(pval, 0) || pval <= 1e-8 {
		return 1e-14
	}

	//fmt.Println("Computing gamma function...")
	pval /= math.Gamma(k)

	//fmt.Println("Returning chi-square value...")
	return 1.0 - pval
}

// Chisqr gives the function Chi-Square.
func Chisqr(df int, cv float64) float64 {
	return lgamma(float64(df)/2.0, cv/2.0, false) / math.Gamma(float64(df)/2.0)
}

/*
ChiSquareTest returns whether variable x and y are statistically independent.
We use the Chi-Square test to find correlations between the two variables.
Argument data is a table with the counting of each variable category, where the first axis is
the counting of each category of variable x and the second axis of variable y. The last element
of each row and column is the total counting. E.g.:

		+------------------------+
		|      X_1 X_2 X_3 total |
		| Y_1  100 200 100  400  |
		| Y_2   50 300  25  375  |
		|total 150 500 125  775  |
		+------------------------+

Argument p is the number of categories (or levels) in x.

Argument q is the number of categories (or levels) in y.

Returns true if independent and false otherwise.
*/
func ChiSquareTest(p, q int, data [][]int, sigval float64) bool {

	// df is the degree of freedom.
	//fmt.Println("Computing degrees of freedom...")
	df := (p - 1) * (q - 1)

	// Expected frequencies
	E := make([][]float64, p)
	for i := 0; i < p; i++ {
		E[i] = make([]float64, q)
	}

	//fmt.Printf("data: %v\n", data)

	//fmt.Println("Computing expected frequencies...")
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			E[i][j] = float64(data[p][j]*data[i][q]) / float64(data[p][q])
			//fmt.Printf("E[%d][%d]: %d*%d/%d=%f\n", i, j, data[p][j], data[i][q], data[p][q], E[i][j])
		}
	}

	// Test statistic.
	//fmt.Println("Computing test statistic...")
	var chi float64
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			if E[i][j] == 0 {
				continue
			}
			diff := float64(data[i][j]) - E[i][j]
			chi += (diff * diff) / E[i][j]
		}
	}

	// Compare cmd with sigval. If cmp < sigval, then dependent. Otherwise independent.
	//fmt.Println("Computing integral of p-value on chi-square distribution...")
	cmp := ChiSquare(chi, df)

	//fmt.Println("Returning if integral >= significance value")
	//fmt.Printf("CHI: df: %d, chi: %f, cmp: %f\n", df, chi, cmp)
	//fmt.Printf("%.40f vs %.40f, %t\n", cmp, sigval, cmp >= sigval)
	return cmp >= sigval
}
