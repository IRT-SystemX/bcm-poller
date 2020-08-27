package ingest

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type BlockEvent interface {
	Number() *big.Int
	ParentHash() string
	Hash() string
	SetFork(bool)
}

type Connector interface {
	Apply(BlockEvent)
	Revert(BlockEvent)
}

type Processor interface {
	NewBlockEvent(*big.Int, string, string) BlockEvent
	Process(*types.Block, BlockEvent)
}
