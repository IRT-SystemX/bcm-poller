package engine

import (
	ingest "github.com/IRT-SystemX/bcm-poller/ingest"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
    "github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
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
	block, err := engine.client.QueryBlock(number.Uint64())
 	if err != nil {
		log.Println("Error block: ", err)
		return nil
	}
	log.Printf("Process block %d", block.Header.Number)
	event := engine.Processor.NewBlockEvent(big.NewInt(int64(block.Header.Number)), hex.EncodeToString(block.Header.PreviousHash), hex.EncodeToString(block.Header.DataHash))
	if engine.Processor != nil && !reflect.ValueOf(engine.Processor).IsNil() {
		engine.Processor.Process(block, event, listening)
	}
    return event
}

func (engine *HlfEngine) Listen() {
	reg, notifier, err := engine.network.RegisterFilteredBlockEvent()
 	if err != nil {
		log.Fatalln("Failed to register filtered block event: %s", err)
	}
	defer engine.network.Unregister(reg)
	for {
		select {
		case bEvent := <-notifier:
			if bEvent != nil {
				//log.Printf("New block #%d", bEvent.FilteredBlock.Number)
				engine.ListenProcess(big.NewInt(int64(bEvent.FilteredBlock.Number)))
			}
		}
	}
}
