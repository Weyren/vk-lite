package infra

import (
	"github.com/Weyren/vk-lite/pkg/utils"
	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg *utils.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		DB:   0,
	})
}
