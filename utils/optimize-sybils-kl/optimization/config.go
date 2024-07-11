package optimization

import "github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"

type Config struct {
	NodesPerCpl        [probability.MaxCpl]int
	Top                int
	MaxKl              float64
	MinScore           float64
	MinSybils          int
	ClosestNodeIsSybil bool
}

func defaultConfig(nodesPerCpl [probability.MaxCpl]int) Config {
	config := Config{}

	config.NodesPerCpl = nodesPerCpl
	config.Top = 5
	config.MaxKl = probability.KlThreshold
	config.MinScore = -1
	config.MinSybils = -1
	config.ClosestNodeIsSybil = false

	return config
}
