package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/benclapp/updog/checks"
	"github.com/benclapp/updog/config"
)

const SPACE = " "

var version string
var logger log.Logger = level.NewFilter(log.NewLogfmtLogger(os.Stdout), level.AllowInfo())
var cfg config.Configurable
var checkersRegistry []checks.Checker

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
	config.Version = version
	cfg = config.NewConfig()

	prometheus.MustRegister(
		httpDurationsHistogram,
		healthCheckDependencyDuration,
		healthChecksTotal,
		healthChecksFailuresTotal,
	)

	//Create Checkers
	level.Info(logger).Log("msg", "Configured Dependencies...")

	for _, dep := range cfg.GetDependencies().HTTP {
		checkersRegistry = append(checkersRegistry, checks.NewHttpChecker(dep.Name, dep.HTTPEndpoint, cfg.GetTimeout()))
		initMetrics(dep.Name, checks.HTTP_TYPE)
	}
	for _, dep := range cfg.GetDependencies().Redis {
		checkersRegistry = append(checkersRegistry, checks.NewRedisChecker(dep.Name, dep.Address, dep.Password, cfg.GetTimeout(), dep.Ssl))
		initMetrics(dep.Name, checks.REDIS_TYPE)
	}
	for _, dep := range cfg.GetDependencies().SQL {
		checkersRegistry = append(checkersRegistry, checks.NewSqlChecker(dep.Name, dep.Type, dep.ConnectionString))
		initMetrics(dep.Name, checks.SQL_TYPE)
	}

	level.Info(logger).Log("msg", "Finished initialisation")
}

func initMetrics(name string, typez string) {
	healthCheckDependencyDuration.WithLabelValues(name, typez).Observe(0)
	healthChecksTotal.WithLabelValues(name, typez).Add(0)
	healthChecksFailuresTotal.WithLabelValues(name, typez).Add(0)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/updog", handleHealth)
	http.HandleFunc("/health", handleHealth)

	level.Error(logger).Log("err", http.ListenAndServe(cfg.GetAddress(), nil))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Fprintf(w, "pong")
	httpDurationsHistogram.WithLabelValues("/ping").Observe(time.Since(start).Seconds())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	level.Debug(logger).Log("msg", "handleHealth")
	start := time.Now()
	defer func() { httpDurationsHistogram.WithLabelValues(r.RequestURI).Observe(time.Since(start).Seconds()) }()

	//loop registered checkers
	resultChan := make(chan checks.Result)
	var wg sync.WaitGroup

	// Start health checks concurrently
	//FIXME https://stackoverflow.com/questions/46010836/using-goroutines-to-process-values-and-gather-results-into-a-slice
	for _, checker := range checkersRegistry {
		wg.Add(1)
		// Process each item with a goroutine and send output to resultChan
		go func(checker checks.Checker) {
			defer wg.Done()
			resultChan <- checker.Check()
		}(checker)
	}

	//Use countdown latch to close after checks return
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make(map[string]checks.Result)
	pass := true
	for r := range resultChan {
		level.Debug(logger).Log("msg", "Dependency returned", r.Name, r.Success)

		pass = pass && r.Success
		results[r.Name] = r

		healthCheckDependencyDuration.WithLabelValues(r.Name, r.Typez).Observe(r.Duration)
		healthChecksTotal.WithLabelValues(r.Name, r.Typez).Inc()

		if r.Err != nil {
			healthChecksFailuresTotal.WithLabelValues(r.Name, r.Typez).Inc()
		}
	}

	// Return 503 if any of the dependencies failed
	if pass == false {
		level.Warn(logger).Log("msg", "StatusServiceUnavailable")
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response, _ := json.MarshalIndent(&results, "", SPACE)
	fmt.Fprintf(w, string(response))
}
