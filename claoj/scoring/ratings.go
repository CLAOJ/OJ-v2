package scoring

import (
	"math"
	"sync"
)

var (
	Beta2         = math.Pow(328.33, 2)
	RatingInit    = 1200.0
	MeanInit      = 1500.0
	VarInit       = math.Pow(200, 2) * (Beta2 / math.Pow(212, 2))
	SdInit        = math.Sqrt(VarInit)
	ValidRangeL   = MeanInit - 20*SdInit
	ValidRangeR   = MeanInit + 20*SdInit
	VarPerContest = 1219.047619 * (Beta2 / math.Pow(212, 2))
	VarLim        = (math.Sqrt(math.Pow(VarPerContest, 2)+4*Beta2*VarPerContest) - VarPerContest) / 2
	SdLim         = math.Sqrt(VarLim)
	TanhC         = math.Sqrt(3) / math.Pi
)

type TanhTerm struct {
	Mu float64
	Sd float64
	Wt float64
}

func evalTanhs(tanhTerms []TanhTerm, x float64) float64 {
	sum := 0.0
	for _, term := range tanhTerms {
		sum += (term.Wt / term.Sd) * math.Tanh((x-term.Mu)/(2*term.Sd))
	}
	return sum
}

func solve(tanhTerms []TanhTerm, yTg float64, linFactor float64, bounds [2]float64) float64 {
	L, R := bounds[0], bounds[1]
	var Ly, Ry *float64

	// Binary search for roots of linFactor*x + evalTanhs(tanhTerms, x) - yTg = 0
	for R-L > 2 {
		x := (L + R) / 2
		y := linFactor*x + evalTanhs(tanhTerms, x)
		if y > yTg {
			R = x
			Ry = &y
		} else if y < yTg {
			L = x
			Ly = &y
		} else {
			return x
		}
	}

	if Ly == nil {
		y := linFactor*L + evalTanhs(tanhTerms, L)
		Ly = &y
	}
	if yTg <= *Ly {
		return L
	}
	if Ry == nil {
		y := linFactor*R + evalTanhs(tanhTerms, R)
		Ry = &y
	}
	if yTg >= *Ry {
		return R
	}
	ratio := (yTg - *Ly) / (*Ry - *Ly)
	return L*(1-ratio) + R*ratio
}

var (
	varCache      = []float64{VarInit}
	varCacheMutex sync.Mutex
)

func getVar(timesRanked int) float64 {
	varCacheMutex.Lock()
	defer varCacheMutex.Unlock()
	for timesRanked >= len(varCache) {
		lastVar := varCache[len(varCache)-1]
		nextVar := 1.0 / (1.0/(lastVar+VarPerContest) + 1.0/Beta2)
		varCache = append(varCache, nextVar)
	}
	return varCache[timesRanked]
}

// RecalculateRatings implements the Elo-MMR algorithm.
func RecalculateRatings(ranking []float64, oldMean []float64, timesRanked []int, historicalP [][]float64) ([]int, []float64, []float64) {
	n := len(ranking)
	newP := make([]float64, n)
	newMean := make([]float64, n)

	if n < 1 {
		return nil, nil, nil
	}

	if n < 2 {
		for i := range oldMean {
			newP[i] = oldMean[i]
			newMean[i] = oldMean[i]
		}
	} else {
		delta := make([]float64, n)
		pTanhTerms := make([]TanhTerm, n)
		for i := 0; i < n; i++ {
			delta[i] = TanhC * math.Sqrt(getVar(timesRanked[i])+VarPerContest+Beta2)
			pTanhTerms[i] = TanhTerm{Mu: oldMean[i], Sd: delta[i], Wt: 1.0}
		}

		solveIdx := func(i int, bounds [2]float64) {
			r := ranking[i]
			yTg := 0.0
			for j := 0; j < n; j++ {
				if ranking[j] > r {
					yTg += 1.0 / delta[j]
				} else if ranking[j] < r {
					yTg -= 1.0 / delta[j]
				}
			}
			newP[i] = solve(pTanhTerms, yTg, 0, bounds)
		}

		var divconq func(i, j int)
		divconq = func(i, j int) {
			if j-i > 1 {
				k := (i + j) / 2
				solveIdx(k, [2]float64{newP[j], newP[i]})
				divconq(i, k)
				divconq(k, j)
			}
		}

		solveIdx(0, [2]float64{ValidRangeL, ValidRangeR})
		solveIdx(n-1, [2]float64{ValidRangeL, ValidRangeR})
		divconq(0, n-1)

		for i := 0; i < n; i++ {
			tanhTerms := make([]TanhTerm, 0, len(historicalP[i])+1)
			wPrev := 1.0
			wSum := 0.0

			// Newest performance first
			currentHist := append([]float64{newP[i]}, historicalP[i]...)
			for j, h := range currentHist {
				gamma2 := 0.0
				if j > 0 {
					gamma2 = VarPerContest
				}
				hVar := getVar(timesRanked[i] + 1 - j)
				k := hVar / (hVar + gamma2)
				w := wPrev * k * k
				tanhTerms = append(tanhTerms, TanhTerm{Mu: h, Sd: math.Sqrt(Beta2) * TanhC, Wt: w})
				wPrev = w
				wSum += w / Beta2
			}
			w0 := 1.0/getVar(timesRanked[i]+1) - wSum
			p0 := evalTanhs(tanhTerms[1:], oldMean[i])/w0 + oldMean[i]
			newMean[i] = solve(tanhTerms, w0*p0, w0, [2]float64{ValidRangeL, ValidRangeR})
		}
	}

	newRating := make([]int, n)
	for i := 0; i < n; i++ {
		r := math.Max(1, math.Round(newMean[i]-(math.Sqrt(getVar(timesRanked[i]+1))-SdLim)))
		newRating[i] = int(r)
	}

	return newRating, newMean, newP
}
