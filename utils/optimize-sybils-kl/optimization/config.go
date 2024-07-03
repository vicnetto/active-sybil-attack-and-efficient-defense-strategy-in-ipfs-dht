package optimization

import "github.com/vicnetto/active-sybil-attack/utils/optimize-sybils-kl/probability"

type Config struct {
	NodesPerCpl [probability.MaxCplProbabilitySize]int
	Top         int
	MaxKl       float64
	MinScore    float64
	MinSybils   int
}

var Flags *Config
