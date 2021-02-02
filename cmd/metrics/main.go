package main

import (
	"errors"
	eth "github.com/IRT-SystemX/bcm-poller/internal/metrics/eth"
	hlf "github.com/IRT-SystemX/bcm-poller/internal/metrics/hlf"
	model "github.com/IRT-SystemX/bcm-poller/poller"
	poller "github.com/IRT-SystemX/bcm-poller/poller/engine"
	utils "github.com/IRT-SystemX/bcm-poller/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
)

const (
	refresh     uint64 = 10
	maxForkSize int    = 10
)

var (
	ethUrl          string = "ws://localhost:8546"
	hlfPath         string = "/tmp/hyperledger-fabric-network/settings/connection-org1.json"
	walletUser      string = "admin"
	orgUser         string = "Admin"
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
	apiUrl          string = "http://localhost:8545"
	metrics         bool   = false
)

func runEth(cmd *cobra.Command, args []string) {
	engine := poller.NewEthEngine(viper.GetString("url"), viper.GetString("syncMode"), viper.GetInt("syncThreadPool"), viper.GetInt("syncThreadSize"))

	log.Printf("Poller is connecting to " + viper.GetString("url"))
	client := interface{}(engine.RawEngine).(*poller.EthEngine).Connect()
	log.Printf("Poller is connected to  " + viper.GetString("url"))

	cache := eth.NewExporterCache(client, utils.NewFetcher(viper.GetString("api")), viper.GetString("config"), viper.GetString("backupPath"), viper.GetBool("restore"), int64(viper.GetInt("backup")))
	fork := eth.NewForkWatcher(interface{}(cache).(model.Connector), maxForkSize)
	processor := eth.NewProcessor(client, fork)

	initEngine(viper.GetString("start"), cache.Stats["block"].Count, viper.GetString("end"), engine, interface{}(cache).(model.Connector), interface{}(processor).(model.Processor))

	run(viper.GetString("port"), viper.GetString("ledgerPath"), engine, interface{}(cache).(model.Connector), map[string]interface{}{"stats": cache.Stats, "tracking": cache.Tracking, "status": engine.Status()})
}

func runHlf(cmd *cobra.Command, args []string) {
	engine := poller.NewHlfEngine(viper.GetString("path"), viper.GetString("walletUser"), viper.GetString("orgUser"), viper.GetString("syncMode"), viper.GetInt("syncThreadPool"), viper.GetInt("syncThreadSize"))

	log.Printf("Poller is connecting")
	interface{}(engine.RawEngine).(*poller.HlfEngine).Connect()
	log.Printf("Poller is connected")

	cache := hlf.NewCache(viper.GetString("config"), viper.GetString("backupPath"), viper.GetBool("restore"), int64(viper.GetInt("backup")))
	processor := hlf.NewProcessor()

	initEngine(viper.GetString("start"), cache.Stats["block"].Count, viper.GetString("end"), engine, interface{}(cache).(model.Connector), interface{}(processor).(model.Processor))

	run(viper.GetString("port"), viper.GetString("ledgerPath"), engine, interface{}(cache).(model.Connector), map[string]interface{}{"stats": cache.Stats, "tracking": cache.Tracking, "status": engine.Status()})
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
	if viper.GetBool("metrics") {
		if interface{}(cache).(*eth.ExporterCache).Fetcher == nil {
			log.Fatal(errors.New("Cannot connect to " + viper.GetString("api") + " for metrics"))
		}
		registry := prometheus.NewPedanticRegistry()
		registry.MustRegister(interface{}(cache).(prometheus.Collector))
		http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: log.New(os.Stderr, log.Prefix(), log.Flags()), ErrorHandling: promhttp.ContinueOnError}))
	}
	server.Start()
}

func main() {
	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("POLLER")
		viper.AutomaticEnv()
	})
	var ethCmd = &cobra.Command{
		Use: "eth",
		Run: runEth,
	}
	ethCmd.Flags().String("url", ethUrl, "Url socket web3")
	ethCmd.Flags().String("api", apiUrl, "Url http web3")
	ethCmd.Flags().Bool("metrics", metrics, "Expose open metrics")
	viper.BindPFlag("url", ethCmd.Flags().Lookup("url"))
	viper.BindPFlag("api", ethCmd.Flags().Lookup("api"))
	viper.BindPFlag("metrics", ethCmd.Flags().Lookup("metrics"))
	var hlfCmd = &cobra.Command{
		Use: "hlf",
		Run: runHlf,
	}
	hlfCmd.Flags().String("path", hlfPath, "Path hlf files")
	hlfCmd.Flags().String("walletUser", walletUser, "Wallet user hlf")
	hlfCmd.Flags().String("orgUser", orgUser, "Org user hlf")
	viper.BindPFlag("path", hlfCmd.Flags().Lookup("path"))
	viper.BindPFlag("walletUser", hlfCmd.Flags().Lookup("walletUser"))
	viper.BindPFlag("orgUser", hlfCmd.Flags().Lookup("orgUser"))
	var rootCmd = &cobra.Command{
		Short: "Event poller with RESTful API",
	}
	rootCmd.AddCommand(ethCmd)
	rootCmd.AddCommand(hlfCmd)
	rootCmd.PersistentFlags().Int("port", port, "Port to run server on")
	rootCmd.PersistentFlags().String("config", config, "Config file")
	rootCmd.PersistentFlags().String("backupPath", backupPath, "Backup file path")
	rootCmd.PersistentFlags().Int("backup", backupFrequency, "Backup frequency in number of blocks")
	rootCmd.PersistentFlags().String("syncMode", syncMode, "Sync mode (fast or normal)")
	rootCmd.PersistentFlags().Int("syncThreadPool", syncThreadPool, "Nb of thread to sync")
	rootCmd.PersistentFlags().Int("syncThreadSize", syncThreadSize, "Nb of blocks per thread per sync round")
	rootCmd.PersistentFlags().String("start", start, "Sync start block")
	rootCmd.PersistentFlags().String("end", end, "Sync end block")
	rootCmd.PersistentFlags().Bool("restore", restore, "Restore backup")
	rootCmd.PersistentFlags().String("ledgerPath", ledgerPath, "Monitored ledger path on disk")
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("backupPath", rootCmd.PersistentFlags().Lookup("backupPath"))
	viper.BindPFlag("backup", rootCmd.PersistentFlags().Lookup("backup"))
	viper.BindPFlag("syncMode", rootCmd.PersistentFlags().Lookup("syncMode"))
	viper.BindPFlag("syncThreadPool", rootCmd.PersistentFlags().Lookup("syncThreadPool"))
	viper.BindPFlag("syncThreadSize", rootCmd.PersistentFlags().Lookup("syncThreadSize"))
	viper.BindPFlag("restore", rootCmd.PersistentFlags().Lookup("restore"))
	viper.BindPFlag("start", rootCmd.PersistentFlags().Lookup("start"))
	viper.BindPFlag("end", rootCmd.PersistentFlags().Lookup("end"))
	viper.BindPFlag("ledgerPath", rootCmd.PersistentFlags().Lookup("ledgerPath"))
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
