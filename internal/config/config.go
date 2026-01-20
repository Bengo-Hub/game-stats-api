package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Env         string
	Port        string
	DatabaseURL string
	RedisURL    string
	LogLevel    string
	JWTSecret   string
}

func Load() *Config {
	viper.SetDefault("ENV", "development")
	viper.SetDefault("PORT", "4000")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6380/0")
	viper.SetDefault("JWT_SECRET", "dev-secret-key")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a .env file exists, read it
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	return &Config{
		Env:         viper.GetString("ENV"),
		Port:        viper.GetString("PORT"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		RedisURL:    viper.GetString("REDIS_URL"),
		LogLevel:    viper.GetString("LOG_LEVEL"),
		JWTSecret:   viper.GetString("JWT_SECRET"),
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
