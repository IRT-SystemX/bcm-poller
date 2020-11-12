package hlf

import (
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	utils "github.com/IRT-SystemX/bcm-poller/internal"
	"math/big"
)

type Cache struct {
	*utils.RawCache
	ingest.Connector
}

func NewCache(configFile string, backupFile string, restore bool, backupFrequency int64) *Cache {
	cache := &Cache{
		RawCache: utils.NewRawCache(configFile, backupFile, restore, backupFrequency),
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
	}
}

func (cache *Cache) Revert(event interface{}) {
	blockEvent := interface{}(event).(*BlockCacheEvent)
	cache.Stats["block"].Decrement()
	if len(blockEvent.Transactions) > 0 {
		cache.Stats["transaction"].Substract(big.NewInt(int64(len(blockEvent.Transactions))))
	}
}
