package main

import (
	"crypto/tls"
	"time"

	"github.com/go-redis/redis"
)

var redisClients = redisClientList{}

func initRedis() {
	for _, red := range config.Dependencies.Redis {
		if red.Ssl {
			redCli := redisClient{
				name: red.Name,
				client: redis.NewClient(
					&redis.Options{
						Addr:      red.Address,
						Password:  red.Password,
						DB:        0,
						TLSConfig: &tls.Config{},
					}),
			}
			redisClients = append(redisClients, redCli)
		} else {
			redCli := redisClient{
				name: red.Name,
				client: redis.NewClient(
					&redis.Options{
						Addr:     red.Address,
						Password: red.Password,
						DB:       0,
					}),
			}
			redisClients = append(redisClients, redCli)
		}

		logger.Log("dependency_type", "Redis", "dependency_name", red.Name, "redis_address", red.Address, "redis_password", "hunter2********")

		healthCheckDependencyDuration.WithLabelValues(red.Name, "redis").Observe(0)
		healthChecksTotal.WithLabelValues(red.Name, "redis").Add(0)
		healthChecksFailuresTotal.WithLabelValues(red.Name, "redis").Add(0)
	}
}

func checkRedis(rc redisClient, ch chan<- RedisResult) {
	start := time.Now()
	pong, err := rc.client.Ping().Result()
	elapsed := time.Since(start).Seconds()

	healthCheckDependencyDuration.WithLabelValues(rc.name, "redis").Observe(elapsed)
	healthChecksTotal.WithLabelValues(rc.name, "redis").Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "type", "Redis", "dependency", rc.name, "err", err)
		healthChecksFailuresTotal.WithLabelValues(rc.name, "redis").Inc()
		ch <- RedisResult{Name: rc.name, Success: false, Duration: elapsed, Err: err}
	} else {
		ch <- RedisResult{Name: rc.name, Success: true, Duration: elapsed, response: pong}
	}
}

type RedisResult struct {
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	response string
	Err      error `json:"error,omitempty"`
}

type redisClientList []struct {
	name   string
	client *redis.Client
}

type redisClient struct {
	name   string
	client *redis.Client
}
