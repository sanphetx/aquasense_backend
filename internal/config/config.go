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
	ServerPort           string
	GinMode              string
	DBHost               string
	DBPort               string
	DBName               string
	DBUser               string
	DBPassword           string
	JWTSecret            string
	JWTExpireHours       int
	CORSAllowOrigins     string
	AdminDefaultPassword string // ADMIN_DEFAULT_PASSWORD — seed admin password
	// OAuth providers
	GoogleClientID string // GOOGLE_CLIENT_ID from Google Cloud Console
	AppleClientID  string // APPLE_CLIENT_ID (bundle ID / service ID)
}

// Load reads .env file and environment variables, returning a Config.
func Load() *Config {
	// Load .env if present (ignored in production where env vars are injected)
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env file not found, using system env vars")
	}

	expireHours, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	ginMode := getEnv("GIN_MODE", "debug")

	// [Security #2] JWT_SECRET must be set in production
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if ginMode == "release" {
			log.Fatal("[FATAL] JWT_SECRET env var must be set in production (GIN_MODE=release)")
		}
		jwtSecret = "insecure_dev_secret_do_not_use_in_production"
		log.Println("[WARN] JWT_SECRET not set — using insecure default (development only)")
	}

	return &Config{
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		GinMode:              ginMode,
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               getEnv("DB_PORT", "3307"),
		DBName:               getEnv("DB_NAME", "aquasense"),
		DBUser:               getEnv("DB_USER", "root"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		JWTSecret:            jwtSecret,
		JWTExpireHours:       expireHours,
		CORSAllowOrigins:     getEnv("CORS_ALLOWED_ORIGINS", "*"),
		AdminDefaultPassword: getEnv("ADMIN_DEFAULT_PASSWORD", ""),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		AppleClientID:        getEnv("APPLE_CLIENT_ID", ""),
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
