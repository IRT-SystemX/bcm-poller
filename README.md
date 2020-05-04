
# eth-poller

Ethereum event poller with RESTful API.
It connects to an Ethereum node and it gathers information about the network.
In particular, it listens for blocks and it keeps count of specific events or of mined blocks. 
It also keeps balance information for specific accounts.

Events and tracked addresses are described in a configuration file such as:
```
events:
    my_nonce: # Keep count of every transaction from the specified address
        - "from = 0x649fFFa0d1b8E3959BED0a7F15f510b959aD4128"

    my_calls: # Keep count of every transaction to the specified address
        - "to = 0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d"

    my_credit: # Keep count of faucet transaction from the specified address and with a value >= 1 ether
        - "from = 0x649fFFa0d1b8E3959BED0a7F15f510b959aD4128"
        - "value >= 1000000000000000000"

    total_deploy: # Keep count of every smart contract deployment
        - "deploy"

miners: # Keep count of every block mined by the specified address
    my_validator: "0x649fFFa0d1b8E3959BED0a7F15f510b959aD4128"

balances: # Track balance for the specified address
    master: "0x1005388E1649240036d199B6ad71EafC0164edAd"
```

Information are available on the exposed REST API and labeled according to the configuration:
* URL: /tracking
* URL: /stats
* URL: /disk
* URL: /status

## Getting Started

* Build docker image
```
docker build --target install -t eth-poller $PWD
```

* Run and connect to node
```
docker run -it --rm --name poller -p 8000:8000 -v $PWD:/root eth-poller --url ws://node:8546 --config /root/config.yml
```

* Test api
```
curl -H "Content-Type: application/json" -XGET localhost:8000/status
```

## Development

* Run go env
```
docker run -it --rm --name dev -p 8000:8000 -v $PWD:/go/src -w /go/src golang:1.14 bash
$ go run cmd/main.go --url ws://localhost:8546
```
