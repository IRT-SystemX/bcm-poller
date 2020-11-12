package utils

import (
	"encoding/json"
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
)

var (
	zero = big.NewInt(0)
	one = big.NewInt(1)
)

type Stats struct {
	Current     *big.Int `json:"-"`
	Count       string   `json:"count"`
	Interval    uint64   `json:"interval"`
	Timestamp   uint64   `json:"timestamp"`
	BlockNumber string   `json:"block"`
}

func NewStats() *Stats {
	return &Stats{Current: big.NewInt(0), Count: "0"}
}

func (stats *Stats) Increment(timestamp uint64, number *big.Int) {
	stats.Update(one, timestamp, number.String())
}

func (stats *Stats) Decrement() {
	stats.Substract(one)
}

func (stats *Stats) Add(incr *big.Int) {
	stats.Current = new(big.Int).Add(stats.Current, incr)
}

func (stats *Stats) Substract(incr *big.Int) {
	stats.Current = new(big.Int).Sub(stats.Current, incr)
}

func (stats *Stats) Update(incr *big.Int, timestamp uint64, number string) {
	stats.Add(incr)
	stats.Count = stats.Current.String()
	if stats.Timestamp != 0 && timestamp-stats.Timestamp > 0 {
		stats.Interval = timestamp - stats.Timestamp
	}
	stats.Timestamp = timestamp
	stats.BlockNumber = number
}

type Event struct {
	Stats
	rules []*EventRule
	Label string `json:"label"`
}

func (event *Event) Rules() []*EventRule {
    return event.rules
}

func NewEvent(key string, rules []*EventRule) *Event {
	event := &Event{rules: rules, Label: key}
	event.Current = big.NewInt(0)
	event.Count = "0"
	return event
}

type EventRule struct {
	Field    Field
	Operator Operator
	Value    string
}

type Field string

const (
	FROM    Field = "from"
	TO      Field = "to"
	VALUE   Field = "value"
	DEPLOY  Field = "deploy"
	UNKNOWN Field = ""
)

var fields = [...]Field{FROM, TO, VALUE, DEPLOY, UNKNOWN}

func parseField(field string) Field {
	for _, f := range fields {
		if field == string(f) {
			return f
		}
	}
	return UNKNOWN
}

type Operator string

const (
	EQ   Operator = "="
	LT   Operator = "<="
	GT   Operator = ">="
	NONE Operator = ""
)

var operators = [...]Operator{EQ, LT, GT, NONE}

func parseOperator(op string) Operator {
	for _, o := range operators {
		if op == string(o) {
			return o
		}
	}
	return NONE
}

func UnmarshalEvents(raw map[interface{}]interface{}, field string) map[string][]*EventRule {
	output := make(map[string][]*EventRule)
	_, ok := raw[field]
	if !ok {
		return output
	}
	tab := raw[field].(map[interface{}]interface{})
	for key, value := range tab {
		arr := value.([]interface{})
		rules := make([]*EventRule, len(arr))
		for i, val := range arr {
			words := strings.Fields(val.(string))
			if len(words) == 1 || len(words) == 3 {
				var field Field = parseField(words[0])
				if field == UNKNOWN {
					log.Fatal(errors.New("Error parsing rule: " + val.(string)))
				}
				var operator Operator = NONE
				var value string = ""
				if len(words) == 3 {
					operator = parseOperator(words[1])
					if ((field == FROM || field == TO) && operator != EQ) || operator == NONE {
						log.Fatal(errors.New("Error parsing rule: " + val.(string)))
					}
					value = words[2]
				}
				rules[i] = &EventRule{Field: field, Operator: operator, Value: value}
			} else {
				log.Fatal(errors.New("Error parsing rule: " + val.(string)))
			}
		}
		output[key.(string)] = rules
	}
	return output
}

func unmarshalStats(key string, value map[interface{}]interface{}, stats map[string]*Stats) {
	x, ok := stats[key]
	if !ok {
		log.Fatal(errors.New("Backup key " + key + " not found in config"))
	}
	x.Count = value["count"].(string)
	x.Current, _ = new(big.Int).SetString(x.Count, 10)
}

func storeBackup(pathFile string, data map[string]interface{}) {
	log.Println("Backuping stats..")
	jsonBytes, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	if err = ioutil.WriteFile(pathFile, jsonBytes, os.ModePerm); err != nil {
		log.Fatal("Error writing backup file", err)
	}
}

func LoadConfig(pathFile string) map[interface{}]interface{} {
	_, err := os.Stat(pathFile)
	if err == nil {
		data, err := ioutil.ReadFile(pathFile)
		if err != nil {
			log.Fatal(err)
		}
		raw := make(map[interface{}]interface{})
		err = yaml.Unmarshal([]byte(data), &raw)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		return raw
	}
	return nil
}

type RawCache struct {
	ready           bool
	backupFile      string
	backupFrequency *big.Int
	Stats           map[string]*Stats
	Backup			map[string]interface{}
}

func NewRawCache(configFile string, backupFile string, restore bool, backupFrequency int64) *RawCache {
	cache := &RawCache{
		backupFile:      backupFile,
		backupFrequency: big.NewInt(backupFrequency),
		Stats:           map[string]*Stats{"block": NewStats(), "transaction": NewStats()},
	}
	_, err := os.Stat(backupFile)
	if restore && err != nil {
		log.Fatal(err)
	}
	if !restore && err == nil {
		os.Remove(backupFile)
	}
	raw := cache.LoadBackup()
	if raw != nil {
		for key, value := range raw["stats"].(map[interface{}]interface{}) {
			unmarshalStats(key.(string), value.(map[interface{}]interface{}), cache.Stats)
		}
	}
	return cache
}

func (cache *RawCache) Ready() bool {
	return cache.ready
}

func (cache *RawCache) SetReady() {
	cache.ready = true
}

func (cache *RawCache) LoadBackup() map[string]interface{} {
	_, err := os.Stat(cache.backupFile)
	if err == nil {
		data, err := ioutil.ReadFile(cache.backupFile)
		if err != nil {
			log.Fatal(err)
		}
		raw := make(map[string]interface{})
		err = yaml.Unmarshal([]byte(data), &raw)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		return raw
	}
	return nil
}

func (cache *RawCache) Save() {
	if len(cache.backupFile) > 0 && cache.backupFrequency.Cmp(zero) != 0 && new(big.Int).Mod(cache.Stats["block"].Current, cache.backupFrequency).Cmp(zero) == 0 {
		storeBackup(cache.backupFile, cache.Backup)
	}
}

