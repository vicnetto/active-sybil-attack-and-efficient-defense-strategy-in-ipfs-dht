package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
	"math"
)

type Position struct {
	currentCpl        int
	currentNodesInCpl int
	minimumCpl        int
	maximumCpl        int
	pathKl            float64
}

type Result struct {
	kl           float64
	score        float64
	sybilsPerCpl map[int]CplInformation
}

var Kl [][]float64

func createNodesPerCplCopy(nodesPerCpl map[int]CplInformation) map[int]CplInformation {
	nodesPerCplCopy := make(map[int]CplInformation)

	for cpl, node := range nodesPerCpl {
		nodesPerCplCopy[cpl] = node
	}

	return nodesPerCplCopy
}

// return: minCpl and reliable
func getReliableNodesInMinCpl(nodesPerCpl map[int]CplInformation, sybilsPerCpl map[int]CplInformation) (int, int) {
	totalSybils := countTotalSybils(sybilsPerCpl)

	var reliableInCpl, minCpl int
	for i := minCpl; i < probability.MaxCplProbabilitySize; i++ {
		if nodesPerCpl[i].Reliable >= totalSybils {
			minCpl = i
			reliableInCpl = nodesPerCpl[i].Reliable - totalSybils
			break
		} else {
			totalSybils -= nodesPerCpl[i].Reliable
		}
	}

	return minCpl, reliableInCpl
}

func printCpl(nodesPerCpl map[int]CplInformation, sybilsPerCpl map[int]CplInformation) {
	fmt.Printf("                 ")
	for i := 0; i < probability.MaxCplProbabilitySize; i++ {
		fmt.Printf("%4d", i)
	}
	fmt.Println()

	fmt.Printf("Nodes per CPL : ")
	fmt.Printf("[")
	cpl, nodes := getReliableNodesInMinCpl(nodesPerCpl, sybilsPerCpl)

	for i := 0; i <= probability.MaxCplProbabilitySize; i++ {
		if i < cpl {
			fmt.Printf(" %3d", 0)
		} else if i == cpl {
			fmt.Printf(" %3d", math.Mod(nodesPerCpl[i]-sybilsPerCpl[i]))
		}

		if sybilBalance != 0 && nodesPerCpl[i].Reliable >= sybilBalance {
			fmt.Printf(" %3d", nodesPerCpl[i].Reliable-sybilBalance)
			sybilBalance = 0
			continue
		}

		if sybilBalance != 0 && nodesPerCpl[i].Reliable < sybilBalance {
			fmt.Printf(" %3d", 0)
			sybilBalance -= nodesPerCpl[i].Reliable
			continue
		}

		fmt.Printf(" %3d", nodesPerCpl[i].Sybil+nodesPerCpl[i].Reliable)
	}
	fmt.Printf(" ]\n")
}

func printFullInformation(score Result) {
	// All nodes
	// printCpl(nodesPerCpl)

	// Only sybils
	fmt.Printf("Sybils per CPL: ")
	fmt.Printf("[")
	for i := 0; i <= probability.MaxCplProbabilitySize; i++ {
		fmt.Printf(" %3d", score.sybilsPerCpl[i].Sybil)
	}
	fmt.Printf(" ]\n")

	// Rest of information
	fmt.Printf("Score: %.2f, Sybils: %d, KL: %f\n", score.score, countTotalSybils(score.sybilsPerCpl), score.kl)
}

func currentNodesInCpl(nodesPerCpl map[int]CplInformation, sybilsPerCpl map[int]CplInformation, position Position) int {
	totalSybils := countTotalSybils(sybilsPerCpl)

	var quantity, cpl int
	for i := position.minimumCpl; i < position.maximumCpl; i++ {
		if nodesPerCpl[i].Reliable >= totalSybils {
			quantity = nodesPerCpl[i].Reliable - totalSybils
			cpl = i
			break
		} else {
			totalSybils -= nodesPerCpl[i].Reliable
		}
	}

	if position.currentCpl > cpl {
		return nodesPerCpl[position.currentCpl].Reliable
	} else if position.currentCpl == cpl {
		return quantity
	} else {
		return 0
	}
}

func countTotalNodes(nodesPerCpl map[int]CplInformation) int {
	var nodeCount int

	for _, node := range nodesPerCpl {
		nodeCount += node.Reliable + node.Sybil
	}

	return nodeCount
}

