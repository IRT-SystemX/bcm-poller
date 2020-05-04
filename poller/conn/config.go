package conn

import (
    "log"
    "os"
    "errors"
    "strings"
    "io/ioutil"
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
	_, err := os.Stat(config)
	if err != nil {
		log.Fatal("Config file is missing: ", config)
	}
    data, err := ioutil.ReadFile(config)
    if err != nil {
        log.Fatal(err)
    }
    raw := make(map[interface{}]interface{})
    err = yaml.Unmarshal([]byte(data), &raw)
    if err != nil {
        log.Fatalf("error: %v", err)
    }
    tracking := &Tracking{Events: make([]*Event,0), Miners: make([]*Miner,0), Balances: make([]*Balance,0)}
    events := unmarshalEvents(raw, "events")
    for key, value := range events {
        tracking.Events = append(tracking.Events, &Event{rules: value, Label: key})
    }
    miners := unmarshalAddress(raw, "miners")
    for key, value := range miners {
        tracking.Miners = append(tracking.Miners, &Miner{Id: value, Label: key})
    }
    balances := unmarshalAddress(raw, "balances")
    for key, value := range balances {
        tracking.Balances = append(tracking.Balances, &Balance{Id: value, Label: key})
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

