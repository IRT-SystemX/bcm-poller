
# bcm-poller

Event poller with RESTful API for both Ethereum and Hyperledger Fabric.

## Ethereum

It connects to an Ethereum node and it gathers information about the network.
In particular, it listens for blocks and it keeps count of specific events or mined blocks into counters available throught a REST API.
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

It is able to detect block reorganisations and it updates the counters according to the new current chain.
It also backups the different counters periodically in order to be able resync from a particular block number in case of crash.
The options in command line allows to configure the poller behaviour:
```
Usage:
  poller eth [flags]

Flags:
  -h, --help                 help for poller
      --url string           Address web3 (default "ws://localhost:8546")
      --config string        Config file (default "config.yml")
      --port int             Port to run server on (default 8000)
      --restore              Restore counters from the backup
      --backup int           Backup frequency in number of blocks (default 0, no backup)
      --backupPath string    Backup file path (default "backup.json")
      --ledgerPath string    Monitored ledger path on disk (default "/chain")
      --start string         Sync start block (default "0", "-1" means from the last backuped block)
      --end string           Sync end block (default "-1": latest block in the chain)
      --syncMode string      Sync mode (fast or normal) (default "normal", fast uses threads)
      --syncThreadPool int   Nb of thread for the sync (default 4)
      --syncThreadSize int   Nb of blocks per thread per sync round (default 25)
```

Information are available on the exposed REST API and labeled according to the configuration:

* `curl -XGET http://localhost:8000/tracking`
```
{
    "events": [
    	{
    	        "label": "total_deploy",                            // event's label from the config
    	        "count": "1",                                       // number of occurence of the event
    	        "interval": 3,                                      // delay in seconds since last update of the event
    	        "timestamp": 1592920752,                            // timestamp of the block corresponding to the last event
    	        "block": "7"                                        // number of the block corresponding to the last event
    	}
    ],
    "miners": [
    	{
    	        "id": "0x649fFFa0d1b8E3959BED0a7F15f510b959aD4128", // miner's address
    	        "label": "my_validator",                            // miner's label from the config
    	        "count": "7",                                       // nb of blocks that have been mined by the address
    	        "interval": 3,                                      // delay in seconds since last update of the event
    	        "timestamp": 1592920752,                            // timestamp of the block corresponding to the last event
    	        "block": "7",                                       // number of the block corresponding to the last event
    	        "currentBlock": "7"                                 // latest block number of the chain
    	}
    ],
    "balances": [
    	{
    	        "id": "0x1005388E1649240036d199B6ad71EafC0164edAd", // account's address
    	        "label": "master",                                  // account's label from the config
    	        "balance": "1000000000000000000000000000000000"     // account's balance
    	}
    ]
}
```

* `curl -XGET http://localhost:8000/stats`
```
{
        "block": {
                "count": "7",               // number of blocks in the chain (if the poller is 100% synced)
                "interval": 3,              // delay in seconds since last update
                "timestamp": 1592920752,    // timestamp of the block corresponding to the last update
                "block": "7"                // number of the block corresponding to the last update
        },
        "transaction": {
                "count": "1",               // number of transactions in the chain
                "interval": 3,              // delay in seconds since last update
                "timestamp": 1592920752,    // timestamp of the block corresponding to the last update
                "block": "7"                // number of the block corresponding to the last update
        },
        "fork": {
                "count": "0",               // number of detected reorganisations
                "interval": 0,              // delay in seconds since last update
                "timestamp": 0,             // timestamp of the block corresponding to the last update
                "block": ""                 // number of the block corresponding to the last update
        }
}
```

* `curl -XGET http://localhost:8000/disk`
```
{
        "used": 284,                        // disk usage (KB)
        "size": 10485760,                   // size (KB) of the disk (10Go)
        "usage": 0,                         // percentage of disk used
        "free": 10485476,                   // disk free (KB)
        "available": 10485476,              // disk available (KB)
        "dir": 213                          // size (KB) of the chain directory (default: /chain)
}
```

* `curl -XGET http://localhost:8000/status`
```
{
        "connected": true,                  // connectivity status of the socket
        "current": "7",                     // latest block number of the chain
        "sync": "100%"                      // percentage of synchronization of the poller
}
```