func countTotalSybils(nodesPerCPL map[int]CplInformation) int {
	var sybils int

	for _, node := range nodesPerCPL {
		sybils += node.Sybil
	}

	return sybils
}

func scoreCountTotal(nodesPerCpl map[int]CplInformation, sybilPerCpl map[int]CplInformation, position Position) float64 {
	minCpl := position.minimumCpl
	score := float64(0)

	for cpl := range sybilPerCpl {
		if cpl == minCpl {
			score += float64(cpl * nodesPerCpl[cpl].Reliable)
		} else {
			score += float64(cpl * sybilPerCpl[cpl].Sybil)
		}
	}

	return score
}

func isPossibleToRemoveReliableNodes(nodesPerCpl map[int]CplInformation, sybilPerCpl map[int]CplInformation, position Position) bool {
	// totalSybils := countTotalSybils(sybilPerCpl) - 1
	quantity := sybilPerCpl[position.currentCpl].Sybil
	totalSybils := countTotalSybils(sybilPerCpl) - quantity

	minCpl := position.minimumCpl

	if position.currentCpl == minCpl || quantity == 0 {
		return true
	}

	reliableInCpl := 0
	for i := minCpl; i < probability.MaxCplProbabilitySize; i++ {
		if nodesPerCpl[i].Reliable >= totalSybils {
			minCpl = i
			reliableInCpl = nodesPerCpl[i].Reliable - totalSybils
			break
		} else {
			totalSybils -= nodesPerCpl[i].Reliable
		}
	}

	for i := minCpl; i < probability.MaxCplProbabilitySize; i++ {
		if i >= position.currentCpl {
			return false
		}

		if reliableInCpl >= quantity {
			return true
		}

		if reliableInCpl < quantity {
			position.minimumCpl++
			quantity -= reliableInCpl
		}

		reliableInCpl = nodesPerCpl[i+1].Reliable
	}

	return false
}

func sybilPositionOptimization(nodesPerCpl map[int]CplInformation, sybilsPerCpl map[int]CplInformation, position Position) {
	// fmt.Println(position.currentCpl, position.currentNodesInCpl, position.maximumCpl, position.minimumCpl, countTotalNodes(nodesPerCpl))
	position.pathKl += Kl[position.currentCpl][position.currentNodesInCpl]

	// If current kl is greater than our threshold we don't continue
	if position.pathKl >= probability.KlThreshold {
		delete(sybilsPerCpl, position.currentCpl)
		return
	}

	// Try to remove actual reliable nodes to add sybils
	if position.currentNodesInCpl != 0 {
		var ok bool
		// sybilsInThisCpl := sybilInCPL(position.currentCpl, position.currentNodesInCpl, nodesPerCpl)
		ok = isPossibleToRemoveReliableNodes(nodesPerCpl, sybilsPerCpl, position)
		if !ok {
			// return nodesPerCpl, -1, position.pathKl
			delete(sybilsPerCpl, position.currentCpl)
			return
		}
	}

	// If we arrived at the last cpl, we should stop
	if position.currentCpl == position.minimumCpl {
		// Possible result
		if nodesPerCpl[19].Sybil == 1 && nodesPerCpl[18].Sybil == 1 && nodesPerCpl[17].Sybil == 1 && nodesPerCpl[16].Sybil == 1 && nodesPerCpl[14].Sybil == 1 && nodesPerCpl[13].Sybil == 2 && nodesPerCpl[12].Sybil == 1 && nodesPerCpl[11].Sybil == 0 {
			if position.currentCpl == 11 {
				fmt.Printf("")
			}
		}
		score := scoreCountTotal(nodesPerCpl, sybilsPerCpl, position)

		fmt.Println("Score:", score)
		fmt.Println("Sybils:", sybilsPerCpl)

		if score > topScores[Flags.Top-1].score && position.pathKl < Flags.MaxKl &&
			score >= Flags.MinScore && countTotalSybils(nodesPerCpl) >= Flags.MinSybils {
			addScore(score, nodesPerCpl, sybilsPerCpl, position.pathKl, position.minimumCpl)
		}

		delete(sybilsPerCpl, position.currentCpl)
		return
	}

	if countTotalNodes(nodesPerCpl) > probability.K {
		delete(sybilsPerCpl, position.currentCpl)
		return
	}
	// fmt.Println(position.currentCpl, position.currentNodesInCpl, position.maximumCpl, position.minimumCpl, countTotalNodes(nodesPerCpl))

	// if nodesPerCpl[19].Sybil == 1 && nodesPerCpl[18].Sybil == 1 && nodesPerCpl[17].Sybil == 1 && nodesPerCpl[16].Sybil == 1 && nodesPerCpl[14].Sybil == 1 && nodespercpl[13].sybil == 2 && nodesPerCpl[12].Sybil == 1 && nodesPerCpl[11].Sybil == 0 && nodespercpl[10].sybil == 5 {
	if nodesPerCpl[19].Sybil == 1 && nodesPerCpl[18].Sybil == 1 && nodesPerCpl[17].Sybil == 1 && nodesPerCpl[16].Sybil == 1 && nodesPerCpl[14].Sybil == 1 && nodesPerCpl[13].Sybil == 2 && nodesPerCpl[12].Sybil == 1 && nodesPerCpl[11].Sybil == 0 {
		if position.currentCpl == 11 {
			fmt.Printf("")
		}
	}

	position.currentCpl--
	// Recall function through the entire array with a cpl-1
	nodes := currentNodesInCpl(nodesPerCpl, sybilsPerCpl, position)
	for j := nodes; j <= probability.K; j++ {
		if Kl[position.currentCpl][j] < probability.KlThreshold {
			position.currentNodesInCpl = j

			if j-nodes != 0 {
				sybilsPerCpl[position.currentCpl] = CplInformation{Reliable: 0, Sybil: j - nodes}
			}

			sybilPositionOptimization(nodesPerCpl, sybilsPerCpl, position)
		} else {
			break
		}
	}

	return
}

