package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the worker
type Config struct {
	VideoPath    string
	StreamID     string
	RedisAddr    string
	FrameRate    float64       // frames per second
	FrameTTL     time.Duration // Redis key TTL
	RestartDelay time.Duration // delay before restarting ffmpeg
	FFmpegPath   string        // path to ffmpeg binary
	QueueMode    bool          // if true, use Redis Lists (RPUSH) instead of SET
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	cfg := &Config{
		VideoPath:    getEnv("VIDEO_PATH", ""),
		StreamID:     getEnv("STREAM_ID", "default"),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		FFmpegPath:   getEnv("FFMPEG_PATH", "ffmpeg"),
		RestartDelay: 5 * time.Second,
	}

	// Parse frame rate (default: 1 FPS)
	frameRate, err := strconv.ParseFloat(getEnv("FRAME_RATE", "1"), 64)
	if err != nil || frameRate <= 0 {
		frameRate = 1
	}
	cfg.FrameRate = frameRate

	// Parse TTL (default: 2x frame interval, minimum 10s)
	ttlSeconds, err := strconv.Atoi(getEnv("FRAME_TTL_SECONDS", "0"))
	if err != nil || ttlSeconds <= 0 {
		// Default: 2x frame interval
		frameInterval := 1.0 / cfg.FrameRate
		ttlSeconds = int(frameInterval * 2)
		if ttlSeconds < 10 {
			ttlSeconds = 10 // minimum 10 seconds
		}
	}
	cfg.FrameTTL = time.Duration(ttlSeconds) * time.Second

	// redis queue mode
	if getEnv("REDIS_QUEUE_MODE", "false") == "true" {
		cfg.QueueMode = true
	}

	return cfg
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.VideoPath == "" {
		return &ConfigError{Field: "VIDEO_PATH", Message: "video path is required"}
	}
	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
