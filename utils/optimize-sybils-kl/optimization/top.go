package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"reflect"
	"strings"
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

func PrintUsefulCpl(nodesPerCpl interface{}, nodeType string) {
	var minCpl, maxCpl int
	nodes := reflect.ValueOf(nodesPerCpl)

	label := nodeType + " per CPL: "

	if nodes.Kind() == reflect.Slice || nodes.Kind() == reflect.Array {
		var count int64

		for i := 0; i < nodes.Len(); i++ {
			if count == 0 && nodes.Index(i).Int() != 0 {
				minCpl = i
			}

			if nodes.Index(i).Int() != 0 {
				maxCpl = i
			}

			count += nodes.Index(i).Int()
		}

		fmt.Print(strings.Repeat(" ", len(label)) + " ")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf("%4d", i)
		}
		fmt.Println()

		fmt.Printf(label)
		fmt.Printf("[")
		for i := minCpl; i <= maxCpl; i++ {
			fmt.Printf(" %3d", nodes.Index(i).Int())
		}
		fmt.Printf(" ]\n")
	}
}

func PrintFullInformation(score Result) {
	// All nodes
	PrintUsefulCpl(score.NodesPerCpl, "Nodes")

	// Only sybils
	PrintUsefulCpl(score.SybilsPerCpl, "Sybil")
	fmt.Println()

	totalSybils := 0
	for _, sybilsInCpl := range score.SybilsPerCpl {
		totalSybils += sybilsInCpl
	}

	// Rest of information
	fmt.Printf("Score: %.2f, Sybils: %d, KL: %f\n", score.Score, totalSybils, score.Kl)
}
