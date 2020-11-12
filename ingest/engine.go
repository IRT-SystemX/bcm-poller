package ingest

import (
	"log"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"time"
)

var (
	retry   = time.Duration(5)
	zero    = big.NewInt(0)
	one     = big.NewInt(1)
	ten     = big.NewInt(10)
	hundred = big.NewInt(100)
)

type Engine struct {
	start          *big.Int
	end            *big.Int
	syncMode       string
	syncThreadPool int
	syncThreadSize int
	synced         int64
	mux            sync.Mutex
	status         map[string]interface{}
	Queue          chan BlockEvent
	Connector      Connector
	Processor      Processor
	RawEngine
}

func NewEngine(syncMode string, syncThreadPool int, syncThreadSize int) *Engine {
	engine := &Engine{
		start:          big.NewInt(0),
		end:            big.NewInt(-1),
		syncMode:       syncMode,
		syncThreadPool: syncThreadPool,
		syncThreadSize: syncThreadSize,
		status: map[string]interface{}{
			"connected": false,
			"sync":      "0%%",
			"current":   zero,
		},
		Queue:          make(chan BlockEvent),
	}
	return engine
}

func (engine *Engine) Status() map[string]interface{} {
	return engine.status
}

func (engine *Engine) Start() *big.Int {
	return engine.start
}

func (engine *Engine) SetStart(val string, plusOne bool) {
	if plusOne {
		start, _ := new(big.Int).SetString(val, 10)
		engine.start = new(big.Int).Add(start, one)
	} else {
		engine.start, _ = new(big.Int).SetString(val, 10)
	}
}

func (engine *Engine) SetEnd(val string) {
	engine.end, _ = new(big.Int).SetString(val, 10)
}

func (engine *Engine) SetConnector(connector Connector) {
	engine.Connector = connector
}

func (engine *Engine) SetProcessor(processor Processor) {
	engine.Processor = processor
}

func (engine *Engine) sync() {
	log.Printf("Syncing to block #%s", engine.end.String())
	if engine.end.Cmp(zero) == 0 {
		engine.synced = 100
		engine.status["sync"] = strconv.FormatInt(engine.synced, 10) + "%%"
		log.Printf("Synced %d", engine.synced)
	}
	if engine.syncMode == "normal" {
		engine.normalSync()
	} else if engine.syncMode == "fast" {
		engine.fastSync()
	} else {
		log.Fatal("Unknown sync mode %s", engine.syncMode)
	}
}

func (engine *Engine) normalSync() {
	for i := new(big.Int).Set(engine.start); i.Cmp(engine.end) < 0 || i.Cmp(engine.end) == 0; i.Add(i, one) {
		blockEvent := engine.Process(i, false)
		if blockEvent != nil {
			engine.Queue <- blockEvent
		}
		current := new(big.Int).Add(engine.start, i)
		if new(big.Int).Mod(current, ten).Cmp(zero) == 0 && current.Cmp(engine.end) != 0 {
			engine.printSync(current)
		}
	}
	engine.printSync(engine.end)
}

func (engine *Engine) fastSync() {
	size := new(big.Int).Sub(engine.end, engine.start)
	if size.Cmp(zero) > 0 {
		blockRange := big.NewInt(int64(engine.syncThreadPool * engine.syncThreadSize))
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
							blockEvent := engine.Process(i, false)
							if blockEvent != nil {
								engine.Queue <- blockEvent
							}
						}
					}(threadBegin)
				}
			}
			wg.Wait()
			engine.printSync(new(big.Int).Add(begin, blockRange))
		}
	}
}

func (engine *Engine) printSync(current *big.Int) {
	if engine.end.Cmp(zero) > 0 {
		engine.synced = new(big.Int).Div(new(big.Int).Mul(current, hundred), engine.end).Int64()
	} else {
		engine.synced = 100
	}
	if engine.synced > 100 {
		engine.synced = 100
	}
	engine.mux.Lock()
	engine.status["sync"] = strconv.FormatInt(engine.synced, 10) + "%%"
	engine.mux.Unlock()
	log.Printf("Synced %d%%", engine.synced)
}

func (engine *Engine) initialize() {
	if engine.status["connected"] == false {
		engine.status["connected"] = true
		go func() {
			for {
				select {
				case blockEvent := <-engine.Queue:
					if engine.Connector != nil && !reflect.ValueOf(engine.Connector).IsNil() {
						engine.Connector.Apply(blockEvent)
					}
					engine.mux.Lock()
					engine.status["current"] = blockEvent.Number().String()
					engine.mux.Unlock()
				}
			}
		}()
	}
}

func (engine *Engine) Init() {
	engine.initialize()
	last, err := engine.Latest()
	if err != nil {
		log.Fatal(err)
	}
	if engine.end.Cmp(zero) <= 0 {
		engine.end = last
	}
	engine.sync()
	engine.end = new(big.Int).Add(last, one)
}

func (engine *Engine) ListenProcess(number *big.Int) {
	for i := new(big.Int).Set(engine.end); i.Cmp(number) < 0 || i.Cmp(number) == 0; i.Add(i, one) {
		blockEvent := engine.Process(i, true)
		if blockEvent != nil {
			engine.Queue <- blockEvent
		}
	}
	engine.end = new(big.Int).Add(number, one)
}
