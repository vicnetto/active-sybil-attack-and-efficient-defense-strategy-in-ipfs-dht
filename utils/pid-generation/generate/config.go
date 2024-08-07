package generate

import "time"

var Timeout = 120 * time.Second

const MaxProbabilities = 40

type NodePerCpl struct {
	Cpl      int
	Quantity int
}

type PidGenerateConfig struct {
	ByInterval bool
	ByClosest  bool
	ByCpl      bool
	UseAllCpus bool

	FirstPeer  string
	SecondPeer string

	Cpl         int
	NodesPerCpl []NodePerCpl

	Quantity  int
	Cid       string
	FirstPort int
	OutFile   string
}
