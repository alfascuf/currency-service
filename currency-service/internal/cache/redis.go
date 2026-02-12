package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alfascuf/currency-service/internal/config"
	"github.com/alfascuf/currency-service/internal/logger"
	"github.com/alfascuf/currency-service/internal/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Cache provides caching operations for currency rates
type Cache interface {
	GetRate(ctx context.Context, base, target, date string) (*models.ExchangeRate, error)
	SetRate(ctx context.Context, rate *models.ExchangeRate) error
	GetHistory(ctx context.Context, base, target, startDate, endDate string) ([]models.ExchangeRate, error)
	SetHistory(ctx context.Context, base, target, startDate, endDate string, rates []models.ExchangeRate) error
	Delete(ctx context.Context, key string) error
	Close() error
}

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// New creates a new Redis cache instance
func New(cfg *config.Config) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Log.Info("Connected to Redis successfully",
		zap.String("host", cfg.RedisHost),
		zap.String("port", cfg.RedisPort),
	)

	return &redisCache{
		client: client,
		ttl:    time.Duration(cfg.CacheTTL) * time.Second,
	}, nil
}

// GetRate retrieves a single rate from cache
func (c *redisCache) GetRate(ctx context.Context, base, target, date string) (*models.ExchangeRate, error) {
	key := fmt.Sprintf("rate:%s:%s:%s", base, target, date)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		logger.Log.Warn("Redis GET error", zap.Error(err), zap.String("key", key))
		return nil, err
	}

	var rate models.ExchangeRate
	if err := json.Unmarshal([]byte(val), &rate); err != nil {
		logger.Log.Warn("Failed to unmarshal cached rate", zap.Error(err))
		return nil, err
	}

	logger.Log.Debug("Cache hit", zap.String("key", key))
	return &rate, nil
}

// SetRate stores a single rate in cache
func (c *redisCache) SetRate(ctx context.Context, rate *models.ExchangeRate) error {
	key := fmt.Sprintf("rate:%s:%s:%s", rate.Base, rate.Target, rate.Date.Format("2006-01-02"))

	data, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("failed to marshal rate: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		logger.Log.Warn("Redis SET error", zap.Error(err), zap.String("key", key))
		return err
	}

	logger.Log.Debug("Cached rate", zap.String("key", key), zap.Duration("ttl", c.ttl))
	return nil
}

// GetHistory retrieves historical rates from cache
func (c *redisCache) GetHistory(ctx context.Context, base, target, startDate, endDate string) ([]models.ExchangeRate, error) {
	key := fmt.Sprintf("history:%s:%s:%s:%s", base, target, startDate, endDate)

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		logger.Log.Warn("Redis GET error", zap.Error(err), zap.String("key", key))
		return nil, err
	}

	var rates []models.ExchangeRate
	if err := json.Unmarshal([]byte(val), &rates); err != nil {
		logger.Log.Warn("Failed to unmarshal cached history", zap.Error(err))
		return nil, err
	}

	logger.Log.Debug("Cache hit", zap.String("key", key), zap.Int("count", len(rates)))
	return rates, nil
}

// SetHistory stores historical rates in cache
func (c *redisCache) SetHistory(ctx context.Context, base, target, startDate, endDate string, rates []models.ExchangeRate) error {
	key := fmt.Sprintf("history:%s:%s:%s:%s", base, target, startDate, endDate)

	data, err := json.Marshal(rates)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		logger.Log.Warn("Redis SET error", zap.Error(err), zap.String("key", key))
		return err
	}

	logger.Log.Debug("Cached history",
		zap.String("key", key),
		zap.Int("count", len(rates)),
		zap.Duration("ttl", c.ttl),
	)
	return nil
}

// Delete removes a key from cache
func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Close closes the Redis connection
func (c *redisCache) Close() error {
	return c.client.Close()
}
