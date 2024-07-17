package probability

import (
	"fmt"
	"math"
)

const K = 20
const MaxCpl = 40
const KlThreshold = 0.94

func GetCplProbability() []float64 {
	probabilities := make([]float64, MaxCpl)

	probabilities[0] = 0
	probabilities[1] = 0
	probabilities[2] = 0
	probabilities[3] = 0
	probabilities[4] = 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000012133255516663741
	probabilities[5] = 0.00000000000000000000000000000000000000000000000000000000000000001864921775637621
	probabilities[6] = 0.00000000000000000000000010657634464176026
	probabilities[7] = 0.000000011802416481855667
	probabilities[8] = 0.013467386203803885
	probabilities[9] = 0.34266293270514997
	probabilities[10] = 0.32065220779026016
	probabilities[11] = 0.16160857478210483
	probabilities[12] = 0.08080444335798351
	probabilities[13] = 0.040402221680884795
	probabilities[14] = 0.020201110839691765
	probabilities[15] = 0.010100555419223462
	probabilities[16] = 0.005050277709353767
	probabilities[17] = 0.002525138854594866
	probabilities[18] = 0.0012625694272746292
	probabilities[19] = 0.0006312847136309252
	probabilities[20] = 0.0003156423568140776
	probabilities[21] = 0.00015782117840651422
	probabilities[22] = 0.0000789105892031905
	probabilities[23] = 0.00003945529460167019
	probabilities[24] = 0.00001972764730073795
	probabilities[25] = 0.000009863823650385627
	probabilities[26] = 0.000004931911825267754
	probabilities[27] = 0.00000246595591245069
	probabilities[28] = 0.0000012329779563224895
	probabilities[29] = 0.0000006164889780557736
	probabilities[30] = 0.000000308244489133358
	probabilities[31] = 0.000000154122244566679
	probabilities[32] = 0.00000007706112227778838
	probabilities[33] = 0.00000003853056114166975
	probabilities[34] = 0.00000001926528057638599
	probabilities[35] = 0.000000009632640279866322
	probabilities[36] = 0.000000004816320142708719
	probabilities[37] = 0.000000002408159965883172
	probabilities[38] = 0.0000000012040801466994822
	probabilities[39] = 0.0000000006020400178385899

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
