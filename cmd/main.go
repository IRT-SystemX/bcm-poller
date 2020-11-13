package main

import (
	model "github.com/IRT-SystemX/bcm-poller/ingest"
	ingest "github.com/IRT-SystemX/bcm-poller/ingest/engine"
	eth "github.com/IRT-SystemX/bcm-poller/internal/eth"
	hlf "github.com/IRT-SystemX/bcm-poller/internal/hlf"
	utils "github.com/IRT-SystemX/bcm-poller/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
)

const (
	refresh         uint64 = 10
	maxForkSize     int    = 10
)

var (
	ethUrl          string = "ws://localhost:8546"
	hlfPath         string = "/tmp/hyperledger-fabric-network/settings/connection-org1.json"
	user            string = "admin"
	port            int    = 8000
	config          string = "config.yml"
	restore         bool   = false
	backupPath      string = "backup.json"
	backupFrequency int    = 0
	start           string = "0"
	end             string = "-1"
	syncMode        string = "normal"
	syncThreadPool  int    = 4
	syncThreadSize  int    = 25
	ledgerPath      string = "/chain"
)

func runEth(cmd *cobra.Command, args []string) {
	engine := ingest.NewEthEngine(viper.GetString("url"), viper.GetString("syncMode"), viper.GetInt("syncThreadPool"), viper.GetInt("syncThreadSize"))

	log.Printf("Poller is connecting to " + viper.GetString("url"))
	client := interface{}(engine.RawEngine).(*ingest.EthEngine).Connect()
	log.Printf("Poller is connected to  " + viper.GetString("url"))
	
	cache := eth.NewCache(client, viper.GetString("config"), viper.GetString("backupPath"), viper.GetBool("restore"), int64(viper.GetInt("backup")))
	fork := eth.NewForkWatcher(interface{}(cache).(model.Connector), maxForkSize)
	processor := eth.NewProcessor(client, fork)
	
	initEngine(viper.GetString("start"), cache.Stats["block"].Count, viper.GetString("end"), engine, interface{}(cache).(model.Connector), interface{}(processor).(model.Processor))
	bind := map[string]interface{}{
		"stats":    cache.Stats,
		"tracking": cache.Tracking,
		"status":   engine.Status(),
	}
	run(viper.GetString("port"), viper.GetString("ledgerPath"), engine, interface{}(cache).(model.Connector), bind)
}

func runHlf(cmd *cobra.Command, args []string) {
	engine := ingest.NewHlfEngine(viper.GetString("path"), viper.GetString("user"), viper.GetString("syncMode"), viper.GetInt("syncThreadPool"), viper.GetInt("syncThreadSize"))

	log.Printf("Poller is connecting")
	interface{}(engine.RawEngine).(*ingest.HlfEngine).Connect()
	log.Printf("Poller is connected")

	cache := hlf.NewCache(viper.GetString("config"), viper.GetString("backupPath"), viper.GetBool("restore"), int64(viper.GetInt("backup")))
	processor := hlf.NewProcessor()

	initEngine(viper.GetString("start"), cache.Stats["block"].Count, viper.GetString("end"), engine, interface{}(cache).(model.Connector), interface{}(processor).(model.Processor))
	bind := map[string]interface{}{
		"stats":    cache.Stats,
		"tracking": cache.Tracking,
		"status":   engine.Status(),
	}
	run(viper.GetString("port"), viper.GetString("ledgerPath"), engine, interface{}(cache).(model.Connector), bind)
}

func initEngine(start string, defaultStart string, end string, engine *model.Engine, cache model.Connector, processor model.Processor) {
	if start == "-1" {
		if viper.GetBool("restore") {
			engine.SetStart(defaultStart, true)
		} else {
			last, err := engine.Latest()
			if err != nil {
				log.Fatal(err)
			}
			engine.SetStart(last.String(), false)
		}
	} else {
		engine.SetStart(start, false)
	}
	engine.SetEnd(end)
	engine.SetConnector(cache)
	engine.SetProcessor(processor)
}

func run(port string, ledgerPath string, engine *model.Engine, cache model.Connector, bind map[string]interface{}) {
	disk := utils.NewDiskUsage(ledgerPath, refresh)
	bind["disk"] = disk
	go func() {
		engine.Init()
		cache.SetReady()
		disk.Start()
		engine.Listen()
	}()
	server := utils.NewServer(port)
	server.Bind(bind)
	server.Start()
}

func fillCmd(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().Int("port", port, "Port to run server on")
	cmd.Flags().String("config", config, "Config file")
	cmd.Flags().String("backupPath", backupPath, "Backup file path")
	cmd.Flags().Int("backup", backupFrequency, "Backup frequency in number of blocks")
	cmd.Flags().String("syncMode", syncMode, "Sync mode (fast or normal)")
	cmd.Flags().Int("syncThreadPool", syncThreadPool, "Nb of thread to sync")
	cmd.Flags().Int("syncThreadSize", syncThreadSize, "Nb of blocks per thread per sync round")
	cmd.Flags().String("start", start, "Sync start block")
	cmd.Flags().String("end", end, "Sync end block")
	cmd.Flags().Bool("restore", restore, "Restore backup")
	cmd.Flags().String("ledgerPath", ledgerPath, "Monitored ledger path on disk")
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("config", cmd.Flags().Lookup("config"))
	viper.BindPFlag("backupPath", cmd.Flags().Lookup("backupPath"))
	viper.BindPFlag("backup", cmd.Flags().Lookup("backup"))
	viper.BindPFlag("syncMode", cmd.Flags().Lookup("syncMode"))
	viper.BindPFlag("syncThreadPool", cmd.Flags().Lookup("syncThreadPool"))
	viper.BindPFlag("syncThreadSize", cmd.Flags().Lookup("syncThreadSize"))
	viper.BindPFlag("restore", cmd.Flags().Lookup("restore"))
	viper.BindPFlag("start", cmd.Flags().Lookup("start"))
	viper.BindPFlag("end", cmd.Flags().Lookup("end"))
	viper.BindPFlag("ledgerPath", cmd.Flags().Lookup("ledgerPath"))
	return cmd
}

func main() {
	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("ETH")
		viper.AutomaticEnv()
	})
	var ethCmd = &cobra.Command{
		Use:   "eth",
		Run:   runEth,
	}
	ethCmd.Flags().String("url", ethUrl, "Url socket web3")
	viper.BindPFlag("url", ethCmd.Flags().Lookup("url"))
	var hlfCmd = &cobra.Command{
		Use:   "hlf",
		Run:   runHlf,
	}
	hlfCmd.Flags().String("path", hlfPath, "Path hlf files")
	hlfCmd.Flags().String("user", user, "User hlf")
	viper.BindPFlag("path", hlfCmd.Flags().Lookup("path"))
	viper.BindPFlag("user", hlfCmd.Flags().Lookup("user"))
	var rootCmd = &cobra.Command{
		Short: "Event poller with RESTful API",
	}
	rootCmd.AddCommand(fillCmd(ethCmd))
	rootCmd.AddCommand(fillCmd(hlfCmd))
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
