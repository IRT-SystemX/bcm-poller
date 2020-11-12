package eth

import (
	"context"
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
	"reflect"
	"time"
)

var (
	retry   = time.Duration(5)
)

type rpcBlockHash struct {
	Hash common.Hash `json:"hash"`
}

type EthBlockEvent interface {
	Number() *big.Int
	ParentHash() string
	Hash() string
	SetFork(bool)
}

type EthEngine struct {
	*ingest.Engine
	url            string
	client         *ethclient.Client
	rawClient      *rpc.Client
	fork           *ForkWatcher
}

func NewEthEngine(web3Socket string, syncMode string, syncThreadPool int, syncThreadSize int, maxForkSize int) *ingest.Engine {
	engine := &EthEngine{
		Engine: ingest.NewEngine(syncMode, syncThreadPool, syncThreadSize, maxForkSize),
		url: web3Socket,
	}
	engine.Engine.RawEngine = engine
	return engine.Engine
}

func (engine *EthEngine) Client() *ethclient.Client {
	return engine.client
}

func (engine *EthEngine) Connect() *ethclient.Client {
	for {
		rawClient, err := rpc.DialContext(context.Background(), engine.url)
		if err != nil {
			time.Sleep(retry * time.Second)
		} else {
			engine.client = ethclient.NewClient(rawClient)
			engine.rawClient = rawClient
			break
		}
	}
	engine.Initialize()
	return engine.client
}

func (engine *EthEngine) Latest() (*big.Int, error) {
	header, err := engine.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, err
	} else {
		return header.Number, nil
	}
}

func (engine *EthEngine) Process(number *big.Int, listening bool) ingest.BlockEvent {
	block, err := engine.client.BlockByNumber(context.Background(), number)
	if err != nil {
		log.Println("Error block: ", err)
		return nil
	}
	var head rpcBlockHash
	err = engine.rawClient.CallContext(context.Background(), &head, "eth_getBlockByNumber", hexutil.EncodeBig(number), false)
	if err != nil {
		log.Println("Error block hash: ", err)
	}
	log.Printf("Process block #%s (%s) %s", block.Number().String(), time.Unix(int64(block.Time()), 0).Format("2006.01.02 15:04:05"), head.Hash.Hex())
	event := engine.Processor.NewBlockEvent(block.Number(), block.ParentHash().Hex(), head.Hash.Hex())
	blockEvent := event.(EthBlockEvent)
	blockEvent.SetFork(false)
	if engine.Processor != nil && !reflect.ValueOf(engine.Processor).IsNil() {
		engine.Processor.Process(block, blockEvent)
	}
	if listening {
		engine.fork.checkFork(blockEvent)
		engine.fork.apply(blockEvent)
	}
	return blockEvent.(ingest.BlockEvent)
}

func (engine *EthEngine) Listen() {
	headers := make(chan *types.Header)
	sub, err := engine.client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case err := <-sub.Err():
			log.Println("Error: ", err)
		case header := <-headers:
			//log.Printf("New block #%s", header.Number.String())
			if header != nil {
				engine.ListenProcess(header.Number)
			}
		}
	}
}
