package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type CacheService struct {
	client     *redis.Client
	defaultTTL time.Duration
}

func NewRedisClient(cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	return client, nil
}

func NewCacheService(c *redis.Client) *CacheService {
	return &CacheService{client: c, defaultTTL: 10 * time.Minute}
}

func (c *CacheService) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, bytes, ttl).Err()
}

func (c *CacheService) Get(ctx context.Context, key string, dest any) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *CacheService) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}


func (c *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("scan for pattern %q failed: %w", pattern, err)
	}
	if len(keys) == 0 {
		return nil
	}
	if err = c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("delete keys for pattern %q failed: %w", pattern, err)
	}
	logrus.WithFields(logrus.Fields{
		"pattern": pattern,
		"count":   len(keys),
	}).Debug("cache keys deleted")
	return nil
}

func (c *CacheService) Close() error { return c.client.Close() }
