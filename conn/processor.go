package conn

import (
	"context"
	ingest "github.com/IRT-SystemX/eth-poller/ingest"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math"
	"math/big"
)

type BlockCacheEvent struct {
	number       *big.Int
	parentHash   string
	hash         string
	Fork         bool
	Size         float64 `json:"block_size"`
	Gas          float64 `json:"block_gas"`
	GasLimit     float64 `json:"block_gas_limit"`
	Usage        float64 `json:"block_usage"`
	Interval     uint64
	Timestamp    uint64
	Miner        string
	Transactions []*TxEvent
}

func (blockEvent *BlockCacheEvent) Number() *big.Int {
	return blockEvent.number
}

func (blockEvent *BlockCacheEvent) ParentHash() string {
	return blockEvent.parentHash
}

func (blockEvent *BlockCacheEvent) Hash() string {
	return blockEvent.hash
}

func (blockEvent *BlockCacheEvent) SetFork(fork bool) {
	blockEvent.Fork = fork
}

type TxEvent struct {
	Sender     string
	Receiver   string
	Value      *big.Int
	FunctionId string
	Events     []string
	Deploy     string
}

type Processor struct {
	client *ethclient.Client
	signer types.EIP155Signer
}

func NewProcessor(client *ethclient.Client) *Processor {
	processor := &Processor{client: client}
	chainID, err := processor.client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	processor.signer = types.NewEIP155Signer(chainID)
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

func (processor *Processor) Process(block *types.Block, event ingest.BlockEvent) {
	blockEvent := interface{}(event).(*BlockCacheEvent)
	blockEvent.Size = float64(block.Size())
	blockEvent.Gas = float64(block.GasUsed())
	blockEvent.GasLimit = float64(block.GasLimit())
	blockEvent.Usage = math.Abs(float64(block.GasUsed()) * 100 / float64(block.GasLimit()))
	blockEvent.Timestamp = block.Time()
	blockEvent.Miner = block.Coinbase().Hex()
	blockEvent.Transactions = make([]*TxEvent, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		//log.Printf("Process tx %s", tx.Hash().Hex())
		txEvent := &TxEvent{Events: make([]string, 0)}
		blockEvent.Transactions[i] = txEvent
		txEvent.Value = tx.Value()
		if tx.To() != nil {
			txEvent.Receiver = tx.To().Hex()
		}
		msg, err := tx.AsMessage(processor.signer)
		if err != nil {
			log.Println("Error msg: ", err)
		} else {
			txEvent.Sender = msg.From().Hex()
			data := msg.Data()
			if len(data) > 4 {
				txEvent.FunctionId = string(hexutil.Encode(data[:4]))
			}
		}
		receipt, err := processor.client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			log.Println("Error receipt: ", err)
		} else {
			txEvent.Deploy = receipt.ContractAddress.Hex()
			for _, vLog := range receipt.Logs {
				for i := range vLog.Topics {
					txEvent.Events = append(txEvent.Events, vLog.Topics[i].Hex())
				}
			}
		}
	}
}
