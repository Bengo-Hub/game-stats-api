package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() *Config {
	viper.SetDefault("PORT", "8080")
	viper.AutomaticEnv()

	return &Config{
		Port:        viper.GetString("PORT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
}
