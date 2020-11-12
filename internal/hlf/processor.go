package hlf

import (
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	"math/big"
)

type BlockCacheEvent struct {
	number       *big.Int
	hash         string
	parentHash   string
	Interval     uint64
	timestamp    uint64
	Transactions []*TxEvent
	ingest.BlockEvent
}

func (blockEvent *BlockCacheEvent) Number() *big.Int {
	return blockEvent.number
}

func (blockEvent *BlockCacheEvent) Timestamp() uint64 {
	return blockEvent.timestamp
}

func (blockEvent *BlockCacheEvent) Hash() string {
	return blockEvent.hash
}

func (blockEvent *BlockCacheEvent) ParentHash() string {
	return blockEvent.parentHash
}

type TxEvent struct {
	Sender     string
	Receiver   string
	Value      *big.Int
	FunctionId string
}

type Processor struct {
}

func NewProcessor() *Processor {
	processor := &Processor{}
	return processor
}

func (processor *Processor) NewBlockEvent(number *big.Int, parentHash string, hash string) ingest.BlockEvent {
	blockEvent := &BlockCacheEvent{
		number:     number,
		parentHash: parentHash,
		hash:       hash,
	}
	return interface{}(blockEvent).(ingest.BlockEvent)
}

func (processor *Processor) Process(obj interface{}, event ingest.BlockEvent, listening bool) {
	/*
	block := obj.(*types.Block)
	blockEvent := interface{}(event).(*BlockCacheEvent)
	blockEvent.timestamp = block.Time()
	blockEvent.Transactions = make([]*TxEvent, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		//log.Printf("Process tx %s", tx.Hash().Hex())
		txEvent := &TxEvent{}
		blockEvent.Transactions[i] = txEvent
	}
	*/
}
