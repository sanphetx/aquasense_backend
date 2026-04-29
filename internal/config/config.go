package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	ServerPort       string
	GinMode          string
	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	JWTSecret        string
	JWTExpireHours   int
	CORSAllowOrigins string
}

// Load reads .env file and environment variables, returning a Config.
func Load() *Config {
	// Load .env if present (ignored in production where env vars are injected)
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env file not found, using system env vars")
	}

	expireHours, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))

	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		GinMode:          getEnv("GIN_MODE", "debug"),
		DBHost:           getEnv("DB_HOST", "localhost"),
		DBPort:           getEnv("DB_PORT", "3306"),
		DBName:           getEnv("DB_NAME", "aquasense"),
		DBUser:           getEnv("DB_USER", "root"),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		JWTSecret:        getEnv("JWT_SECRET", "change_me_in_production"),
		JWTExpireHours:   expireHours,
		CORSAllowOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

// DSN returns the MySQL data source name.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
