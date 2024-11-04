package probability

import (
	"fmt"
	"gonum.org/v1/gonum/stat/distuv"
	"math"
)

const K = 20
const MaxCpl = 30
const KlThreshold = 0.94
const DefaultNetworkSize = 13239
const keySize = 256

// UpdateIdealDistFromNetSize returns an array containing the probabilities associated with each CPL
// - The code was extracted from the paper "Content Censorship in IPFS".
func UpdateIdealDistFromNetSize(n int) []float64 {
	orderPmfs := make([][]float64, K)
	s := make([]float64, keySize)
	for i := 0; i < K; i++ {
		orderPmfs[i] = make([]float64, keySize)
		for x := 0; x < keySize; x++ {
			b := distuv.Binomial{
				N: float64(n),
				P: math.Pow(0.5, float64(x+1)),
			}
			s[x] += b.Prob(float64(i))
			if x == 0 {
				orderPmfs[i][x] = s[x]
			} else {
				orderPmfs[i][x] = s[x] - s[x-1]
			}
		}
	}
	avgPmf := make([]float64, keySize)
	for x := 0; x < keySize; x++ {
		for i := 0; i < K; i++ {
			avgPmf[x] += orderPmfs[i][x]
		}
		avgPmf[x] /= float64(K)
	}

	return avgPmf
}

func GetCplProbability(networkSize int) []float64 {
	probabilities := UpdateIdealDistFromNetSize(networkSize)

	return probabilities
}

func GetAllPartialKl(probabilities []float64) [][]float64 {
	var allPartialKl [][]float64

	allPartialKl = make([][]float64, MaxCpl)
	for i := 0; i < MaxCpl; i++ {
		allPartialKl[i] = make([]float64, K+1)
	}

	for i := 0; i < MaxCpl; i++ {
		for j := 0; j <= K; j++ {
			prob := float64(j) / 20
			partialKl := prob * math.Log(prob/probabilities[i])

			if math.IsNaN(partialKl) {
				partialKl = 0
			}

			allPartialKl[i][j] = partialKl
		}
	}

	return allPartialKl
}

func PrintPartialKl(kl [][]float64) {
	fmt.Println("Table of KL's:")
	fmt.Printf("N° of nodes: ")
	for i := 0; i <= K; i++ {
		fmt.Printf("%7d", i)
	}

	for i := 0; i < MaxCpl; i++ {
		fmt.Printf("\n     CPL %.2d)  ", i)

		for j := 0; j <= K; j++ {
			partialKl := kl[i][j]

			if partialKl < KlThreshold {
				fmt.Printf("%6.2f ", partialKl)
			} else {
				fmt.Printf("%6s ", "X")
			}
		}
	}

	fmt.Printf("\n\n")
}
