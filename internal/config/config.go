// Package config loads and validates all application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the SmartBed backend.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	MQTT     MQTTConfig
	JWT      JWTConfig
	Log      LogConfig
	Seed     SeedConfig
}

// AppConfig holds general server settings.
type AppConfig struct {
	Env        string
	Port       string
	Name       string
	Migrations string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host                string
	Port                string
	Name                string
	User                string
	Password            string
	SSLMode             string
	MaxOpenConns        int
	MaxIdleConns        int
	ConnMaxLifetime     time.Duration
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// MQTTConfig holds MQTT broker settings.
type MQTTConfig struct {
	Broker           string
	ClientID         string
	Username         string
	Password         string
	QOS              byte
	ACKTimeoutSec    int
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	PrivateKeyPath          string
	PublicKeyPath           string
	AccessTokenExpiryMins   int
	RefreshTokenExpiryDays  int
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string
	Pretty bool
}

// SeedConfig holds seed data settings.
type SeedConfig struct {
	AdminEmail    string
	AdminPassword string
}

// Load reads .env (if present) and loads config from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists; ignore error in production.
	_ = godotenv.Load()

	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	connMaxLifetimeMins, _ := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME_MINUTES", "5"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	mqttQOS, _ := strconv.Atoi(getEnv("MQTT_QOS", "1"))
	mqttACKTimeout, _ := strconv.Atoi(getEnv("MQTT_ACK_TIMEOUT_SECONDS", "30"))
	accessTokenExpiry, _ := strconv.Atoi(getEnv("JWT_ACCESS_TOKEN_EXPIRY_MINUTES", "60"))
	refreshTokenExpiry, _ := strconv.Atoi(getEnv("JWT_REFRESH_TOKEN_EXPIRY_DAYS", "7"))
	logPretty, _ := strconv.ParseBool(getEnv("LOG_PRETTY", "true"))

	cfg := &Config{
		App: AppConfig{
			Env:        getEnv("APP_ENV", "development"),
			Port:       getEnv("APP_PORT", "8080"),
			Name:       getEnv("APP_NAME", "smartbed"),
			Migrations: getEnv("MIGRATIONS_PATH", "file://migrations"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "smartbed"),
			User:            getEnv("DB_USER", "smartbed"),
			Password:        getEnv("DB_PASSWORD", "smartbed_secret"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: time.Duration(connMaxLifetimeMins) * time.Minute,
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		MQTT: MQTTConfig{
			Broker:        getEnv("MQTT_BROKER", "tcp://localhost:1883"),
			ClientID:      getEnv("MQTT_CLIENT_ID", "smartbed-server"),
			Username:      getEnv("MQTT_USERNAME", ""),
			Password:      getEnv("MQTT_PASSWORD", ""),
			QOS:           byte(mqttQOS),
			ACKTimeoutSec: mqttACKTimeout,
		},
		JWT: JWTConfig{
			PrivateKeyPath:         getEnv("JWT_PRIVATE_KEY_PATH", "./keys/private.pem"),
			PublicKeyPath:          getEnv("JWT_PUBLIC_KEY_PATH", "./keys/public.pem"),
			AccessTokenExpiryMins:  accessTokenExpiry,
			RefreshTokenExpiryDays: refreshTokenExpiry,
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "debug"),
			Pretty: logPretty,
		},
		Seed: SeedConfig{
			AdminEmail:    getEnv("SEED_ADMIN_EMAIL", "admin@smartbed.local"),
			AdminPassword: getEnv("SEED_ADMIN_PASSWORD", "Admin@2024!"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		c.Database.Host, c.Database.Port, c.Database.Name,
		c.Database.User, c.Database.Password, c.Database.SSLMode,
	)
}

// RedisAddr returns the Redis address string.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

func (c *Config) validate() error {
	required := map[string]string{
		"DB_NAME":       c.Database.Name,
		"DB_USER":       c.Database.User,
		"DB_PASSWORD":   c.Database.Password,
		"JWT_PRIVATE_KEY_PATH": c.JWT.PrivateKeyPath,
		"JWT_PUBLIC_KEY_PATH":  c.JWT.PublicKeyPath,
	}
	for key, val := range required {
		if val == "" {
			return fmt.Errorf("required env var %q is not set", key)
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
