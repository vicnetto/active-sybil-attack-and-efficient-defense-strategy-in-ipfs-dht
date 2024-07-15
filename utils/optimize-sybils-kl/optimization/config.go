package optimization

import (
	"fmt"
	"github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"
)

type Config struct {
	NodesPerCpl        [probability.MaxCpl]int
	Top                int
	MaxKl              float64
	MinKl              float64
	MinScore           float64
	MinSybils          int
	ClosestNodeIsSybil bool
}

func DefaultConfig(nodesPerCpl []int) (Config, error) {
	config := Config{}

	fixedSizeNodesPerCpl := [probability.MaxCpl]int{}

	count := 0
	for i := 0; i < probability.MaxCpl; i++ {
		quantity := nodesPerCpl[i]

		if quantity != 0 {
			count += quantity
		}

		fixedSizeNodesPerCpl[i] = quantity
	}

	if count != probability.K {
		errorMessage := fmt.Sprintf("wrong quantity of nodes. k must be = %d with maxCpl = %d",
			probability.K, probability.MaxCpl)
		return Config{}, fmt.Errorf(errorMessage)
	}

	config.NodesPerCpl = fixedSizeNodesPerCpl
	config.Top = 5
	config.MaxKl = probability.KlThreshold
	config.MinScore = -1
	config.MinSybils = -1
	config.ClosestNodeIsSybil = false

	return config, nil
}
