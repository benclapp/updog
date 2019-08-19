package checks

import (
	"crypto/tls"
	"time"

	"github.com/go-redis/redis"
)

const REDIS_TYPE = "redis"

type RedisChecker struct {
	name   string
	client *redis.Client
}

func NewRedisChecker(name, address, password string, timeout time.Duration, isTls bool) *RedisChecker {
	var tlsConfig *tls.Config = nil
	if isTls {
		tlsConfig = &tls.Config{}
	}
	redisClient := redis.NewClient(
		&redis.Options{
			Addr:        address,
			Password:    password,
			DB:          0,
			ReadTimeout: timeout,
			TLSConfig:   tlsConfig,
		})
	return &RedisChecker{name: name, client: redisClient}
}

func (receiver *RedisChecker) Check() Result {
	start := time.Now()
	pong, err := receiver.client.Ping().Result()
	elapsed := time.Since(start).Seconds()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "type", "Redis", "dependency", receiver.name, "err", &err)
		return Result{Name: receiver.name, Typez: REDIS_TYPE, Success: false, Duration: elapsed, Err: err}
	} else {
		return Result{Name: receiver.name, Typez: REDIS_TYPE, Success: true, Duration: elapsed, Err: nil, Reason: pong}
	}
}
