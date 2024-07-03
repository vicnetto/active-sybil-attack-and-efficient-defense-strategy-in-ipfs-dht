package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
)

type Position struct {
	currentCpl   int
	currentNodes int
	minimumCpl   int
	maximumCpl   int
	pathKl       float64
}

type Result struct {
	kl          float64
	score       float64
	nodesPerCpl [probability.MaxCplProbabilitySize]int
}

var Kl [][]float64

var startNodesPerCpl [probability.MaxCplProbabilitySize]int

func getSybils(nodesPerCpl [probability.MaxCplProbabilitySize]int) [probability.MaxCplProbabilitySize]int {
	var sybils [probability.MaxCplProbabilitySize]int

	for i, quantity := range nodesPerCpl {
		sybils[i] = sybilInCPL(i, quantity, nodesPerCpl)
	}

	return sybils
}

func printCpl(nodesPerCpl [probability.MaxCplProbabilitySize]int) {
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

func printFullInformation(score Result) {
	// All nodes
	printCpl(score.nodesPerCpl)

	// Only sybils
	fmt.Printf("Sybils per CPL: ")
	fmt.Printf("[")
	for i, nodesInCpl := range score.nodesPerCpl {
		fmt.Printf(" %3d", sybilInCPL(i, nodesInCpl, score.nodesPerCpl))
	}
	fmt.Printf(" ]\n")

	// Rest of information
	fmt.Printf("Score: %.2f, Sybils: %d, KL: %f\n", score.score, countTotalSybils(score.nodesPerCpl), score.kl)
}

func countNodes(nodesPerCpl [probability.MaxCplProbabilitySize]int) int {
	var nodeCount int

	for _, cpl := range nodesPerCpl {
		nodeCount += cpl
	}

	return nodeCount
}

func countTotalSybils(nodesPerCPL [probability.MaxCplProbabilitySize]int) int {
	var count int

	for i := 0; i < len(nodesPerCPL); i++ {
		if count == 0 && nodesPerCPL[i] != 0 {
			count += nodesPerCPL[i]
			continue
		}

		if count != 0 {
			count += nodesPerCPL[i] - startNodesPerCpl[i]
		}
	}

	return count
}

func getNewMinAndMaxCpl(nodesPerCPL [probability.MaxCplProbabilitySize]int) (int, int) {
	var minCpl, maxCpl, nodes int

	for i, nodesInCpl := range nodesPerCPL {
		if nodes == 0 && nodesInCpl != 0 {
			nodes += nodesInCpl
			minCpl = i

			continue
		}

		nodes += nodesInCpl

		if nodes == probability.K {
			maxCpl = i
			break
		}
	}

	return minCpl, maxCpl
}

func getNewMinCpl(nodesPerCpl [probability.MaxCplProbabilitySize]int) int {
	var newMinCpl int

	for i := 0; i < len(nodesPerCpl); i++ {
		if nodesPerCpl[i] != 0 {
			newMinCpl = i
			break
		}
	}

	return newMinCpl
}

func sybilInCPL(cpl int, currentNodesInCpl int, nodesPerCpl [probability.MaxCplProbabilitySize]int) int {
	minCpl := getNewMinCpl(nodesPerCpl)

	if cpl < minCpl {
		return 0
	}

	if minCpl == cpl {
		return currentNodesInCpl
	}

	return currentNodesInCpl - startNodesPerCpl[cpl]
}

func scoreCountTotal(pathKl float64, nodesPerCpl [probability.MaxCplProbabilitySize]int) float64 {
	_, _ = getNewMinAndMaxCpl(startNodesPerCpl)
	newMinCpl, newMaxCpl := getNewMinAndMaxCpl(nodesPerCpl)
	score := float64(0)

	// currentTotalNodes := 0

	for cpl := newMinCpl; cpl <= newMaxCpl; cpl++ {
		// if nodesPerCpl[cpl] == 0 && cpl > oldMaxCpl && cpl < newMaxCpl {
		// 	score = 0
		// 	continue
		// }

		sybilInCpl := sybilInCPL(cpl, nodesPerCpl[cpl], nodesPerCpl)
		// reliableInCpl := nodesPerCpl[cpl] - sybilInCpl
		// currentTotalNodes += reliableInCpl
		score += float64(cpl) * float64(sybilInCpl)
	}

	// score *= float64(countTotalSybils(nodesPerCpl))

	return score
}

func removeFromCpl(nodesPerCpl [probability.MaxCplProbabilitySize]int, quantity int, position Position) (bool, Position, [probability.MaxCplProbabilitySize]int) {
	nodesPerCplWithoutChanges := nodesPerCpl
	newMinCpl := getNewMinCpl(nodesPerCpl)

	if position.currentCpl == newMinCpl {
		return true, position, nodesPerCpl
	}

	for i := newMinCpl; i < len(nodesPerCpl); i++ {
		if position.currentCpl == i {
			return false, position, nodesPerCplWithoutChanges
		}

		if nodesPerCpl[i] >= quantity {
			nodesPerCpl[i] -= quantity

			return true, position, nodesPerCpl
		}

		if nodesPerCpl[i] < quantity {
			quantity -= nodesPerCpl[i]
			nodesPerCpl[i] = 0
		}
	}

	return false, position, nodesPerCpl
}

func sybilPositionOptimization(position Position, nodesPerCpl [probability.MaxCplProbabilitySize]int) {
	// fmt.Println(position.currentCpl, position.currentNodes, position.maximumCpl, position.minimumCpl, countNodes(nodesPerCpl))
	position.pathKl += Kl[position.currentCpl][position.currentNodes]

	// If current kl is greater than our threshold we don't continue
	if position.pathKl >= probability.KlThreshold {
		return
	}

	// Try to remove actual reliable nodes to add sybils
	if position.currentNodes != 0 {
		var ok bool
		sybilsInThisCpl := sybilInCPL(position.currentCpl, position.currentNodes, nodesPerCpl)

		ok, position, nodesPerCpl = removeFromCpl(nodesPerCpl, sybilsInThisCpl, position)
		if !ok {
			// return nodesPerCpl, -1, position.pathKl
			return
		}

		nodesPerCpl[position.currentCpl] = position.currentNodes
	}

	// If we arrived at the last cpl, we should stop
	if position.currentCpl == position.minimumCpl {
		// Possible result
		if countNodes(nodesPerCpl) == probability.K {
			score := scoreCountTotal(position.pathKl, nodesPerCpl)

			if score > topScores[Flags.Top-1].score && position.pathKl < Flags.MaxKl &&
				score >= Flags.MinScore && countTotalSybils(nodesPerCpl) >= Flags.MinSybils {
				addScore(Result{kl: position.pathKl, score: score, nodesPerCpl: nodesPerCpl})
			}

			return
		} else {
			//return nodesPerCpl, -1, position.pathKl
			return
		}
	}

	if countNodes(nodesPerCpl) > probability.K {
		return
	}
	// fmt.Println(position.currentCpl, position.currentNodes, position.maximumCpl, position.minimumCpl, countNodes(nodesPerCpl))

	position.currentCpl--

	// if nodesPerCpl[21] == 1 && nodesPerCpl[20] == 1 && nodesPerCpl[15] == 1 && nodesPerCpl[14] == 1 && nodesPerCpl[13] == 1 && nodesPerCpl[12] == 3 && nodesPerCpl[11] == 7 {
	// 	if position.currentCpl == 11 {
	// 		fmt.Printf("")
	// 	}
	// }

	// Recall function through the entire array with a cpl-1
	for j := nodesPerCpl[position.currentCpl]; j <= probability.K; j++ {
		if Kl[position.currentCpl][j] < probability.KlThreshold {
			position.currentNodes = j

			sybilPositionOptimization(position, nodesPerCpl)
		} else {
			break
		}
	}

	return
}

func BeginSybilPositionOptimization() ([probability.MaxCplProbabilitySize]int, error) {
	fmt.Println("Optimizing the sybils in the following peers configuration:")
	printCpl(Flags.NodesPerCpl)
	fmt.Println("\nWith the following rules:")
	fmt.Println("Top:", Flags.Top)
	fmt.Println("MaxKl:", Flags.MaxKl)
	fmt.Println("MinScore:", Flags.MinScore)
	fmt.Println("MinSybils:", Flags.MinSybils, "\n")

	topScores = make([]Result, Flags.Top)

	startNodesPerCpl = Flags.NodesPerCpl

	var startMinimumCpl, startMaximumCpl int
	startMaximumCpl = probability.MaxCplProbabilitySize - 1

	for i, cpl := range Flags.NodesPerCpl {
		if cpl != 0 && startMinimumCpl == 0 {
			startMinimumCpl = i - 1
		}
	}

	for i := Flags.NodesPerCpl[startMaximumCpl]; i < probability.K+1; i++ {
		position := Position{startMaximumCpl, i, startMinimumCpl, startMaximumCpl, 0}
		sybilPositionOptimization(position, Flags.NodesPerCpl)
	}

	fmt.Printf("> Top %d results:\n", Flags.Top)
	for i, score := range topScores {
		if score.score != 0 {
			fmt.Printf("\nResult %d)\n", i+1)
			printFullInformation(score)
		}
	}

	if len(topScores) != 0 {
		return getSybils(topScores[0].nodesPerCpl), nil
	} else {
		return [probability.MaxCplProbabilitySize]int{}, fmt.Errorf("no optmization available following the parameters")
	}
}
