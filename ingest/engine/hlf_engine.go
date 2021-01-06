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
	"os"
	"io/ioutil"
	"encoding/json"
	"errors"
)

type HlfEngine struct {
	*ingest.Engine
	path           string
	walletUser	   string
	orgUser        string
	network        *gateway.Network
	client         *ledger.Client
}

func NewHlfEngine(path string, walletUser string, orgUser string, syncMode string, syncThreadPool int, syncThreadSize int) *ingest.Engine {
	engine := &HlfEngine{
		Engine: ingest.NewEngine(syncMode, syncThreadPool, syncThreadSize),
		path: path,
		walletUser: walletUser,
		orgUser: orgUser,
	}
	engine.Engine.RawEngine = engine
	return engine.Engine
}

func loadProfile(pathFile string) (map[string]interface{}, error) {
	_, err := os.Stat(pathFile)
	if err == nil {
		data, err := ioutil.ReadFile(pathFile)
		if err != nil {
			log.Fatal(err)
		}
		raw := make(map[string]interface{})
		err = json.Unmarshal([]byte(data), &raw)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		return raw, nil
	} else {
		return nil, errors.New("Error: connection profile not found ("+pathFile+")")
	}
}

func getWalletPath(raw map[string]interface{}) (string, error) {
	_, ok := raw["client"]
	if !ok {
		return "", errors.New("Error: client not found in profile")
	}
	client := raw["client"].(map[string]interface{})
	_, ok = client["credentialStore"]
	if !ok {
		return "", errors.New("Error: credentialStore not found in profile")
	}
	credentialStore := client["credentialStore"].(map[string]interface{})
	_, ok = credentialStore["path"]
	if !ok {
		return "", errors.New("Error: path not found in profile")
	}
	return credentialStore["path"].(string), nil
}

func getChannelName(raw map[string]interface{}) (string, error) {
	_, ok := raw["channels"]
	if !ok {
		return "", errors.New("Error: channels not found in profile")
	}
	channels := raw["channels"].(map[string]interface{})
	return reflect.ValueOf(channels).MapKeys()[0].String(), nil
}

func getOrgName(raw map[string]interface{}) (string, error) {
	_, ok := raw["client"]
	if !ok {
		return "", errors.New("Error: client not found in profile")
	}
	client := raw["client"].(map[string]interface{})
	_, ok = client["organization"]
	if !ok {
		return "", errors.New("Error: organization not found in profile")
	}
	return client["organization"].(string), nil
}

func (engine *HlfEngine) Connect() {
	profile, err := loadProfile(engine.path)
	if err != nil {
		log.Fatal(err)
	}
	channelName, err := getChannelName(profile)
	if err != nil {
		log.Fatal(err)
	}
	configFile := config.FromFile(engine.path)
	// create client
	sdk, err := fabsdk.New(configFile)
	if err != nil {
		log.Fatal(err)
	}
	orgName, err := getOrgName(profile)
	if err != nil {
		log.Fatal(err)
	}
	contextOrg := fabsdk.WithOrg(orgName)
	contextUser := fabsdk.WithUser(engine.orgUser)
	client, err := ledger.New(sdk.ChannelContext(channelName, contextUser, contextOrg, ))
	if err != nil {
		log.Fatal(err)
	}
	engine.client = client
	log.Printf("ledger ok")
	// create network
	walletPath, err := getWalletPath(profile)
	if err != nil {
		log.Fatal(err)
	}
    wallet, err := gateway.NewFileSystemWallet(walletPath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = wallet.Get(engine.walletUser)
	if err != nil {
		log.Fatal(err)
	}
	gw, err := gateway.Connect(
		gateway.WithConfig(configFile),
		gateway.WithIdentity(wallet, engine.walletUser),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("gateway ok")
	for {
		network, err := gw.GetNetwork(channelName)
		if err != nil {
			time.Sleep(retry * time.Second)
		} else {
		    log.Printf("network ok")
			engine.network = network
			break
		}
	}
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
