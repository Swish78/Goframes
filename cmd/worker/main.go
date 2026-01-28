package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swayampatil/frameextracter/internal/config"
	"github.com/swayampatil/frameextracter/internal/ffmpeg"
	"github.com/swayampatil/frameextracter/internal/parser"
	"github.com/swayampatil/frameextracter/internal/redis"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("frame extraction worker starting...")

	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	log.Printf("config: video=%s, stream=%s, fps=%.2f, ttl=%s",
		cfg.VideoPath, cfg.StreamID, cfg.FrameRate, cfg.FrameTTL)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("received signal: %v, shutting down...", sig)
		cancel()
	}()

	// Connect to Redis
	publisher := redis.NewPublisher(cfg.RedisAddr, cfg.StreamID, cfg.FrameTTL, cfg.QueueMode)
	defer publisher.Close()

	if err := publisher.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}
	log.Println("connected to Redis")

	// Main loop: run ffmpeg with restart on failure
	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutdown complete")
			return
		default:
		}

		if err := runPipeline(ctx, cfg, publisher); err != nil {
			if ctx.Err() != nil {
				// Context cancelled, exit gracefully
				log.Println("worker shutdown complete")
				return
			}
			log.Printf("pipeline error: %v, restarting in %s...", err, cfg.RestartDelay)
			time.Sleep(cfg.RestartDelay)
		}
	}
}

// runPipeline runs the ffmpeg -> parser -> redis pipeline
func runPipeline(ctx context.Context, cfg *config.Config, publisher *redis.Publisher) error {
	// Create ffmpeg process
	proc := ffmpeg.NewProcess(cfg.VideoPath, cfg.FrameRate, cfg.FFmpegPath)

	// Start ffmpeg
	stdout, err := proc.Start(ctx)
	if err != nil {
		return err
	}
	defer proc.Stop()

	// Create frame channel
	frames := make(chan parser.Frame, 1)

	// Start parser in goroutine
	parserCtx, parserCancel := context.WithCancel(ctx)
	defer parserCancel()

	go func() {
		p := parser.NewParser(stdout)
		if err := p.ParseFrames(parserCtx, frames); err != nil && parserCtx.Err() == nil {
			log.Printf("parser error: %v", err)
		}
	}()

	// Process frames
	frameCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case frame, ok := <-frames:
			if !ok {
				// Parser finished (ffmpeg exited or EOF)
				log.Printf("processed %d frames, ffmpeg exited", frameCount)
				return proc.Wait()
			}

			frameCount++

			// Publish to Redis (latest frame only)
			if err := publisher.PublishFrame(ctx, frame.Data); err != nil {
				log.Printf("publish error: %v", err)
				// Continue processing, don't fail the whole pipeline
			}
		}
	}
}
