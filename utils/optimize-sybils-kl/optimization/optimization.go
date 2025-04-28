package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
)

type Position struct {
	cpl                    int
	nodesInCpl             int
	nodeCount              int
	sybils                 int
	minimumCpl             int
	maximumCpl             int
	pathKl                 float64
	closestReliableNodeCpl int
}

type Result struct {
	Kl           float64
	Score        float64
	NodesPerCpl  [probability.MaxCpl]int
	SybilsPerCpl [probability.MaxCpl]int
}

var Kl [][]float64
var startNodesPerCpl [probability.MaxCpl]int
var resultCount int

func addClosestSybil(nodesPerCpl [probability.MaxCpl]int) [probability.MaxCpl]int {
	minCpl, maxCpl := getMinAndMaxCpl(nodesPerCpl)

	nodesPerCpl[minCpl] -= 1

	if Kl[maxCpl][nodesPerCpl[maxCpl]+1] < Kl[maxCpl+1][nodesPerCpl[maxCpl+1]+1] {
		nodesPerCpl[maxCpl] += 1
	} else {
		nodesPerCpl[maxCpl+1] += 1
	}

	return nodesPerCpl
}

func getMinAndMaxCpl(nodesPerCPL [probability.MaxCpl]int) (int, int) {
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

func sybilInCPL(cpl int, currentNodesInCpl int, minCpl int) int {
	if cpl < minCpl {
		return 0
	}

	if minCpl == cpl {
		return currentNodesInCpl
	}

	return currentNodesInCpl - startNodesPerCpl[cpl]
}

func scoreCountTotal(nodesPerCpl [probability.MaxCpl]int, position Position, priority ScorePriority) float64 {
	score := float64(0)

	for cpl := position.minimumCpl; cpl <= position.maximumCpl; cpl++ {
		sybilInCpl := sybilInCPL(cpl, nodesPerCpl[cpl], position.minimumCpl)

		if sybilInCpl != 0 {
			switch priority {
			case Quantity:
				score += float64(sybilInCpl)
			case Distribution:
				score += float64(cpl) * float64(sybilInCpl)
			case Proximity:
				if cpl >= position.closestReliableNodeCpl {
					score += float64(cpl) * float64(sybilInCpl)
				}
			}
		}
	}

	return score
}

func removeFromCpl(nodesPerCpl [probability.MaxCpl]int, quantity int, position Position) (bool, Position, [probability.MaxCpl]int) {
	nodesPerCplWithoutChanges := nodesPerCpl

	if position.cpl == position.minimumCpl {
		return true, position, nodesPerCpl
	}

	for i := position.minimumCpl; i < len(nodesPerCpl); i++ {
		if position.cpl <= i {
			return false, position, nodesPerCplWithoutChanges
		}

		if nodesPerCpl[i] >= quantity {
			// Update minimum CPL according where the nodes are being removed
			if nodesPerCpl[i] > quantity {
				position.minimumCpl = i
			} else {
				position.minimumCpl = i + 1
			}

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

func sybilPositionOptimization(optimizationFlags Config, position Position, nodesPerCpl [probability.MaxCpl]int) {
	position.pathKl += Kl[position.cpl][position.nodesInCpl]

	// If current Kl is greater than our threshold we don't continue
	if position.pathKl >= probability.KlThreshold {
		return
	}

	// Try to remove actual reliable nodes to add sybils
	if position.nodesInCpl != 0 {
		var ok bool
		sybilsInThisCpl := sybilInCPL(position.cpl, position.nodesInCpl, position.minimumCpl)

		ok, position, nodesPerCpl = removeFromCpl(nodesPerCpl, sybilsInThisCpl, position)
		if !ok {
			return
		}

		// If first CPL with nodes, we should set the maximum CPL.
		if position.maximumCpl == 0 {
			position.minimumCpl = position.cpl
		}

		nodesPerCpl[position.cpl] = position.nodesInCpl
		position.nodeCount += position.nodesInCpl

		if position.cpl != position.minimumCpl {
			position.sybils += position.nodesInCpl - startNodesPerCpl[position.cpl]
		}
	}

	// If we arrived at the last cpl, we should stop
	if position.cpl == position.minimumCpl {
		position.sybils += nodesPerCpl[position.cpl]

		if position.nodeCount == probability.K {
			resultCount++
			score := scoreCountTotal(nodesPerCpl, position, optimizationFlags.ScorePriority)

			if score > topScores[optimizationFlags.Top-1].Score && position.pathKl < optimizationFlags.MaxKl &&
				score >= optimizationFlags.MinScore && position.pathKl >= optimizationFlags.MinKl &&
				position.sybils >= optimizationFlags.MinSybils {

				addScore(Result{Kl: position.pathKl, Score: score, NodesPerCpl: nodesPerCpl})
			}

			return
		} else {
			return
		}
	}

	if position.nodeCount > probability.K {
		return
	}

	position.cpl--

	// Recall function through the entire array with a cpl-1
	for j := nodesPerCpl[position.cpl]; j <= probability.K; j++ {
		if Kl[position.cpl][j] < probability.KlThreshold {
			position.nodesInCpl = j

			sybilPositionOptimization(optimizationFlags, position, nodesPerCpl)
		} else {
			break
		}
	}

	return
}

func BeginSybilPositionOptimization(optimizationFlags Config) ([]Result, error) {
	probabilities := probability.GetCplProbability(optimizationFlags.NetworkSize)
	Kl = probability.GetAllPartialKl(probabilities)

	topScores = make([]Result, optimizationFlags.Top)

	startNodesPerCpl = optimizationFlags.NodesPerCpl

	var nodesPerCpl [probability.MaxCpl]int
	if optimizationFlags.ClosestNodeIsSybil {
		nodesPerCpl = addClosestSybil(optimizationFlags.NodesPerCpl)
	} else {
		nodesPerCpl = optimizationFlags.NodesPerCpl
	}

	var startMinimumCpl, startMaximumCpl int
	startMaximumCpl = probability.MaxCpl - 1

	nodeCount, closestReliableNodeCpl := 0, 0
	for cpl, nodes := range optimizationFlags.NodesPerCpl {
		nodeCount += nodes

		if nodes != 0 && startMinimumCpl == 0 {
			startMinimumCpl = cpl
		}

		if nodeCount == probability.K {
			closestReliableNodeCpl = cpl
			break
		}
	}

	for i := optimizationFlags.NodesPerCpl[startMaximumCpl]; i < probability.K+1; i++ {
		position := Position{startMaximumCpl, i, 0, 0, startMinimumCpl,
			startMaximumCpl, 0, closestReliableNodeCpl}
		sybilPositionOptimization(optimizationFlags, position, nodesPerCpl)
	}
	fmt.Println("Results:", resultCount)

	if len(topScores) != 0 {
		for i, score := range topScores {
			if score.Score != 0 {
				topScores[i].SybilsPerCpl = getSybils(score.NodesPerCpl)
			}
		}

		return topScores, nil
	} else {
		return nil, fmt.Errorf("no optmization available following the parameters")
	}
}
