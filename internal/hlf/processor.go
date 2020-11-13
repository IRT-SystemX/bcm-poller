package hlf

import (
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
    "github.com/hyperledger/fabric-config/protolator"
    "github.com/golang/protobuf/proto"
	"github.com/tidwall/gjson"
	b64 "encoding/base64"
	"bytes"
	"strconv"
	"log"
	"math/big"
)

type BlockCacheEvent struct {
	number       *big.Int
	hash         string
	parentHash   string
	Interval     uint64
	timestamp    uint64
	Transactions []*TxEvent
	ingest.BlockEvent
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
	id			string
	creator     string
	timestamp   string
	Chaincode   string
	Method      string
}

type Processor struct {
}

func NewProcessor() *Processor {
	processor := &Processor{}
	return processor
}

func (processor *Processor) NewBlockEvent(number *big.Int, parentHash string, hash string) ingest.BlockEvent {
	blockEvent := &BlockCacheEvent{
		number:     number,
		parentHash: parentHash,
		hash:       hash,
	}
	return interface{}(blockEvent).(ingest.BlockEvent)
}

func (processor *Processor) Process(obj interface{}, event ingest.BlockEvent, listening bool) {
	var buf bytes.Buffer
	err := protolator.DeepMarshalJSON(&buf, obj.(proto.Message))
	if err != nil {
        log.Fatalln("DeepMarshalJSON error:", err)
    }
	blockEvent := interface{}(event).(*BlockCacheEvent)
	txs := gjson.Get(buf.String(), "data.data").Array()
	blockEvent.Transactions = make([]*TxEvent, len(txs))
	for i, _ := range txs {
		prefix := "data.data."+strconv.Itoa(i)
		id := gjson.Get(buf.String(), prefix+".payload.header.channel_header.tx_id").String()
		creator := gjson.Get(buf.String(), prefix+".payload.header.channel_header.creator").String()
		timestamp := gjson.Get(buf.String(), prefix+".payload.header.signature_header.timestamp").String()
		name := gjson.Get(buf.String(), prefix+".payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.chaincode_id.name").String()
		args := gjson.Get(buf.String(), prefix+".payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.input.args").Array()
		txEvent := &TxEvent{id: id, creator: creator, timestamp: timestamp, Chaincode: name, }
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
	}
}
