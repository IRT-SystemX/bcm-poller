package conn

import (
    "log"
    "math/big"
    "context"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    ingest "github.com/IRT-SystemX/eth-poller/ingest"
)

type Stats struct {
    Count int64 `json:"count"`
    Interval uint64 `json:"interval"`
    Timestamp uint64 `json:"timestamp"`
    BlockNumber string `json:"block"`
}

func (stats *Stats) increment(blockEvent *ingest.BlockEvent) {
    stats.update(1, blockEvent.Timestamp, blockEvent.Number.String())
}

func (stats *Stats) update(count int, timestamp uint64, number string) {
    stats.Count += int64(count)
    if stats.Timestamp != 0 {
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

type Miner struct {
    Stats
    Id string `json:"id"`
    Label string `json:"label"`
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
    Stats map[string]*Stats
    Tracking *Tracking
}

func NewCache(client *ethclient.Client, configFile string) *Cache {
    return &Cache{
        client: client,
        Stats: map[string]*Stats{"block": &Stats{}, "transaction": &Stats{}, "fork": &Stats{}},
        Tracking: parseConfig(configFile),
    }
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
        cache.Stats["transaction"].update(len(blockEvent.Transactions), blockEvent.Timestamp, blockEvent.Number.String())
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
    for _, balance := range cache.Tracking.Balances {
        res, err := cache.client.BalanceAt(context.Background(), common.HexToAddress(balance.Id), nil)
        if err != nil {
          log.Fatal(err)
        }
        balance.Balance = res.String()
    }
}
