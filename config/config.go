package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string        `json:"port"`
	PollInterval    time.Duration `json:"pollInterval"`
	RequestTimeout  time.Duration `json:"requestTimeout"`
	ServerTimeout   time.Duration `json:"serverTimeout"`
	MaxNewsItems    int           `json:"maxNewsItems"`
	EnableSentiment bool          `json:"enableSentiment"`
	LogLevel        string        `json:"logLevel"`
	DatabasePath    string        `json:"databasePath"`
	CacheTimeout    time.Duration `json:"cacheTimeout"`
	MaxConcurrent   int           `json:"maxConcurrent"`
	RateLimitRPM    int           `json:"rateLimitRPM"`
}

func LoadConfig() *Config {
	config := &Config{
		Port:            getEnvWithDefault("PORT", ":8080"),
		PollInterval:    getDurationEnvWithDefault("POLL_INTERVAL", 5*time.Minute),
		RequestTimeout:  getDurationEnvWithDefault("REQUEST_TIMEOUT", 30*time.Second),
		ServerTimeout:   getDurationEnvWithDefault("SERVER_TIMEOUT", 30*time.Second),
		MaxNewsItems:    getIntEnvWithDefault("MAX_NEWS_ITEMS", 1000),
		EnableSentiment: getBoolEnvWithDefault("ENABLE_SENTIMENT", true),
		LogLevel:        getEnvWithDefault("LOG_LEVEL", "info"),
		DatabasePath:    getEnvWithDefault("DATABASE_PATH", "./data/news.db"),
		CacheTimeout:    getDurationEnvWithDefault("CACHE_TIMEOUT", 10*time.Minute),
		MaxConcurrent:   getIntEnvWithDefault("MAX_CONCURRENT", 10),
		RateLimitRPM:    getIntEnvWithDefault("RATE_LIMIT_RPM", 60),
	}

	// Try to load from config file if exists
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		if data, err := os.ReadFile(configFile); err == nil {
			json.Unmarshal(data, config)
		}
	}

	return config
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnvWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnvWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func (c *Config) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
