package ingest

import (
	"math/big"
)

type BlockEvent interface {
	Number() *big.Int
}

type RawEngine interface {
    Latest() (*big.Int, error)
    Process(number *big.Int, listening bool) BlockEvent
    Listen()
}

type Connector interface {
	Apply(interface{})
	Revert(interface{})
}

type Processor interface {
	NewBlockEvent(*big.Int, string, string) BlockEvent
	Process(interface{}, BlockEvent)
}
