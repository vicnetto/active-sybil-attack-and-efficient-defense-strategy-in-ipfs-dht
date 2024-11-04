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
	NetworkSize        int
	ScorePriority      ScorePriority
}

type ScorePriority int

const (
	Quantity ScorePriority = iota
	Distribution
	Proximity
)
const (
	QuantityString     = "quantity"
	DistributionString = "distribution"
	ProximityString    = "proximity"
)

func GetStringFromScorePriority(scorePriority ScorePriority) string {
	var scorePriorityString string

	switch scorePriority {
	case Quantity:
		scorePriorityString = QuantityString
	case Distribution:
		scorePriorityString = DistributionString
	case Proximity:
		scorePriorityString = ProximityString
	}

	return scorePriorityString
}

func GetScorePriorityFromString(scorePriorityAsString string) ScorePriority {
	var scorePriority ScorePriority

	switch scorePriorityAsString {
	case QuantityString:
		scorePriority = Quantity
	case DistributionString:
		scorePriority = Distribution
	case ProximityString:
		scorePriority = Proximity
	default:
		return -1
	}

	return scorePriority
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
	config.ScorePriority = Distribution

	return config, nil
}
