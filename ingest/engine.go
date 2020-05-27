package ingest

import (
    "log"
    "time"
    "math"
    "math/big"
    "context"
    "reflect"
    "strconv"
    "sync"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/common/hexutil"
)

var (
    retry = time.Duration(5)
    zero = big.NewInt(0)
    one = big.NewInt(1)
    hundred = big.NewInt(100)
)

type Engine struct {
    url string
    client *ethclient.Client
    signer types.EIP155Signer
    start *big.Int
    end *big.Int
    syncThreadPool int
    syncThreadSize int
    synced int64
    mux sync.Mutex
    status map[string]interface{}
    queue chan *BlockEvent 
    connector Connector
    fork *ForkWatcher
}

func NewEngine(web3Socket string, syncThreadPool int, syncThreadSize int, maxForkSize int) *Engine {
    return &Engine{url: web3Socket, 
        start: big.NewInt(0),
        end: big.NewInt(-1),
        syncThreadPool: syncThreadPool,
        syncThreadSize: syncThreadSize,
        queue: make(chan *BlockEvent),
        fork: NewForkWatcher(maxForkSize),
        status: map[string]interface{}{
            "connected": false,
            "sync": "0%%",
            "current": zero,
        },
    }
}

func (engine *Engine) Client() *ethclient.Client {
    return engine.client
}

func (engine *Engine) Status() map[string]interface{} {
    return engine.status
}

func (engine *Engine) Latest() *big.Int {
    header, err := engine.client.HeaderByNumber(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }
    return header.Number
}

func (engine *Engine) SetStart(val string) {
    engine.start, _ = new(big.Int).SetString(val, 10)
}

func (engine *Engine) SetEnd(val string) {
    engine.end, _ = new(big.Int).SetString(val, 10)
}

func (engine *Engine) SetConnector(connector Connector) {
    engine.connector = connector
}

func (engine *Engine) Connect() *ethclient.Client {
	for {
        client, err := ethclient.Dial(engine.url)
		if err != nil {
			time.Sleep(retry * time.Second)
		} else {
			engine.client = client
			break
		}
	}
	engine.initialize()
	return engine.client
}

func (engine *Engine) initialize() {
    if engine.status["connected"] == false {
    	engine.status["connected"] = true
    	chainID, err := engine.client.NetworkID(context.Background())
        if err != nil {
            log.Fatal(err)
        }
        engine.signer = types.NewEIP155Signer(chainID)
        go func() {
            for {
                select {
                case blockEvent := <-engine.queue:
                    if engine.connector != nil && !reflect.ValueOf(engine.connector).IsNil() {
                        engine.connector.Apply(blockEvent)
                    }
                    engine.mux.Lock()
                    engine.status["current"] = blockEvent.Number.String()
                    engine.mux.Unlock()
                }
            }
        }()
    }
}

func (engine *Engine) process(number *big.Int) *BlockEvent {
    block, err := engine.client.BlockByNumber(context.Background(), number)
    if err != nil {
        log.Println("Error: ", err)
        log.Fatal(err)
    }
    log.Printf("Process block #%s (%s)", block.Number().String(), time.Unix(int64(block.Time()), 0).Format("2006.01.02 15:04:05"))
    blockEvent := &BlockEvent{ 
        Size: float64(block.Size()),
        Gas: float64(block.GasUsed()),
        GasLimit: float64(block.GasLimit()),
        Usage: math.Abs(float64(block.GasUsed()) * 100 / float64(block.GasLimit())),
        Timestamp: block.Time(),
        Number: block.Number(),
        Miner: block.Coinbase().Hex(),
        Transactions: make([]*TxEvent, len(block.Transactions())),
        ParentHash: block.ParentHash().Hex(),
        Hash: block.Hash().Hex(),
    }
    for i, tx := range block.Transactions() {
        //log.Printf("Process tx %s", tx.Hash().Hex())
        txEvent := &TxEvent{Events: make([]string, 0)}
        blockEvent.Transactions[i] = txEvent
        txEvent.Value = tx.Value()
        if tx.To() != nil {
            txEvent.Receiver = tx.To().Hex()
        }
        msg, err := tx.AsMessage(engine.signer); 
        if err != nil {
            log.Fatal(err)
        }
        txEvent.Sender = msg.From().Hex()
        data := msg.Data()
        if data != nil && len(data) > 4 {
            txEvent.FunctionId = string(hexutil.Encode(data[:4]))
        }
        receipt, err := engine.client.TransactionReceipt(context.Background(), tx.Hash())
        if err != nil {
            log.Fatal(err)
        }
        txEvent.Deploy = receipt.ContractAddress.Hex()
        for _, vLog := range receipt.Logs {
            for i := range vLog.Topics {
                txEvent.Events = append(txEvent.Events, vLog.Topics[i].Hex())
            }
        }
    }
    return blockEvent
}

