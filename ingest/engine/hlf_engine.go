package engine

import (
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
    "github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
    //"github.com/hyperledger/fabric-config/protolator"
    //"github.com/hyperledger/fabric/protoutil"
    "reflect"
	"log"
	"encoding/hex"
	"math/big"
	"time"
)

type HlfEngine struct {
	*ingest.Engine
	path           string
	network        *gateway.Network
	client         *ledger.Client
}

func NewHlfEngine(path string, syncMode string, syncThreadPool int, syncThreadSize int) *ingest.Engine {
	engine := &HlfEngine{
		Engine: ingest.NewEngine(syncMode, syncThreadPool, syncThreadSize),
		path: path,
	}
	engine.Engine.RawEngine = engine
	return engine.Engine
}

var (
    WALLET = "/tmp/hyperledger-fabric-network/wallets/organizations/org1.bnc.com"
    PATH = "/tmp/hyperledger-fabric-network/settings/connection-org1.json"
    USER = "admin"
    CHANNEL = "mychannel"
    contextUser fabsdk.ContextOption = fabsdk.WithUser("Admin")
	contextOrg fabsdk.ContextOption = fabsdk.WithOrg("org1")
)

func (engine *HlfEngine) Connect() {
    wallet, err := gateway.NewFileSystemWallet(WALLET)
	if err != nil {
		log.Fatal(err)
	}
	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(PATH)),
		gateway.WithIdentity(wallet, USER),
	)
	if err != nil {
		log.Fatal(err)
	}
	//defer gw.Close()
	log.Printf("gateway ok")
	for {
		network, err := gw.GetNetwork(CHANNEL)
		if err != nil {
			time.Sleep(retry * time.Second)
		} else {
		    log.Printf("network ok")
			engine.network = network
			break
		}
	}
	sdk, err := fabsdk.New(config.FromFile(PATH))
	if err != nil {
		log.Fatal(err)
	}
	//defer sdk.Close()
	context := sdk.ChannelContext(CHANNEL, contextUser, contextOrg, )
	client, err := ledger.New(context)
	if err != nil {
		log.Fatal(err)
	}
	engine.client = client
}

func (engine *HlfEngine) Latest() (*big.Int, error) {
	info, err := engine.client.QueryInfo()
	if err != nil {
		return nil, err
	} else {
    	return big.NewInt(int64(info.BCI.Height)-1), nil
	}
}

func (engine *HlfEngine) Process(number *big.Int, listening bool) ingest.BlockEvent {
	log.Printf("Processing block #%s", number.String())
	block, err := engine.client.QueryBlock(number.Uint64())
 	if err != nil {
		log.Println("Error block: ", err)
		return nil
	}
	//log.Printf("Process block #%s (%s) %s", block.Number().String(), time.Unix(int64(block.Time()), 0).Format("2006.01.02 15:04:05"), head.Hash.Hex())
	log.Printf("Block %d", block.Header.Number)
	
	/*
	env, err := protoutil.ExtractEnvelope(block, 0)
	if err != nil {
		log.Println("Error block envelope: ", err)
		return nil
	}
	payload, err := protoutil.UnmarshalPayload(env.Payload)
 	if err != nil {
		log.Println("Error block payload: ", err)
		return nil
	}
	log.Printf(payload)
	
	log.Printf("%d\n", len(block.Data.Data))
	var buf bytes.Buffer
	err = protolator.DeepMarshalJSON(&buf, block)
	if err != nil {
        log.Fatalln("DeepMarshalJSON err:", err)
    }

    	"encoding/json"
		"github.com/tidwall/gjson"
		b64 "encoding/base64"

	var jsonUnmarshalled map[string]interface{}
	json.Unmarshal(buf.Bytes(), &jsonUnmarshalled)
	fmt.Printf("unmarshalled: %s\n\n", jsonUnmarshalled["data"])
	data := jsonUnmarshalled["data"].(map[string]interface{})
	fmt.Printf("unmarshalled data: %s\n\n", data["data"])

	args := gjson.Get(
		buf.String(),
		"data.data.0.payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.input.args",
	).Array()
	for _, element := range args {
		value, _ := b64.StdEncoding.DecodeString(element.String())
		fmt.Printf(
			"value: %s\n",
			value,
		)
	}
	*/
	
	event := engine.Processor.NewBlockEvent(big.NewInt(int64(block.Header.Number)), hex.EncodeToString(block.Header.PreviousHash), hex.EncodeToString(block.Header.DataHash))
	if engine.Processor != nil && !reflect.ValueOf(engine.Processor).IsNil() {
		engine.Processor.Process(block, event, listening)
	}
    return event
}

func (engine *HlfEngine) Listen() {
	/*
	
	headers := make(chan *types.Header)
	sub, err := engine.client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case err := <-sub.Err():
			log.Println("Error: ", err)
		case header := <-headers:
			//log.Printf("New block #%s", header.Number.String())
			if header != nil {
				engine.ListenProcess(header.Number)
			}
		}
	}
	
	
	reg, notifier, err := engine.network.RegisterFilteredBlockEvent()
 	if err != nil {
		log.Fatalln("Failed to register filtered block event: %s", err)
	}
	defer network.Unregister(reg)
	
	go func() {
		var bEvent *fab.FilteredBlockEvent
		for true {
			select {
			case bEvent = <-notifier:
				log.Printf("Received block event: %#v\n", bEvent)
				log.Printf("block number: %d\n", bEvent.FilteredBlock.Number)
				log.Printf("number of transactions: %d\n", len(bEvent.FilteredBlock.FilteredTransactions))
				for _, value := range bEvent.FilteredBlock.FilteredTransactions {
					log.Printf("%s\n", value.String())

					dump_transaction(client, fab.TransactionID(value.Txid))
					
					transaction, _ := client.QueryTransaction(txid)

					var buf bytes.Buffer
					err := protolator.DeepMarshalJSON(&buf, transaction)
					if err != nil {
				        log.Fatalln("DeepMarshalJSON err:", err)
				    }
					// fmt.Printf("json: %s\n", buf.String())
					args := gjson.Get(
						buf.String(),
						"transactionEnvelope.payload.data.actions.0.payload.chaincode_proposal_payload.input.chaincode_spec.input.args",
					).Array()
				
					for _, element := range args {
						value, _ := b64.StdEncoding.DecodeString(element.String())
						fmt.Printf(
							"value: %s\n",
							value,
						)
					}

				}

			case <-time.After(time.Second * 20):
				log.Printf("Did NOT receive block event\n")
			}
		}
	}()
	
	*/
}
