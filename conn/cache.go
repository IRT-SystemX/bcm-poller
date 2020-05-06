package conn

import (
    "log"
    "os"
    "math/big"
    "context"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    ingest "github.com/IRT-SystemX/eth-poller/ingest"
)

var (
    zero = big.NewInt(0)
    one = big.NewInt(1)
)

type Stats struct {
    Current *big.Int `json:"-"`
    Count string `json:"count"`
    Interval uint64 `json:"interval"`
    Timestamp uint64 `json:"timestamp"`
    BlockNumber string `json:"block"`
}

func NewStats() *Stats {
    return &Stats{Current: big.NewInt(0), Count: "0"}
}

func (stats *Stats) increment(blockEvent *ingest.BlockEvent) {
    stats.update(one, blockEvent.Timestamp, blockEvent.Number.String())
}

func (stats *Stats) update(incr *big.Int, timestamp uint64, number string) {
    stats.Current = new(big.Int).Add(stats.Current, incr)
    stats.Count = stats.Current.String()
    if stats.Timestamp != 0 &&  timestamp - stats.Timestamp > 0 {
        stats.Interval = timestamp - stats.Timestamp
    }
    stats.Timestamp = timestamp
    stats.BlockNumber = number
}

type Event struct {
    Stats
    rules []*Rule
    Label string `json:"label"`
}

func NewEvent(key string, rules []*Rule) *Event {
    event := &Event{rules: rules, Label: key}
    event.Current = big.NewInt(0)
    event.Count = "0"
    return event
}

type Miner struct {
    Stats
    Id string `json:"id"`
    Label string `json:"label"`
}

func NewMiner(key string, id string) *Miner {
    miner := &Miner{Id: id, Label: key}
    miner.Current = big.NewInt(0)
    miner.Count = "0"
    return miner
}

type Balance struct {
    Id string `json:"id"`
    Label string `json:"label"`
    Balance string `json:"balance"`
}

type Tracking struct {
    Events []*Event `json:"events"`
    Miners []*Miner `json:"miners"`
    Balances []*Balance `json:"balances"`
}

type Cache struct {
    client *ethclient.Client
    ready bool
    backupFile string
    backupFrequency *big.Int
    Stats map[string]*Stats
    Tracking *Tracking
}

func NewCache(client *ethclient.Client, configFile string, backupFile string, restore bool, backupFrequency int64) *Cache {
    cache := &Cache{
        client: client,
        backupFile: backupFile,
        backupFrequency: big.NewInt(backupFrequency),
        Stats: map[string]*Stats{"block": NewStats(), "transaction": NewStats(), "fork": NewStats()},
        Tracking: parseConfig(configFile),
    }
    _, err := os.Stat(backupFile)
    if restore && err != nil {
        log.Fatal(err)
    }
	if !restore && err == nil {
	    os.Remove(backupFile)
	}
    loadBackup(backupFile, cache.Stats, cache.Tracking)
    return cache
}

func (cache *Cache) SetReady() {
    cache.ready = true
}

func (rule *Rule) check(tx *ingest.TxEvent) bool {
    switch rule.field {
    case FROM:
        val := common.HexToAddress(rule.value).Hex()
        return val == tx.Sender
    case TO:
        val := common.HexToAddress(rule.value).Hex()
        return val == tx.Receiver
    case VALUE:
        val := new(big.Int).SetBytes([]byte(rule.value))
        switch rule.operator {
            case EQ:
                return tx.Value.Cmp(val) == 0
            case GT:
                return tx.Value.Cmp(val) >= 0
            case LT:
                return tx.Value.Cmp(val) <= 0
        }
    case DEPLOY:
        return tx.Deploy != "0x0000000000000000000000000000000000000000"
    }
    return false
}

func (cache *Cache) Apply(blockEvent *ingest.BlockEvent) {
    cache.Stats["block"].increment(blockEvent)
    if len(blockEvent.Transactions) > 0 {
        cache.Stats["transaction"].update(big.NewInt(int64(len(blockEvent.Transactions))), blockEvent.Timestamp, blockEvent.Number.String())
        for _, tx := range blockEvent.Transactions {
            for _, event := range cache.Tracking.Events {
                var check bool = true
                for _, rule := range event.rules {
                    check = check && rule.check(tx)
                }
                if check {
                    log.Printf("> detect event %s", event.Label)
                    event.increment(blockEvent)
                }
            }
        }
    }
    if blockEvent.Fork {
        cache.Stats["fork"].increment(blockEvent)
    }
    for _, miner := range cache.Tracking.Miners {
        val := common.HexToAddress(miner.Id).Hex()
        if val == blockEvent.Miner {
            log.Printf("> detect miner %s", miner.Label)
            miner.increment(blockEvent)
        }
    }
    if cache.ready {
        for _, balance := range cache.Tracking.Balances {
            res, err := cache.client.BalanceAt(context.Background(), common.HexToAddress(balance.Id), nil)
            if err != nil {
              log.Fatal(err)
            }
            balance.Balance = res.String()
        }
    }
    if len(cache.backupFile) > 0 && cache.backupFrequency.Cmp(zero) != 0 && new(big.Int).Mod(cache.Stats["block"].Current, cache.backupFrequency).Cmp(zero) == 0 {
        storeBackup(cache.backupFile, map[string]interface{}{"stats": cache.Stats,"tracking": cache.Tracking})
    }
}

