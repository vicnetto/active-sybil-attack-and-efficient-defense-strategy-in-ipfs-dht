package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
)

var topScores []Result

func addScore(result Result) {
	position := len(topScores)

	for i := 0; i < len(topScores); i++ {
		// If Score is bigger, should replace in the position.
		if result.Score > topScores[i].Score {
			position = i
			break
		}

		// If equal, should replace only if Kl is lower.
		if result.Score == topScores[i].Score && result.Kl < topScores[i].Kl {
			position = i
			break
		}
	}

	copy(topScores[position+1:], topScores[position:])
	topScores[position] = result
}

func getSybils(nodesPerCpl [probability.MaxCpl]int) [probability.MaxCpl]int {
	var sybils [probability.MaxCpl]int
	minCpl, _ := getMinAndMaxCpl(nodesPerCpl)

	for i, quantity := range nodesPerCpl {
		sybils[i] = sybilInCPL(i, quantity, minCpl)
	}

	return sybils
}

func PrintCpl(nodesPerCpl [probability.MaxCpl]int) {
	fmt.Printf("                 ")
	for i := 0; i < 40; i++ {
		fmt.Printf("%4d", i)
	}
	fmt.Println()

	fmt.Printf("Nodes per CPL : ")
	fmt.Printf("[")
	for _, node := range nodesPerCpl {
		fmt.Printf(" %3d", node)
	}
	fmt.Printf(" ]\n")
}

func PrintFullInformation(score Result) {
	// All nodes
	PrintCpl(score.NodesPerCpl)

	// Only sybils
	fmt.Printf("Sybils per CPL: ")
	fmt.Printf("[")
	for _, sybilsInCpl := range score.SybilsPerCpl {
		fmt.Printf(" %3d", sybilsInCpl)
	}
	fmt.Printf(" ]\n")

	totalSybils := 0
	for _, sybilsInCpl := range score.SybilsPerCpl {
		totalSybils += sybilsInCpl
	}

	// Rest of information
	fmt.Printf("Score: %.2f, Sybils: %d, KL: %f\n", score.Score, totalSybils, score.Kl)
}
