package hlf

import (
	metrics "github.com/IRT-SystemX/bcm-poller/internal/metrics"
	poller "github.com/IRT-SystemX/bcm-poller/poller"
	"log"
	"math/big"
)

type Tracking struct {
	Events []*metrics.Event `json:"events"`
}

type Cache struct {
	*metrics.RawCache
	Tracking *Tracking
	poller.Connector
}

func NewCache(configFile string, backupFile string, restore bool, backupFrequency int64) *Cache {
	cache := &Cache{
		RawCache: metrics.NewRawCache(backupFile, restore, backupFrequency),
		Tracking: parseConfig(configFile),
	}
	raw := cache.LoadBackup()
	if raw != nil {
		metrics.UnmarshalTrackingEvents(raw["tracking"].(map[interface{}]interface{})["events"].([]interface{}), cache.Tracking.Events)
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
		for _, tx := range blockEvent.Transactions {
			cache.Stats["transaction"].Increment(tx.Timestamp, blockEvent.Number())
			for _, event := range cache.Tracking.Events {
				var check bool = true
				for _, rule := range event.Rules() {
					check = check && cache.check(rule, tx)
				}
				if check {
					log.Printf("> detect event %s", event.Label)
					event.Increment(tx.Timestamp, blockEvent.Number())
				}
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
}

func parseConfig(config string) *Tracking {
	tracking := &Tracking{Events: make([]*metrics.Event, 0)}
	raw := metrics.LoadConfig(config)
	if raw != nil {
		events := metrics.UnmarshalEvents(raw, "events")
		for key, value := range events {
			tracking.Events = append(tracking.Events, metrics.NewEvent(key, value))
		}
	}
	return tracking
}

func (*Cache) check(rule *metrics.EventRule, obj interface{}) bool {
	tx := obj.(*TxEvent)
	switch rule.Field {
	case metrics.TO:
		return rule.Value == tx.Chaincode
	case metrics.METHOD:
		return rule.Value == tx.Method
	}
	return false
}
