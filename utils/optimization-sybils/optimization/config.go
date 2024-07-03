package optimization

import "optimization-sybils/probability"

type Config struct {
	NodesPerCpl [probability.MaxCplProbabilitySize]int
	Top         int
	MaxKl       float64
	MinScore    float64
	MinSybils   int
}

var Flags *Config
