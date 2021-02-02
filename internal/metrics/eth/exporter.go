package eth

import (
	utils "github.com/IRT-SystemX/bcm-poller/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
	"strconv"
	"time"
)

type Measure struct {
	name        string
	help        string
	valueType   prometheus.ValueType
	value       float64
	labelsName  []string
	labelsValue []string
}

func NewMeasure(name string, desc string, valueType prometheus.ValueType, value float64, labels map[string]interface{}) *Measure {
	measure := &Measure{name: name, help: desc, valueType: valueType, value: value}
	if labels != nil {
		measure.labelsName = make([]string, len(labels))
		measure.labelsValue = make([]string, len(labels))
		i := 0
		for key, val := range labels {
			measure.labelsName[i] = key
			if val != nil {
				measure.labelsValue[i] = val.(string)
			} else {
				measure.labelsValue[i] = ""
			}
			i++
		}
	}
	return measure
}

func (measure *Measure) desc() *prometheus.Desc {
	return prometheus.NewDesc(measure.name, measure.help, measure.labelsName, nil)
}

func (measure *Measure) metric() prometheus.Metric {
	return prometheus.MustNewConstMetric(measure.desc(), measure.valueType, measure.value, measure.labelsValue...)
}

type ExporterCache struct {
	*Cache
	startTime time.Time
	measures  map[string]*Measure
	fetcher   *utils.Fetcher
}

func NewExporterCache(client *ethclient.Client, fetcher *utils.Fetcher, configFile string, backupFile string, restore bool, backupFrequency int64) *ExporterCache {
	cache := &ExporterCache{
		Cache:     NewCache(client, configFile, backupFile, restore, backupFrequency),
		startTime: time.Now(),
		measures:  make(map[string]*Measure),
		fetcher:   fetcher,
	}
	cache.update(nil)
	return cache
}

func (cache *ExporterCache) Describe(ch chan<- *prometheus.Desc) {
	for _, val := range cache.measures {
		ch <- val.desc()
	}
}

func (cache *ExporterCache) Collect(ch chan<- prometheus.Metric) {
	for _, val := range cache.measures {
		ch <- val.metric()
	}
}

func (cache *ExporterCache) set(name string, desc string, valueType prometheus.ValueType, value float64, labels map[string]interface{}) {
	cache.measures[name] = NewMeasure(name, desc, valueType, value, labels)
}

func (cache *ExporterCache) updateInfos() {
	version := cache.fetcher.Get("parity_versionInfo").(map[string]interface{})
	version_num := version["version"].(map[string]interface{})
	labels := map[string]interface{}{
		"parity_chain_name":   cache.fetcher.Get("parity_chain"),
		"parity_enode":        cache.fetcher.Get("parity_enode"),
		"parity_node_name":    cache.fetcher.Get("parity_nodeName"),
		"parity_version":      utils.FloatToString(version_num["major"].(float64)) + "." + utils.FloatToString(version_num["minor"].(float64)) + "." + utils.FloatToString(version_num["patch"].(float64)),
		"parity_version_hash": version["hash"].(string),
		"web3_version":        cache.fetcher.Get("web3_clientVersion"),
		"eth_coinbase":        cache.fetcher.Get("eth_coinbase"),
		"is_mining":           strconv.FormatBool(cache.fetcher.Get("eth_mining").(bool)),
		"is_listening":        strconv.FormatBool(cache.fetcher.Get("net_listening").(bool)),
		"net_peerCount":       cache.fetcher.Get("net_peerCount"),
		"poller_uptime":       utils.IntToString(int64(time.Now().Sub(cache.startTime).Seconds())),
	}
	cache.set("parity_node_info", "infos", prometheus.CounterValue, 1, labels)
}

