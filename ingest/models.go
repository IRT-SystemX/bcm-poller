package ingest

import (
    "math/big"
)

type BlockEvent struct {
    Size float64 `json:"block_size"`
    Gas float64 `json:"block_gas"`
    GasLimit float64 `json:"block_gas_limit"`
    Usage float64 `json:"block_usage"`
    Interval uint64
    Timestamp uint64
    Number *big.Int
    Miner string
    Transactions []*TxEvent
    ParentHash string
    Hash string
    Fork bool
}

type TxEvent struct {
    Sender string
    Receiver string
    Value *big.Int
    FunctionId string
    Events []string
    Deploy string
}

type Connector interface {
    Apply(*BlockEvent)
    Revert(*BlockEvent)
}
