package eth

import (
	"context"
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	utils "github.com/IRT-SystemX/bcm-poller/internal"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
)

type Miner struct {
	utils.Stats
	Id           string `json:"id"`
	Label        string `json:"label"`
	CurrentBlock string `json:"currentBlock"`
}

func NewMiner(key string, id string) *Miner {
	miner := &Miner{Id: id, Label: key, CurrentBlock: ""}
	miner.Current = big.NewInt(0)
	miner.Count = "0"
	return miner
}

type Balance struct {
	Id      string `json:"id"`
	Label   string `json:"label"`
	Balance string `json:"balance"`
}

type Tracking struct {
	Events   []*utils.Event   `json:"events"`
	Miners   []*Miner   `json:"miners"`
	Balances []*Balance `json:"balances"`
}

type Cache struct {
	*utils.RawCache
	Tracking *Tracking
	client *ethclient.Client
	ingest.Connector
}

func NewCache(client *ethclient.Client, configFile string, backupFile string, restore bool, backupFrequency int64) *Cache {
	cache := &Cache{
		RawCache: utils.NewRawCache(configFile, backupFile, restore, backupFrequency),
		Tracking: parseConfig(configFile),
		client: client,
	}
	cache.RawCache.Stats["fork"] = utils.NewStats()
	raw := cache.LoadBackup()
	if raw != nil {
		utils.UnmarshalTrackingEvents(raw["tracking"].(map[interface{}]interface{})["events"].([]interface{}), cache.Tracking.Events)
		unmarshalTrackingMiners(raw["tracking"].(map[interface{}]interface{})["miners"].([]interface{}), cache.Tracking.Miners)
	}
	return cache
}

func (cache *Cache) SetReady() {
	cache.RawCache.SetReady()
}

func (cache *Cache) Apply(event interface{}) {
	blockEvent := interface{}(event).(*BlockCacheEvent)
	cache.Stats["block"].Increment(blockEvent.Timestamp(), blockEvent.Number())
	if len(blockEvent.Transactions) > 0 {
		cache.Stats["transaction"].Update(big.NewInt(int64(len(blockEvent.Transactions))), blockEvent.Timestamp(), blockEvent.Number().String())
		for _, tx := range blockEvent.Transactions {
			for _, event := range cache.Tracking.Events {
				var check bool = true
				for _, rule := range event.Rules() {
					check = check && cache.check(rule, tx)
				}
				if check {
					log.Printf("> detect event %s", event.Label)
					event.Increment(blockEvent.Timestamp(), blockEvent.Number())
				}
			}
		}
	}
	if blockEvent.Fork {
		cache.Stats["fork"].Increment(blockEvent.Timestamp(), blockEvent.Number())
	}
	for _, miner := range cache.Tracking.Miners {
		val := common.HexToAddress(miner.Id).Hex()
		if val == blockEvent.Miner {
			log.Printf("> detect miner %s", miner.Label)
			miner.Increment(blockEvent.Timestamp(), blockEvent.Number())
		}
		miner.CurrentBlock = blockEvent.Number().String()
	}
	if cache.RawCache.Ready() {
		for _, balance := range cache.Tracking.Balances {
			res, err := cache.client.BalanceAt(context.Background(), common.HexToAddress(balance.Id), nil)
			if err != nil {
				log.Println("Error: ", err)
			} else {
				balance.Balance = res.String()
			}
		}
	}
	cache.RawCache.Save()
}

func (cache *Cache) Revert(event interface{}) {
	blockEvent := interface{}(event).(*BlockCacheEvent)
	cache.Stats["block"].Decrement()
	if len(blockEvent.Transactions) > 0 {
		cache.Stats["transaction"].Substract(big.NewInt(int64(len(blockEvent.Transactions))))
		for _, tx := range blockEvent.Transactions {
			for _, event := range cache.Tracking.Events {
				var check bool = true
				for _, rule := range event.Rules() {
					check = check && cache.check(rule, tx)
				}
				if check {
					//log.Printf("> revert event %s", event.Label)
					event.Decrement()
				}
			}
		}
	}
	for _, miner := range cache.Tracking.Miners {
		val := common.HexToAddress(miner.Id).Hex()
		if val == blockEvent.Miner {
			//log.Printf("> revert miner %s", miner.Label)
			miner.Decrement()
		}
	}
}

func unmarshalTrackingMiners(arr []interface{}, miners []*Miner) {
	for _, obj := range arr {
		for _, x := range miners {
			if x.Label == obj.(map[interface{}]interface{})["label"] {
				x.Count = obj.(map[interface{}]interface{})["count"].(string)
				x.Current, _ = new(big.Int).SetString(x.Count, 10)
			}
		}
	}
}

func unmarshalAddress(raw map[interface{}]interface{}, field string) map[string]string {
	output := make(map[string]string)
	_, ok := raw[field]
	if !ok {
		return output
	}
	tab := raw[field].(map[interface{}]interface{})
	for key, value := range tab {
		output[key.(string)] = value.(string)
	}
	return output
}

func parseConfig(config string) *Tracking {
	tracking := &Tracking{Events: make([]*utils.Event, 0), Miners: make([]*Miner, 0), Balances: make([]*Balance, 0)}
	raw := utils.LoadConfig(config)
	if raw != nil {
		events := utils.UnmarshalEvents(raw, "events")
		for key, value := range events {
			tracking.Events = append(tracking.Events, utils.NewEvent(key, value))
		}
		miners := unmarshalAddress(raw, "miners")
		for key, value := range miners {
			tracking.Miners = append(tracking.Miners, NewMiner(key, value))
		}
		balances := unmarshalAddress(raw, "balances")
		for key, value := range balances {
			tracking.Balances = append(tracking.Balances, &Balance{Id: value, Label: key})
		}
	}
	return tracking
}

func (*Cache) check(rule *utils.EventRule, obj interface{}) bool {
	tx := obj.(*TxEvent)
	switch rule.Field {
	case utils.FROM:
		val := common.HexToAddress(rule.Value).Hex()
		return val == tx.Sender
	case utils.TO:
		val := common.HexToAddress(rule.Value).Hex()
		return val == tx.Receiver
	case utils.VALUE:
		val, _ := new(big.Int).SetString(rule.Value, 10)
		switch rule.Operator {
		case utils.EQ:
			return tx.Value.Cmp(val) == 0
		case utils.GT:
			return tx.Value.Cmp(val) >= 0
		case utils.LT:
			return tx.Value.Cmp(val) <= 0
		}
	case utils.DEPLOY:
		return tx.Deploy != "0x0000000000000000000000000000000000000000"
	}
	return false
}
