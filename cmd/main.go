package main

import (
	"log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
    ingest "eth-poller/poller/ingest"
    probe "eth-poller/poller/probe"
    conn "eth-poller/poller/conn"
    utils "eth-poller/poller/utils"
)

var (
	refresh uint64 = 10
)

func run(web3 string, path string, config string, port string) {
	engine := ingest.NewEngine(web3)
	disk, err := probe.NewDiskUsage(path, refresh)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Poller is connecting to "+web3)
	client := engine.Connect()
	log.Printf("Poller is connected to  "+web3)

	cache := conn.NewCache(client, config)
	engine.SetConnector(interface{}(cache).(ingest.Connector))

	engine.Start()
    disk.Start()
	
	server :=  utils.NewServer(port)
	server.Bind(map[string]interface{}{
        "disk": disk,
        "stats": cache.Stats,
        "tracking": cache.Tracking,
        "status": engine.Status(),
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
		Run: func(cmd *cobra.Command, args []string) {
			run(viper.GetString("url"), viper.GetString("path"), viper.GetString("config"), viper.GetString("server_port"))
		},
	}
	rootCmd.Flags().Int("server_port", 8000, "Port to run server on")
	rootCmd.Flags().String("url", "localhost:8546", "Address web3")
	rootCmd.Flags().String("config", "config.yml", "Config file")
	rootCmd.Flags().String("path", "/home", "Path disk")
	viper.BindPFlag("server_port", rootCmd.Flags().Lookup("server_port"))
	viper.BindPFlag("url", rootCmd.Flags().Lookup("url"))
	viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))
	viper.BindPFlag("path", rootCmd.Flags().Lookup("path"))
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
