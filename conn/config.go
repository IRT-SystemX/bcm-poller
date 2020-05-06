package conn

import (
    "log"
    "os"
    "math/big"
    "errors"
    "strings"
    "io/ioutil"
    "encoding/json"
    "gopkg.in/yaml.v2"
)

type Rule struct {
    field Field
    operator Operator
    value string
}

type Field string

const (
    FROM Field = "from"
    TO Field = "to"
    VALUE Field = "value"
    DEPLOY Field = "deploy"
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
    EQ Operator = "="
    LT Operator = "<="
    GT Operator = ">="
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

func parseConfig(config string) *Tracking {
    tracking := &Tracking{Events: make([]*Event,0), Miners: make([]*Miner,0), Balances: make([]*Balance,0)}
	_, err := os.Stat(config)
	if err == nil {
        data, err := ioutil.ReadFile(config)
        if err != nil {
            log.Fatal(err)
        }
        raw := make(map[interface{}]interface{})
        err = yaml.Unmarshal([]byte(data), &raw)
        if err != nil {
            log.Fatalf("error: %v", err)
        }
        events := unmarshalEvents(raw, "events")
        for key, value := range events {
            tracking.Events = append(tracking.Events, NewEvent(key, value))
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

func unmarshalEvents(raw map[interface{}]interface{}, field string) map[string][]*Rule {
    output := make(map[string][]*Rule)
    _, ok := raw[field]
    if !ok {
        return output
    }
    tab := raw[field].(map[interface{}]interface{})
    for key, value := range tab {
        arr := value.([]interface{})
        rules := make([]*Rule, len(arr))
        for i, val := range arr {
            words := strings.Fields(val.(string))
            if len(words) == 1 || len(words) == 3 {
                var field Field = parseField(words[0])
                if field == UNKNOWN {
                    log.Fatal(errors.New("Error parsing rule: "+val.(string)))
                }
                var operator Operator = NONE
                var value string = ""
                if len(words) == 3 {
                    operator = parseOperator(words[1])
                    if ((field == FROM || field == TO) && operator != EQ) || operator == NONE {
                        log.Fatal(errors.New("Error parsing rule: "+val.(string)))
                    }
                    value = words[2]
                }
                rules[i] = &Rule{ field: field, operator: operator, value: value }
            } else {
                log.Fatal(errors.New("Error parsing rule: "+val.(string)))
            }
        }
    	output[key.(string)] = rules
    }
    return output
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

func loadBackup(pathFile string, stats map[string]*Stats, tracking *Tracking) {
    _, err := os.Stat(pathFile)
	if err == nil {
        data, err := ioutil.ReadFile(pathFile)
        if err != nil {
            log.Fatal(err)
        }
        raw := make(map[string]interface{})
        err = yaml.Unmarshal([]byte(data), &raw)
        if err != nil {
            log.Fatalf("error: %v", err)
        }
        for key, value := range raw["stats"].(map[interface{}]interface{}) {
            unmarshalStats(key.(string), value.(map[interface{}]interface{}), stats)
        }
        unmarshalTrackingEvents(raw["tracking"].(map[interface{}]interface{})["events"].([]interface{}), tracking.Events)
        unmarshalTrackingMiners(raw["tracking"].(map[interface{}]interface{})["miners"].([]interface{}), tracking.Miners)
	}
}

func unmarshalStats(key string, value map[interface{}]interface{}, stats map[string]*Stats) {
    x, ok := stats[key];
    if !ok {
        log.Fatal(errors.New("Backup key "+key+" not found in config"))
    }
    x.Count = value["count"].(string)
    x.Current, _ = new(big.Int).SetString(x.Count, 10)
}

func unmarshalTrackingEvents(arr []interface{}, events []*Event) {
    for _, obj := range arr {
        for _, x := range events {
            if x.Label == obj.(map[interface{}]interface{})["label"] {
                x.Count = obj.(map[interface{}]interface{})["count"].(string)
                x.Current, _ = new(big.Int).SetString(x.Count, 10)
            }
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