func (cache *ExporterCache) updateFromApi() {
	cache.set("eth_gas_price", "gas_price", prometheus.GaugeValue, utils.StringToFloat(utils.Decode(cache.fetcher.Get("eth_gasPrice").(string)).String()), nil)
	cache.set("eth_hashrate", "hashrate", prometheus.GaugeValue, utils.StringToFloat(utils.Decode(cache.fetcher.Get("eth_hashrate").(string)).String()), nil)
	cache.set("parity_pendingTransactions", "pending", prometheus.GaugeValue, float64(len(cache.fetcher.Get("parity_pendingTransactions").([]interface{}))), map[string]interface{}{
		"node_pending_transactions_limit": utils.FloatToString(cache.fetcher.Get("parity_transactionsLimit").(float64)),
	})
	syncing := cache.fetcher.Get("eth_syncing")
	if reflect.TypeOf(syncing).Name() == "bool" {
		cache.set("eth_syncing", "sync", prometheus.GaugeValue, 1, map[string]interface{}{
			"eth_sync_starting": nil,
			"eth_sync_highest":  nil,
		})
	} else {
		syncingMap := syncing.(map[string]interface{})
		cache.set("eth_syncing", "sync", prometheus.GaugeValue, utils.StringToFloat(utils.Decode(syncingMap["currentBlock"].(string)).String()), map[string]interface{}{
			"eth_sync_starting": syncingMap["startingBlock"].(string),
			"eth_sync_highest":  syncingMap["highestBlock"].(string),
		})
	}
	peers := cache.fetcher.Get("parity_netPeers").(map[string]interface{})
	cache.set("parity_node_peers", "peers", prometheus.GaugeValue, peers["connected"].(float64), map[string]interface{}{
		"peers_max":       utils.FloatToString(peers["max"].(float64)),
	})
}

func (cache *ExporterCache) updateFromBlock(event interface{}) {
	if event != nil {
		block := interface{}(event).(*BlockCacheEvent)
		labels := map[string]interface{}{
			"eth_block_number":     block.Number().String(),
			"eth_block_timestamp":  utils.IntToString(int64(block.Timestamp())),
			"eth_block_hash":       block.Hash(),
			"eth_block_parent":     block.ParentHash(),
			"eth_block_gas_limit":  utils.FloatToString(block.GasLimit),
			"eth_block_miner":      block.Miner,
			"eth_block_difficulty": block.difficulty,
			"eth_block_uncles":     utils.IntToString(int64(block.uncles)),
		}
		cache.set("eth_block_info", "block", prometheus.CounterValue, 1, labels)
		cache.set("eth_block_transactions", "transactions", prometheus.GaugeValue, float64(len(block.Transactions)), nil)
		cache.set("eth_block_usage", "usage", prometheus.GaugeValue, block.Usage, nil)
		cache.set("eth_block_size", "size", prometheus.GaugeValue, block.Size, nil)
		cache.set("eth_block_gas", "gas", prometheus.GaugeValue, block.Gas, nil)
	} else {
		cache.set("eth_block_info", "block", prometheus.CounterValue, 1, nil)
		cache.set("eth_block_transactions", "transactions", prometheus.GaugeValue, 0, nil)
		cache.set("eth_block_usage", "usage", prometheus.GaugeValue, 0, nil)
		cache.set("eth_block_size", "size", prometheus.GaugeValue, 0, nil)
		cache.set("eth_block_gas", "gas", prometheus.GaugeValue, 0, nil)
	}
}

func (cache *ExporterCache) updateFromCache() {
	cache.set("poller_block_interval", "block_interval", prometheus.GaugeValue, float64(cache.Stats["block"].Interval), nil)
	cache.set("poller_transaction_interval", "transaction_interval", prometheus.GaugeValue, float64(cache.Stats["transaction"].Interval), nil)
	cache.set("poller_fork", "fork", prometheus.GaugeValue, float64(cache.Stats["fork"].Interval), nil)
	for _, event := range cache.Tracking.Events {
		cache.set("poller_tracking_events_"+event.Label, "events", prometheus.GaugeValue, float64(event.Stats.Interval), nil)
	}
	for _, event := range cache.Tracking.Miners {
		cache.set("poller_tracking_miners_"+event.Label, "miners", prometheus.GaugeValue, float64(event.Stats.Interval), nil)
	}
	for _, event := range cache.Tracking.Balances {
		cache.set("poller_tracking_balances_"+event.Label, "balances", prometheus.GaugeValue, utils.StringToFloat(event.Balance), nil)
	}
}

func (cache *ExporterCache) update(event interface{}) {
	cache.updateInfos()
	cache.updateFromApi()
	cache.updateFromBlock(event)
	cache.updateFromCache()
}

func (cache *ExporterCache) Apply(event interface{}) {
	cache.Cache.Apply(event)
	cache.update(event)
}

func (cache *ExporterCache) Revert(event interface{}) {
	cache.Cache.Revert(event)
	cache.update(event)
}

func (cache *ExporterCache) SetReady() {
	cache.Cache.SetReady()
}
