package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/chandanpasunoori/dns-sync/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const (
	version = "0.0.2"
)

var verbose bool
var configDoc string
var config pkg.Config

var rootCmd = &cobra.Command{
	Use:     "dns-sync",
	Short:   "Built to ease process of syncing changing node IPs to target dns zone",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			logger.SetLevel(log.DebugLevel)
		}
		configBytes, err := ioutil.ReadFile(configDoc)
		if err != nil {
			logger.WithError(err).Errorf("config file not found at " + configDoc)
			os.Exit(1)
		}
		if err := json.Unmarshal(configBytes, &config); err != nil {
			logger.WithError(err).Errorf("error parsing config")
			os.Exit(1)
		}
		logger.Info(
			"dns-sync " + version + " is ready to sync zones",
		)
		pkg.SyncZones(config)
	},
}

var logger = log.Logger{
	Out: os.Stdout,
	Formatter: &log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	},
	Level: log.InfoLevel,
}

func Execute() {
	if genDoc := os.Getenv("GEN_DOC"); genDoc == "true" {
		err := doc.GenMarkdownTree(rootCmd, "./docs")
		if err != nil {
			log.Errorf("Failed generating docs: %v", err)
		}
	}

	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Errorf("error executing command")
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	rootCmd.PersistentFlags().StringVarP(&configDoc, "config", "c", "app.json", "job configuration file path")
	_ = rootCmd.MarkFlagRequired("config")
}
