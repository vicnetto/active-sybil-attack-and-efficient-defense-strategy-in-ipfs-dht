package probability

import (
	"fmt"
	"math"
)

const K = 20
const MaxCpl = 40
const KlThreshold = 0.94

func GetCplProbability() []float64 {
	cplProbability := make([]float64, MaxCpl)

	cplProbability[0] = 0
	cplProbability[1] = 0
	cplProbability[2] = 0
	cplProbability[3] = 0.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001887
	cplProbability[4] = 0.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004876291911003931
	cplProbability[5] = 0.000000000000000000000000000000000000000000000000000000000000002444662584819837
	cplProbability[6] = 0.00000000000000000000000026729120202014326
	cplProbability[7] = 0.000000004824488839228112
	cplProbability[8] = 0.005765516773093056
	cplProbability[9] = 0.28533460862065907
	cplProbability[10] = 0.3504717648056841
	cplProbability[11] = 0.17921301711695825
	cplProbability[12] = 0.08960754391568906
	cplProbability[13] = 0.04480377197194292
	cplProbability[14] = 0.0224018859861426
	cplProbability[15] = 0.011200942992874795
	cplProbability[16] = 0.005600471496304202
	cplProbability[17] = 0.0028002357481022007
	cplProbability[18] = 0.00140011787403603
	cplProbability[19] = 0.0007000589370138995
	cplProbability[20] = 0.0003500294685058597
	cplProbability[21] = 0.000175014734252667
	cplProbability[22] = 0.00008750736712623885
	cplProbability[23] = 0.00004375368356310472
	cplProbability[24] = 0.000021876841781579004
	cplProbability[25] = 0.000010938420890758138
	cplProbability[26] = 0.000005469210445389061
	cplProbability[27] = 0.00000273460522271951
	cplProbability[28] = 0.00000136730261133422
	cplProbability[29] = 0.0000006836513056523996
	cplProbability[30] = 0.0000003418256528359142
	cplProbability[31] = 0.0000001709128264110182
	cplProbability[32] = 0.00000008545641322632581
	cplProbability[33] = 0.00000004272820658846044
	cplProbability[34] = 0.000000021364103314214235
	cplProbability[35] = 0.000000010682051642119105
	cplProbability[36] = 0.000000005341025826055556
	cplProbability[37] = 0.000000002670512924130008
	cplProbability[38] = 0.000000001335256412104968
	cplProbability[39] = 0.0000000006676282932049914

	return cplProbability
}

func GetAllPartialKl(probabilities []float64, print bool) [][]float64 {
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

	if print {
		PrintPartialKl(allPartialKl)
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
