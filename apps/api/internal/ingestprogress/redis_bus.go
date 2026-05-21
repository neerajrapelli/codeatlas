package ingestprogress

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBus uses Redis Pub/Sub for multi-instance SSE; keeps a local latest-event cache.
type RedisBus struct {
	client *redis.Client
	prefix string
	cache  *Broadcaster
	logger *slog.Logger
}

func NewRedisBus(redisURL string, logger *slog.Logger) (EventBus, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisBus{
		client: client,
		prefix: "codeatlas:ingest:",
		cache:  NewBroadcaster(),
		logger: logger,
	}, nil
}

func (b *RedisBus) channel(repositoryID int64) string {
	return b.prefix + strconv.FormatInt(repositoryID, 10)
}

func (b *RedisBus) Publish(repositoryID int64, ev StreamEvent) {
	b.cache.Publish(repositoryID, ev)
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := b.client.Publish(ctx, b.channel(repositoryID), payload).Err(); err != nil && b.logger != nil {
		b.logger.Warn("ingest_progress_publish_failed", "repository_id", repositoryID, "error", err)
	}
}

func (b *RedisBus) Latest(repositoryID int64) (StreamEvent, bool) {
	return b.cache.Get(repositoryID)
}

func (b *RedisBus) Subscribe(ctx context.Context, repositoryID int64) (<-chan StreamEvent, func(), error) {
	pubsub := b.client.Subscribe(ctx, b.channel(repositoryID))
	if err := pubsub.Ping(ctx); err != nil {
		_ = pubsub.Close()
		return nil, nil, err
	}
	out := make(chan StreamEvent, 8)
	var once sync.Once
	stop := make(chan struct{})
	go func() {
		defer close(out)
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var ev StreamEvent
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					continue
				}
				b.cache.Publish(repositoryID, ev)
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				case <-stop:
					return
				}
			}
		}
	}()
	cleanup := func() {
		once.Do(func() {
			close(stop)
			_ = pubsub.Close()
		})
	}
	return out, cleanup, nil
}

// NewEventBus returns Redis when redisURL is set and reachable; otherwise in-memory.
func NewEventBus(redisURL string, logger *slog.Logger) EventBus {
	if redisURL == "" {
		if logger != nil {
			logger.Info("ingest_progress_bus", "backend", "memory")
		}
		return NewBroadcaster()
	}
	bus, err := NewRedisBus(redisURL, logger)
	if err != nil {
		if logger != nil {
			logger.Warn("ingest_progress_bus_redis_unavailable", "error", err, "fallback", "memory")
		}
		return NewBroadcaster()
	}
	if logger != nil {
		logger.Info("ingest_progress_bus", "backend", "redis")
	}
	return bus
}
