package config

import (
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	DefaultPort              = "8080"
	DefaultPubSubHostName    = "redis:6379"
	DefaultPubSubChannelName = "hub-messages-pub-sub-channel"
)

type Config struct {
	Port              string
	PubSubHostName    string
	PubSubChannelName string
	HubName           string
	BroadcastWorkers  int
	RedisUsername     string
	RedisPassword     string
}

func LoadConfig(logger *zap.Logger) *Config {
	var cfg Config

	rootCmd := &cobra.Command{
		Use:   "hubserver",
		Short: "HubServer is a realtime messaging server",
		Run: func(cmd *cobra.Command, args []string) {
			if cfg.HubName == "" {
				logger.Error("hub-name is required")
				_ = cmd.Help()
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&cfg.Port, "port", DefaultPort, "Port for websocket connection")
	rootCmd.Flags().StringVar(&cfg.PubSubHostName, "pub-sub-host", DefaultPubSubHostName, "Redis server address")
	rootCmd.Flags().StringVar(&cfg.PubSubChannelName, "pub-sub-channel", DefaultPubSubChannelName, "Redis Pub-Sub channel name")
	rootCmd.Flags().StringVar(&cfg.HubName, "hub-name", "", "Name of the hub (required)")
	rootCmd.Flags().IntVar(&cfg.BroadcastWorkers, "broadcast-workers", 2, "Name of the broadcast workers to run in parallel")
	rootCmd.Flags().StringVar(&cfg.RedisUsername, "redis-username", "redis", "Username for Redis")
	rootCmd.Flags().StringVar(&cfg.RedisPassword, "redis-password", "password", "Password for Redis")

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Error parsing arguments", zap.Error(err))
		os.Exit(1)
	}

	// Override with environment variables if present
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}
	if pubSubHost := os.Getenv("PUB_SUB_HOST"); pubSubHost != "" {
		cfg.PubSubHostName = pubSubHost
	}
	if pubSubChannel := os.Getenv("PUB_SUB_CHANNEL"); pubSubChannel != "" {
		cfg.PubSubChannelName = pubSubChannel
	}
	if hubName := os.Getenv("HUB_NAME"); hubName != "" {
		cfg.HubName = hubName
	}

	return &cfg
}