func (engine *Engine) sync() {
    log.Printf("Syncing to block #%s", engine.end.String())
    if engine.end.Cmp(zero) == 0 {
        engine.synced = 100
        engine.status["sync"] = strconv.FormatInt(engine.synced, 10)+"%%"
        log.Printf("Synced %d", engine.synced)
    }
    size := new(big.Int).Sub(engine.end, engine.start)
    if size.Cmp(zero) > 0 {
        blockRange := big.NewInt(int64(engine.syncThreadPool*engine.syncThreadSize))
        iterMax := new(big.Int).Div(size, blockRange)
        for iter := big.NewInt(0); iter.Cmp(iterMax) < 0 || iter.Cmp(iterMax) == 0; iter.Add(iter, one) {
            begin := new(big.Int).Add(engine.start, new(big.Int).Mul(iter, blockRange))
            var wg sync.WaitGroup
            for k := 0; k < engine.syncThreadPool; k++ {
                threadBegin := new(big.Int).Add(begin, big.NewInt(int64(k*engine.syncThreadSize)))
                if threadBegin.Cmp(engine.end) <= 0 {
                    wg.Add(1)
                    go func(threadBegin *big.Int) {
                        defer wg.Done()
                        for j := 0; j < engine.syncThreadSize; j++ {
                            i := new(big.Int).Add(threadBegin, big.NewInt(int64(j)))
                            if i.Cmp(engine.end) > 0 {
                                break
                            }
                            blockEvent := engine.process(i)
                            engine.queue <- blockEvent

                        }
                    }(threadBegin)
                }
            }
            wg.Wait()
            engine.synced = new(big.Int).Div(new(big.Int).Mul(new(big.Int).Add(begin, blockRange), hundred), engine.end).Int64()
            if engine.synced > 100 {
                engine.synced = 100
            }
            engine.mux.Lock()
            engine.status["sync"] = strconv.FormatInt(engine.synced, 10)+"%%"
            engine.mux.Unlock()
            log.Printf("Synced %d%%", engine.synced)
        }
    }
}

func (engine *Engine) Init() {
    header, err := engine.client.HeaderByNumber(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }
    if engine.end.Cmp(zero) <= 0 {
        engine.end = header.Number
    }
    engine.sync()
    engine.end = new(big.Int).Add(header.Number, one)
}

func (engine *Engine) Listen() {
    headers := make(chan *types.Header)
    sub, err := engine.client.SubscribeNewHead(context.Background(), headers)
    if err != nil {
        log.Fatal(err)
    }
    for {
        select {
        case err := <-sub.Err():
            log.Fatal(err)
        case header := <-headers:
            //log.Printf("New block #%s", header.Number.String())
            if header != nil {
                for i := new(big.Int).Set(engine.end); i.Cmp(header.Number) < 0 || i.Cmp(header.Number) == 0; i.Add(i, one) {
                    blockEvent := engine.process(i)
                    engine.queue <- blockEvent
                    engine.fork.apply(blockEvent)
                }
                engine.end = new(big.Int).Add(header.Number, one)
            }
        }
    }
}
