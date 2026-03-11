package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database       DatabaseConfig
	Worker         WorkerConfig
	LogLevel       string
	ServiceName    string
	WorkerInterval time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type WorkerConfig struct {
	BatchSize     int
	MaxRetries    int
	RetryDelay    time.Duration
	PollInterval  time.Duration
	ConsumerNames []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "eventbus"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Worker: WorkerConfig{
			BatchSize:    getEnvAsInt("WORKER_BATCH_SIZE", 10),
			MaxRetries:   getEnvAsInt("WORKER_MAX_RETRIES", 3),
			RetryDelay:   getEnvAsDuration("WORKER_RETRY_DELAY", 5*time.Second),
			PollInterval: getEnvAsDuration("WORKER_POLL_INTERVAL", 5*time.Second),
		},
		LogLevel:       getEnv("LOG_LEVEL", "INFO"),
		ServiceName:    getEnv("SERVICE_NAME", "eventbus-worker"),
		WorkerInterval: getEnvAsDuration("WORKER_INTERVAL", 5*time.Second),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	return nil
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
