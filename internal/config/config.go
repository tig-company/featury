package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string `mapstructure:"port"`
	Database    string `mapstructure:"database"`
	DatabaseURL string `mapstructure:"database_url"`
	LogLevel    string `mapstructure:"log_level"`
	Environment string `mapstructure:"environment"`
}

func Load() *Config {
	viper.SetDefault("port", "8080")
	viper.SetDefault("database", "featury.db")
	viper.SetDefault("database_url", "postgres://user:password@localhost/featury?sslmode=disable")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("environment", "development")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	return &cfg
}