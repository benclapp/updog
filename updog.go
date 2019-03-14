package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	logg "github.com/go-kit/kit/log"
	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

var version string
var gitCommit string

var logger logg.Logger

var config = conf{}

var httpClient = http.Client{}
var redisClients = redisClientList{}

var (
	timeout       = kingpin.Flag("timeout", "Timeout for dependency checks").Short('t').Default("5s").Duration()
	configPath    = kingpin.Flag("config.path", "Path of configuration file").Short('c').Default("updog.yaml").String()
	listenAddress = kingpin.Flag("listen.address", "Address to listen on for HTTP requests").Default(":1111").String()
)

var (
	depTimeout time.Duration
	addr       string
)

var (
	httpDurationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "updog_http_request_duration_seconds",
			Help: "HTTP Latency histogram",
		},
		[]string{"path"},
	)

	healthCheckDependencyDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "updog_dependency_duration_seconds",
			Help: "Duration of a health check dependency in seconds",
		},
		[]string{"dependency"},
	)

	healthChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_checks_total",
			Help: "Count of total health checks per dependency",
		},
		[]string{"dependency"},
	)

	healthChecksFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_check_failures_total",
			Help: "Count of total health check failures per dependency",
		},
		[]string{"dependency"},
	)
)

func init() {
	logger = logg.NewLogfmtLogger(logg.NewSyncWriter(os.Stderr))
	logger = logg.With(logger, "ts", logg.DefaultTimestampUTC, "caller", logg.DefaultCaller)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(version)
	kingpin.CommandLine.Help = "Service to aggregate health checks. Returns 502 if any fail."
	kingpin.Parse()

	if configPath == nil {
		logger.Log("msg", "Config file required")
	}
	cfg := *configPath
	config.getConf(cfg)

	if timeout == nil {
		logger.Log("msg", "timeout required")
	}
	depTimeout = *timeout
	httpClient = http.Client{
		Timeout: depTimeout,
	}

	if listenAddress == nil {
		logger.Log("msg", "listen.address required")
	}
	addr = *listenAddress

	prometheus.MustRegister(
		httpDurationsHistogram,
		healthCheckDependencyDuration,
		healthChecksTotal,
		healthChecksFailuresTotal,
	)

	logger.Log("msg", "Dependencies:")
	for _, dep := range config.Dependencies.HTTP {
		logger.Log("dependency_type", "http", "dependency_name", dep.Name, "dependency_endpoint", dep.HTTPEndpoint)

		healthCheckDependencyDuration.WithLabelValues(dep.Name).Observe(0)
		healthChecksTotal.WithLabelValues(dep.Name).Add(0)
		healthChecksFailuresTotal.WithLabelValues(dep.Name).Add(0)
	}

	for _, red := range config.Dependencies.Redis {
		logger.Log("dependency_type", "Redis", "dependency_name", red.Name, "redis_address", red.Address, "redis_password", "hunter2********")

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
	}
}

func main() {

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/health", handleHealth)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Fprintf(w, "pong")
	httpDurationsHistogram.WithLabelValues("/ping").Observe(time.Since(start).Seconds())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	httpCh := make(chan HTTPResult)
	redisCh := make(chan RedisResult)
	results := resultResponse{}
	pass := true

	for _, dep := range config.Dependencies.HTTP {
		go checkHTTP(dep.HTTPEndpoint, dep.Name, httpCh)
	}
	for _, redCli := range redisClients {
		go checkRedis(redCli, redisCh)
	}

	for range config.Dependencies.HTTP {
		res := <-httpCh
		results.Dependencies.HTTPResult = append(results.Dependencies.HTTPResult, res)
		if res.Success == false {
			pass = false
		}
	}
	for range config.Dependencies.Redis {
		res := <-redisCh
		results.Dependencies.RedisResult = append(results.Dependencies.RedisResult, res)
		if res.Success == false {
			pass = false
		}
	}

	if pass == false {
		w.WriteHeader(http.StatusBadGateway)
	}

	re := &results
	response, _ := json.MarshalIndent(re, "", "    ")
	fmt.Fprintf(w, string(response))

	httpDurationsHistogram.WithLabelValues("/health").Observe(time.Since(start).Seconds())
}

func checkHTTP(e, n string, ch chan<- HTTPResult) {
	start := time.Now()

	//hit endpoint
	resp, err := httpClient.Get(e)

	//stop timing
	elapsed := float64(time.Since(start).Seconds())
	healthCheckDependencyDuration.WithLabelValues(n).Observe(elapsed)
	healthChecksTotal.WithLabelValues(n).Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "dependency", n, "err", err)
		healthChecksFailuresTotal.WithLabelValues(n).Inc()
		ch <- HTTPResult{Name: n, Success: false, Duration: elapsed}
		return
	}

	//check response code
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		ch <- HTTPResult{Name: n, Success: true, Duration: elapsed}
	} else {
		healthChecksFailuresTotal.WithLabelValues(n).Inc()
		logger.Log("msg", "health check dependency failed", "dependency_name", n, "response_code", resp.Status, "duration", elapsed)
		ch <- HTTPResult{Name: n, Success: false, Duration: elapsed, HTTPStatus: resp.Status}
	}
}

func checkRedis(rc redisClient, ch chan<- RedisResult) {
	start := time.Now()

	pong, err := rc.client.Ping().Result()

	elapsed := time.Since(start).Seconds()
	logger.Log("pong", pong, "err", err, "duration", elapsed)

	healthCheckDependencyDuration.WithLabelValues(rc.name).Observe(elapsed)
	healthChecksTotal.WithLabelValues(rc.name).Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "dependency", rc.name, "err", err)
		healthChecksFailuresTotal.WithLabelValues(rc.name).Inc()
		ch <- RedisResult{Name: rc.name, Success: false, Duration: elapsed}
	} else {
		ch <- RedisResult{Name: rc.name, Success: true, Duration: elapsed}
	}
}

func (c *conf) getConf(path string) *conf {

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

type resultResponse struct {
	Dependencies struct {
		HTTPResult []struct {
			Name       string  `json:"name"`
			Success    bool    `json:"success"`
			Duration   float64 `json:"duration"`
			HTTPStatus string  `json:"httpStatus,omitempty"`
		} `json:"http"`
		RedisResult []struct {
			Name     string  `json:"name"`
			Success  bool    `json:"success"`
			Duration float64 `json:"duration"`
		} `json:"redis"`
	} `json:"results"`
}

type HTTPResult struct {
	Name       string  `json:"name"`
	Success    bool    `json:"success"`
	Duration   float64 `json:"duration"`
	HTTPStatus string  `json:"httpStatus,omitempty"`
}

type RedisResult struct {
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
}

type redisClientList []struct {
	name   string
	client *redis.Client
}

type redisClient struct {
	name   string
	client *redis.Client
}

type conf struct {
	Dependencies struct {
		HTTP []struct {
			Name         string `yaml:"name"`
			HTTPEndpoint string `yaml:"http_endpoint"`
		} `yaml:"http"`
		Redis []struct {
			Name     string `yaml:"name"`
			Address  string `yaml:"address"`
			Password string `yaml:"password"`
			Ssl      bool   `yaml:"ssl"`
		}
	} `yaml:"dependencies"`
}
