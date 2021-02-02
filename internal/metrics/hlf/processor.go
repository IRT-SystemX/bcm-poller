package hlf

import (
	"bytes"
	b64 "encoding/base64"
	poller "github.com/IRT-SystemX/bcm-poller/poller"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-config/protolator"
	"github.com/tidwall/gjson"
	"log"
	"math/big"
	"strconv"
	"time"
)

type BlockCacheEvent struct {
	number       *big.Int
	hash         string
	parentHash   string
	Interval     uint64
	timestamp    uint64
	Transactions []*TxEvent
	poller.BlockEvent
}

func (blockEvent *BlockCacheEvent) Number() *big.Int {
	return blockEvent.number
}

func (blockEvent *BlockCacheEvent) Hash() string {
	return blockEvent.hash
}

func (blockEvent *BlockCacheEvent) ParentHash() string {
	return blockEvent.parentHash
}

func (blockEvent *BlockCacheEvent) Timestamp() uint64 {
	return blockEvent.timestamp
}

type TxEvent struct {
	id        string
	creator   string
	Timestamp uint64
	Chaincode string
	Method    string
}

type Processor struct {
}

func NewProcessor() *Processor {
	processor := &Processor{}
	return processor
}

func (processor *Processor) NewBlockEvent(number *big.Int, parentHash string, hash string) poller.BlockEvent {
	blockEvent := &BlockCacheEvent{
		number:     number,
		parentHash: parentHash,
		hash:       hash,
	}
	return interface{}(blockEvent).(poller.BlockEvent)
}

func (processor *Processor) Process(obj interface{}, event poller.BlockEvent, listening bool) {
	var buf bytes.Buffer
	err := protolator.DeepMarshalJSON(&buf, obj.(proto.Message))
	if err != nil {
		log.Fatalln("DeepMarshalJSON error:", err)
	}
	blockEvent := interface{}(event).(*BlockCacheEvent)
	txs := gjson.Get(buf.String(), "data.data").Array()
	blockEvent.Transactions = make([]*TxEvent, len(txs))
	blockEvent.timestamp = 0
	for i, _ := range txs {
		prefix := "data.data." + strconv.Itoa(i)
		id := gjson.Get(buf.String(), prefix+".payload.header.channel_header.tx_id").String()
		creator := gjson.Get(buf.String(), prefix+".payload.header.channel_header.creator").String()
		timestampStr := gjson.Get(buf.String(), prefix+".payload.header.channel_header.timestamp").String()
		name := gjson.Get(buf.String(), prefix+".payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.chaincode_id.name").String()
		args := gjson.Get(buf.String(), prefix+".payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.input.args").Array()
		timestamp, err := time.Parse("2006-01-02T15:04:05Z", timestampStr)
		if err != nil {
			log.Fatal(err)
		}
		txEvent := &TxEvent{id: id, creator: creator, Timestamp: uint64(timestamp.Unix()), Chaincode: name}
		for _, val := range args {
			value, err := b64.StdEncoding.DecodeString(val.String())
			if err != nil {
				log.Fatal(err)
			}
			//log.Printf("value: %s\n", value)
			txEvent.Method = string(value)
			break
		}
		log.Printf("Process tx %s > %s_%s", txEvent.id, txEvent.Chaincode, txEvent.Method)
		blockEvent.Transactions[i] = txEvent
		if blockEvent.timestamp == 0 {
			blockEvent.timestamp = txEvent.Timestamp
		}
	}
}
