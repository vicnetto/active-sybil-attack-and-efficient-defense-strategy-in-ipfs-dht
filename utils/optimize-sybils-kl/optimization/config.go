package optimization

type CplInformation struct {
	Reliable int
	Sybil    int
}

type Config struct {
	NodesPerCplMap     map[int]CplInformation
	Top                int
	MaxKl              float64
	MinScore           float64
	MinSybils          int
	ClosestNodeIsSybil bool
}

var Flags *Config
