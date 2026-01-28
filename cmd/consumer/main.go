package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/swayampatil/frameextracter/internal/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("frame consumer starting...")

	// Load configuration
	cfg := config.Load()

	// Ensure output directory exists
	outputDir := "captured_frames"
	if envDir := os.Getenv("OUTPUT_DIR"); envDir != "" {
		outputDir = envDir
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}
	log.Printf("saving frames to: %s", outputDir)

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer rdb.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("shutting down...")
		cancel()
	}()

	// Construct the key to listen to
	// Note: We need to match the key format from the publisher: stream:{streamID}:frame 
	// This is sort of placeholder, will be changed to use actual stream ID once we are able to get proper stream ID from android app
	streamKey := fmt.Sprintf("stream:%s:frame", cfg.StreamID)
	log.Printf("listening on list key: %s", streamKey)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// BLPOP: blocking list pop
			// 0 means no timeout (block indefinitely), but we might want a small timeout to check for context cancellation
			result, err := rdb.BLPop(ctx, 1*time.Second, streamKey).Result()
			if err != nil {
				if err == redis.Nil {
					// Timeout, just continue loop to check context
					continue
				}
				if ctx.Err() != nil {
					return
				}
				log.Printf("error popping frame: %v", err)
				time.Sleep(1 * time.Second) // backoff
				continue
			}

			// result[0] is the key, result[1] is the value
			frameData := []byte(result[1])

			// Generate filename with timestamp
			timestamp := time.Now().Format("20060102-150405.000")
			filename := fmt.Sprintf("frame-%s.jpg", timestamp)
			filepath := filepath.Join(outputDir, filename)

			if err := os.WriteFile(filepath, frameData, 0644); err != nil {
				log.Printf("failed to write file %s: %v", filename, err)
			} else {
				log.Printf("saved %s (%d bytes)", filename, len(frameData))
			}
		}
	}
}