func BeginSybilPositionOptimization() (map[int]CplInformation, error) {
	fmt.Println("Optimizing the sybils in the following peers configuration:")
	// todo: make this work
	// printCpl(Flags.NodesPerCpl)

	fmt.Println("\nWith the following rules:")
	fmt.Println("Top:", Flags.Top)
	fmt.Println("Max Kl:", Flags.MaxKl)
	fmt.Println("Min Score:", Flags.MinScore)
	fmt.Println("Min Sybils:", Flags.MinSybils)
	fmt.Println("Closest Node Is Sybil:", Flags.ClosestNodeIsSybil, "\n")

	topScores = make([]Result, Flags.Top)

	// todo: make this work
	// startNodesPerCpl = Flags.NodesPerCpl
	// var nodesPerCpl [probability.MaxCplProbabilitySize]int
	// if Flags.ClosestNodeIsSybil {
	// 	nodesPerCpl = addClosestSybil(Flags.NodesPerCpl)
	// } else {
	// 	nodesPerCpl = Flags.NodesPerCpl
	// }

	// startMaximumCpl := probability.MaxCplProbabilitySize - 1
	startMaximumCpl := 9

	sybilsPerCpl := map[int]CplInformation{}
	position := Position{startMaximumCpl, 0, 7, 9, 0}
	sybilPositionOptimization(Flags.NodesPerCplMap, sybilsPerCpl, position)

	position = Position{startMaximumCpl, 1, 7, 9, 0}
	sybilsPerCpl[position.currentCpl] = CplInformation{Reliable: 0, Sybil: 1}
	sybilPositionOptimization(Flags.NodesPerCplMap, sybilsPerCpl, position)

	position = Position{startMaximumCpl, 2, 7, 9, 0}
	sybilsPerCpl[position.currentCpl] = CplInformation{Reliable: 0, Sybil: 2}
	sybilPositionOptimization(Flags.NodesPerCplMap, sybilsPerCpl, position)

	position = Position{startMaximumCpl, 3, 7, 9, 0}
	sybilsPerCpl[position.currentCpl] = CplInformation{Reliable: 0, Sybil: 3}
	sybilPositionOptimization(Flags.NodesPerCplMap, sybilsPerCpl, position)

	position = Position{startMaximumCpl, 4, 7, 9, 0}
	sybilsPerCpl[position.currentCpl] = CplInformation{Reliable: 0, Sybil: 4}
	sybilPositionOptimization(Flags.NodesPerCplMap, sybilsPerCpl, position)

	fmt.Printf("> Top %d results:\n", Flags.Top)
	for i, score := range topScores {
		if score.score != 0 {
			fmt.Printf("\nResult %d)\n", i+1)
			printFullInformation(score)
		}
	}

	if len(topScores) != 0 {
		return topScores[0].sybilsPerCpl, nil
	} else {
		return nil, fmt.Errorf("no optmization available following the parameters")
	}
}
