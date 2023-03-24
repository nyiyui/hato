package yuuni

// Length in um
type Length int64

type NodeID string

type Layout struct {
	LSs      map[NodeID]LineSection `json:"lines"`
	Switches map[NodeID]Switch      `json:"switches"`
	// STs      map[NodeID]ST          `json:"sts"`
}

type LineSection struct {
	Length Length
	ConnA  NodeID
	ConnB  NodeID
}

type Switch struct {
	Conns []NodeID

	// States lists what conn can connect to where.
	// The 2D slice represents all possible states, while the 1D slice represents which Node connects where.
	// For example, [ [1 0 -1] [2 -1 0] ] means that conn index 0 can connect to index 1 or index 2.
	States [][]int
}