The API is exposed by a server that listens by default on port 8000.
It uses Websocket interface to collect the metrics. Although, it was only tested with [Open Ethereum](https://github.com/openethereum/openethereum).

## Hyperledger Fabric

It connects to an Hyperledger Fabric peer and it gathers information about the network.
In particular, it listens for blocks and it keeps count of specific events or mined blocks into counters available throught a REST API.
It also keeps balance information for specific accounts.

Events and tracked addresses are described in a configuration file such as:
```
events:
    my_chaincode: # Keep count of every transaction to the specified chaincode
        - "to = mycc"

    my_method: # Keep count of every transaction to the specified chaincode and method
        - "to = mycc"
        - "method = init"
```

It is able to detect block reorganisations and it updates the counters according to the new current chain.
It also backups the different counters periodically in order to be able resync from a particular block number in case of crash.
The options in command line allows to configure the poller behaviour:
```
Usage:
  poller hlf [flags]

Flags:
  -h, --help                 help for poller
      --path string          Path hlf files (default "/tmp/hyperledger-fabric-network")
      --config string        Config file (default "config.yml")
      --port int             Port to run server on (default 8000)
      --restore              Restore counters from the backup
      --backup int           Backup frequency in number of blocks (default 0, no backup)
      --backupPath string    Backup file path (default "backup.json")
      --ledgerPath string    Monitored ledger path on disk (default "/chain")
      --start string         Sync start block (default "0", "-1" means from the last backuped block)
      --end string           Sync end block (default "-1": latest block in the chain)
      --syncMode string      Sync mode (fast or normal) (default "normal", fast uses threads)
      --syncThreadPool int   Nb of thread for the sync (default 4)
      --syncThreadSize int   Nb of blocks per thread per sync round (default 25)
```

Information are available on the exposed REST API and labeled according to the configuration:

* `curl -XGET http://localhost:8000/tracking`
```
{
    "events": [
    	{
    	        "label": "my_chaincode",                            // event's label from the config
    	        "count": "1",                                       // number of occurence of the event
    	        "interval": 3,                                      // delay in seconds since last update of the event
    	        "timestamp": 1592920752,                            // timestamp of the block corresponding to the last event
    	        "block": "7"                                        // number of the block corresponding to the last event
    	}
    ]
}
```

* `curl -XGET http://localhost:8000/stats`
```
{
        "block": {
                "count": "7",               // number of blocks in the chain (if the poller is 100% synced)
                "interval": 3,              // delay in seconds since last update
                "timestamp": 1592920752,    // timestamp of the block corresponding to the last update
                "block": "7"                // number of the block corresponding to the last update
        },
        "transaction": {
                "count": "1",               // number of transactions in the chain
                "interval": 3,              // delay in seconds since last update
                "timestamp": 1592920752,    // timestamp of the block corresponding to the last update
                "block": "7"                // number of the block corresponding to the last update
        }
}
```

* `curl -XGET http://localhost:8000/disk`
```
{
        "used": 284,                        // disk usage (KB)
        "size": 10485760,                   // size (KB) of the disk (10Go)
        "usage": 0,                         // percentage of disk used
        "free": 10485476,                   // disk free (KB)
        "available": 10485476,              // disk available (KB)
        "dir": 213                          // size (KB) of the chain directory (default: /chain)
}
```

* `curl -XGET http://localhost:8000/status`
```
{
        "connected": true,                  // connectivity status of the socket
        "current": "7",                     // latest block number of the chain
        "sync": "100%"                      // percentage of synchronization of the poller
}
```

The API is exposed by a server that listens by default on port 8000.
It uses hyperledger fabric files to connect a gateway and collect the metrics.

## Getting Started

* Build the docker image
```
docker build --target install -t bcm-poller $PWD
```

* Run and connect to node
```
docker run -it --rm --name poller -p 8000:8000 -v $PWD:/backup bcm-poller --url ws://node:8546 --config /backup/config.yml
```

* Test api
```
curl -H "Content-Type: application/json" -XGET localhost:8000/status
```

## Development

* Run go env

```
docker run -it --rm --name dev -p 8000:8000 -v $PWD:/go/src -w /go/src golang:1.14 bash
$ go run cmd/main.go eth
$ go run cmd/main.go hlf
```
