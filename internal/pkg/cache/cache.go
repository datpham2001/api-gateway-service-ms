package cache

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	logger      *logger.Logger
	cacheClient *redis.Client
}

func NewCacheClient(logger *logger.Logger, cfg *config.Config) *Cache {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Cache.Host, cfg.Cache.Port),
		Password: cfg.Cache.Password,
		DB:       cfg.Cache.DB,
	})

	return &Cache{
		logger:      logger,
		cacheClient: redisClient,
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	if err := c.cacheClient.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}

func (c *Cache) Get(ctx context.Context, key string, obj interface{}) error {
	result, err := c.cacheClient.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(result), &obj)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.cacheClient.Set(ctx, key, value, expiration).Err(); err != nil {
		return err
	}

	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.cacheClient.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.cacheClient.TTL(ctx, key).Result()
}

func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.cacheClient.Incr(ctx, key).Result()
}

func (c *Cache) Close() error {
	return c.cacheClient.Close()
}
