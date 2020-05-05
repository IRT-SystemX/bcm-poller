package ingest

import (
    "log"
    "time"
    "math"
    "math/big"
    "context"
    "errors"
    "container/list"
    "reflect"
    "strconv"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/common/hexutil"
)

var (
    backupSize = 5
    retry = time.Duration(5)
    zero = big.NewInt(0)
    one = big.NewInt(1)
    hundred = big.NewInt(100)
)

type Engine struct {
    url string
    client *ethclient.Client
    status map[string]interface{}
    chainSigner types.EIP155Signer
    chain *list.List
    connector Connector
}

func NewEngine(web3Socket string) *Engine {
    return &Engine{url: web3Socket, chain: list.New(),
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

func (engine *Engine) SetCurrent(val *big.Int) {
    engine.status["current"] = val
}

func (engine *Engine) SetConnector(connector Connector) {
    engine.connector = connector
}

func (engine *Engine) Connect() *ethclient.Client {
	for {
        client, err := ethclient.Dial(engine.url)
		if err != nil {
			//log.Println(err)
			time.Sleep(retry * time.Second)
		} else {
			engine.client = client
			engine.status["connected"] = true
			chainID, err := engine.client.NetworkID(context.Background())
            if err != nil {
                log.Fatal(err)
            }
            engine.chainSigner = types.NewEIP155Signer(chainID)
			break
		}
	}
	return engine.client
}
	
func (engine *Engine) Start() error {
    if !engine.status["connected"].(bool) {
        return errors.New("Poller connection error with "+engine.url)
    }
    go func() {
		engine.init()
		engine.listen()
	}()
	return nil
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
        msg, err := tx.AsMessage(engine.chainSigner); 
        if err != nil {
            log.Fatal(err)
        }
        txEvent.Sender = msg.From().Hex()
        data := msg.Data()
        if data != nil && len(data) > 0 {
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

func (engine *Engine) init() {
    header, err := engine.client.HeaderByNumber(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Syncing to block #%s", header.Number.String())
    for i := new(big.Int).Set(engine.status["current"].(*big.Int)); i.Cmp(header.Number) < 0 || i.Cmp(header.Number) == 0; i.Add(i, one) {
        blockEvent := engine.process(i)
        if zero.Cmp(header.Number) != 0 {
            engine.status["sync"] = strconv.FormatInt(new(big.Int).Div(new(big.Int).Mul(i, hundred), header.Number).Int64(), 10)+"%%"
        } else {
            engine.status["sync"] = "100%%"
        }
        engine.apply(blockEvent)
    }
    //log.Printf("Synced %s", engine.status["sync"])
}

func (engine *Engine) listen() {
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
                blockEvent := engine.process(header.Number)
                engine.apply(blockEvent)
            }
        }
    }
}

func (engine *Engine) apply(blockEvent *BlockEvent) {
    if engine.connector != nil && !reflect.ValueOf(engine.connector).IsNil() {
        engine.connector.Apply(blockEvent)
    }
    if engine.chain.Len() >= backupSize {
        engine.chain.Remove(engine.chain.Front())
    }
    engine.chain.PushBack(blockEvent)
    engine.status["current"] = blockEvent.Number.String()
}
