package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Publisher publishes frames to Redis
type Publisher struct {
	client    *redis.Client
	streamID  string
	ttl       time.Duration
	queueMode bool
}

// NewPublisher creates a new Redis publisher
func NewPublisher(addr, streamID string, ttl time.Duration, queueMode bool) *Publisher {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "", // No password by default
		DB:           0,  // Default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &Publisher{
		client:    client,
		streamID:  streamID,
		ttl:       ttl,
		queueMode: queueMode,
	}
}

// frameKey returns the Redis key for the current frame
func (p *Publisher) frameKey() string {
	return fmt.Sprintf("stream:%s:frame", p.streamID)
}

// Ping tests the Redis connection
func (p *Publisher) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

// PublishFrame stores a frame in Redis with TTL
func (p *Publisher) PublishFrame(ctx context.Context, data []byte) error {
	key := p.frameKey()

	if p.queueMode {
		// RPUSH to a list for queue-based consumption
		// Appends to the tail of the list
		err := p.client.RPush(ctx, key, data).Err()
		if err != nil {
			return fmt.Errorf("failed to publish frame to queue: %w", err)
		}
		// Set TTL on the list key if it's new (or refresh it)
		p.client.Expire(ctx, key, p.ttl)

		log.Printf("pushed frame to queue: key=%s, size=%d bytes", key, len(data))
	} else {
		// SET with TTL (overwrites previous frame)
		err := p.client.Set(ctx, key, data, p.ttl).Err()
		if err != nil {
			return fmt.Errorf("failed to publish frame: %w", err)
		}

		log.Printf("published frame: key=%s, size=%d bytes, ttl=%s",
			key, len(data), p.ttl)
	}

	return nil
}

// Close closes the Redis connection
func (p *Publisher) Close() error {
	return p.client.Close()
}
