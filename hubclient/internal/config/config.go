package config

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
)

const (
	DefaultPort = "9090"
)

type Config struct {
	Port    string
	HubAddr string
}

func LoadConfig(logger *zap.Logger) *Config {
	var cfg Config

	rootCmd := &cobra.Command{
		Use:   "hubclient",
		Short: "hubClient is a simple web-server that serves an HTML page to communicate with the hubServer",
		Run: func(cmd *cobra.Command, args []string) {
			if cfg.HubAddr == "" {
				logger.Error("hub-addr is required")
				_ = cmd.Help()
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&cfg.Port, "port", DefaultPort, "Port for serving the HTML page")
	rootCmd.Flags().StringVar(&cfg.HubAddr, "hub-addr", "", "Address of the HubServer")
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Error parsing arguments", zap.Error(err))
		os.Exit(1)
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	if hubAddr := os.Getenv("HUBADDR"); hubAddr != "" {
		cfg.HubAddr = hubAddr
	}

	return &cfg
}
