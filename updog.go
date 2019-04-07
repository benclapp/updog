package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	logg "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var version string

var logger logg.Logger

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
		[]string{"dependency", "dependency_type"},
	)

	healthChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_checks_total",
			Help: "Count of total health checks per dependency",
		},
		[]string{"dependency", "dependency_type"},
	)

	healthChecksFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_check_failures_total",
			Help: "Count of total health check failures per dependency",
		},
		[]string{"dependency", "dependency_type"},
	)
)

func init() {
	logger = logg.NewLogfmtLogger(logg.NewSyncWriter(os.Stderr))
	logger = logg.With(logger, "ts", logg.DefaultTimestampUTC, "caller", logg.DefaultCaller)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(version)
	kingpin.CommandLine.Help = "Service to aggregate health checks. Returns 502 if any fail."
	kingpin.Parse()

	configure()

	prometheus.MustRegister(
		httpDurationsHistogram,
		healthCheckDependencyDuration,
		healthChecksTotal,
		healthChecksFailuresTotal,
	)

	logger.Log("msg", "Congigured Dependencies...")
	initHTTP()
	initRedis()
	initSQL()
	initRabbit()
	logger.Log("msg", "Finished initilisation")
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/updog", handleHealth)
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
	sqlCh := make(chan sqlResult)
	// rabbitCh := make(chan rabbitResult)

	results := resultResponse{}
	pass := true

	// Start health checks concurrently
	for _, dep := range config.Dependencies.HTTP {
		go checkHTTP(dep.HTTPEndpoint, dep.Name, httpCh)
	}
	for _, redCli := range redisClients {
		go checkRedis(redCli, redisCh)
	}
	for _, sqlCli := range sqlClients {
		go checkSQL(sqlCli.Name, sqlCli.Type, sqlCli.Db, sqlCh)
	}
	for _, rabbit := range rabbitClients {
		go checkRabbit(rabbit)
	}

	//Wait for health checks to return
	for range config.Dependencies.HTTP {
		res := <-httpCh
		results.Dependencies.HTTPResult = append(results.Dependencies.HTTPResult, res)
		if res.Success == false {
			pass = false
		}
	}
	for range redisClients {
		res := <-redisCh
		results.Dependencies.RedisResult = append(results.Dependencies.RedisResult, res)
		if res.Success == false {
			pass = false
		}
	}
	for range sqlClients {
		res := <-sqlCh
		results.Dependencies.SqlResult = append(results.Dependencies.SqlResult, res)
		if res.Success == false {
			pass = false
		}
	}
	// for range config.Dependencies.RabbitMQ {
	// 	res := <-rabbitCh
	// 	results.Dependencies.RabbitResult = append(results.Dependencies.RabbitResult, res)
	// 	if res.Success == false {
	// 		pass = false
	// 	}
	// }

	// Return 502 if any dependencies failed
	if pass == false {
		w.WriteHeader(http.StatusBadGateway)
	}

	response, _ := json.MarshalIndent(&results, "", "    ")
	fmt.Fprintf(w, string(response))

	httpDurationsHistogram.WithLabelValues(r.RequestURI).Observe(time.Since(start).Seconds())
}

type resultResponse struct {
	Dependencies struct {
		HTTPResult   []HTTPResult   `json:"http"`
		RedisResult  []RedisResult  `json:"redis"`
		SqlResult    []sqlResult    `json:"sql"`
		RabbitResult []rabbitResult `json:"rabbitMQ"`
	} `json:"results"`
}
