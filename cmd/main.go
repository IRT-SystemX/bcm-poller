package main

import (
	conn "github.com/IRT-SystemX/eth-poller/conn"
	ingest "github.com/IRT-SystemX/eth-poller/ingest"
	probe "github.com/IRT-SystemX/eth-poller/probe"
	utils "github.com/IRT-SystemX/eth-poller/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
)

var (
	refresh         uint64 = 10
	port            int    = 8000
	url             string = "ws://localhost:8546"
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
	maxForkSize     int    = 10
)

func run(cmd *cobra.Command, args []string) {
	engine := ingest.NewEngine(viper.GetString("url"), viper.GetString("syncMode"), viper.GetInt("syncThreadPool"), viper.GetInt("syncThreadSize"), maxForkSize)
	disk := probe.NewDiskUsage(viper.GetString("ledgerPath"), refresh)

	log.Printf("Poller is connecting to " + viper.GetString("url"))
	client := engine.Connect()
	log.Printf("Poller is connected to  " + viper.GetString("url"))

	cache := conn.NewCache(client, viper.GetString("config"), viper.GetString("backupPath"), viper.GetBool("restore"), int64(viper.GetInt("backup")))
	processor := conn.NewProcessor(client)
	if viper.GetString("start") == "-1" {
		if viper.GetBool("restore") {
			engine.SetStart(cache.Stats["block"].Count, true)
		} else {
			last, err := engine.Latest()
			if err != nil {
				log.Fatal(err)
			}
			engine.SetStart(last.String(), false)
		}
	} else {
		engine.SetStart(viper.GetString("start"), false)
	}
	engine.SetEnd(viper.GetString("end"))
	engine.SetConnector(interface{}(cache).(ingest.Connector))
	engine.SetProcessor(interface{}(processor).(ingest.Processor))

	go func() {
		engine.Init()
		cache.SetReady()
		disk.Start()
		engine.Listen()
	}()

	server := utils.NewServer(viper.GetString("port"))
	server.Bind(map[string]interface{}{
		"disk":     disk,
		"stats":    cache.Stats,
		"tracking": cache.Tracking,
		"status":   engine.Status(),
	})
	server.Start()
}

func main() {
	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("ETH")
		viper.AutomaticEnv()
	})
	var rootCmd = &cobra.Command{
		Use:   "eth-poller",
		Short: "Ethereum event poller with RESTful API",
		Run:   run,
	}
	rootCmd.Flags().Int("port", port, "Port to run server on")
	rootCmd.Flags().String("url", url, "Address web3")
	rootCmd.Flags().String("config", config, "Config file")
	rootCmd.Flags().String("backupPath", backupPath, "Backup file path")
	rootCmd.Flags().Int("backup", backupFrequency, "Backup frequency in number of blocks")
	rootCmd.Flags().String("syncMode", syncMode, "Sync mode (fast or normal)")
	rootCmd.Flags().Int("syncThreadPool", syncThreadPool, "Nb of thread to sync")
	rootCmd.Flags().Int("syncThreadSize", syncThreadSize, "Nb of blocks per thread per sync round")
	rootCmd.Flags().String("start", start, "Sync start block")
	rootCmd.Flags().String("end", end, "Sync end block")
	rootCmd.Flags().Bool("restore", restore, "Restore backup")
	rootCmd.Flags().String("ledgerPath", ledgerPath, "Monitored ledger path on disk")
	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("url", rootCmd.Flags().Lookup("url"))
	viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))
	viper.BindPFlag("backupPath", rootCmd.Flags().Lookup("backupPath"))
	viper.BindPFlag("backup", rootCmd.Flags().Lookup("backup"))
	viper.BindPFlag("syncMode", rootCmd.Flags().Lookup("syncMode"))
	viper.BindPFlag("syncThreadPool", rootCmd.Flags().Lookup("syncThreadPool"))
	viper.BindPFlag("syncThreadSize", rootCmd.Flags().Lookup("syncThreadSize"))
	viper.BindPFlag("restore", rootCmd.Flags().Lookup("restore"))
	viper.BindPFlag("start", rootCmd.Flags().Lookup("start"))
	viper.BindPFlag("end", rootCmd.Flags().Lookup("end"))
	viper.BindPFlag("ledgerPath", rootCmd.Flags().Lookup("ledgerPath"))
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
