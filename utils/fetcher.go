package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
)

type Fetcher struct {
	host string
}

func NewFetcher(host string) *Fetcher {
	_, err := http.Post(host, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return nil
	}
	return &Fetcher{host: host}
}

func request(host string, params map[string]interface{}) map[string]interface{} {
	buf, err := json.Marshal(params)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Post(host, "application/json", bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var doc map[string]interface{}
	err = json.Unmarshal(body, &doc)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func (fetcher *Fetcher) Get(method string) interface{} {
	return request(fetcher.host, map[string]interface{}{
		"method":  method,
		"params":  make([]string, 0),
		"id":      1,
		"jsonrpc": "2.0",
	})["result"]
}

func (fetcher *Fetcher) GetBalance(address string) *big.Int {
	return Decode(request(fetcher.host, map[string]interface{}{
		"method":  "eth_getBalance",
		"params":  [1]string{address},
		"id":      1,
		"jsonrpc": "2.0",
	})["result"].(string))
}
