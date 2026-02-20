package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

	// Swagger
	SwaggerHost      string
}

func Load() *Config {
	viper.SetDefault("ENV", "development")
	viper.SetDefault("PORT", "4000")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6380/0")
	viper.SetDefault("JWT_SECRET", "dev-secret-key")
	viper.SetDefault("METABASE_BASE_URL", "https://analytics.ultimatestats.co.ke")
	viper.SetDefault("METABASE_USERNAME", "admin")
	viper.SetDefault("METABASE_PASSWORD", "")
	viper.SetDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	viper.SetDefault("OLLAMA_MODEL", "duckdb-nsql:7b")
	viper.SetDefault("RUN_MIGRATION", "true")
	viper.SetDefault("FIXTURES_DIR", "./scripts/fixtures")
	viper.SetDefault("SWAGGER_HOST", "localhost:4000")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a .env file exists, read it
	viper.SetConfigFile(".env")
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No .env file found, using environment variables")
	}

	return &Config{
		Env:              viper.GetString("ENV"),
		Port:             viper.GetString("PORT"),
		DatabaseURL:      viper.GetString("DATABASE_URL"),
		RedisURL:         viper.GetString("REDIS_URL"),
		LogLevel:         viper.GetString("LOG_LEVEL"),
		JWTSecret:        viper.GetString("JWT_SECRET"),
		MetabaseBaseURL:  viper.GetString("METABASE_BASE_URL"),
		MetabaseUsername: viper.GetString("METABASE_USERNAME"),
		MetabasePassword: viper.GetString("METABASE_PASSWORD"),
		OllamaBaseURL:    viper.GetString("OLLAMA_BASE_URL"),
		OllamaModel:      viper.GetString("OLLAMA_MODEL"),
        
		RunMigration:     viper.GetBool("RUN_MIGRATION"),
		FixturesDir:      viper.GetString("FIXTURES_DIR"),
		SwaggerHost:      viper.GetString("SWAGGER_HOST"),
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
