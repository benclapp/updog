package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	logg "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

var version string
var gitCommit string

var logger logg.Logger

var config = conf{}

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
		logger.Log("dependency_name", dep.Name, "dependency_type", "http", "dependency_endpoint", dep.HTTPEndpoint)

		healthCheckDependencyDuration.WithLabelValues(dep.Name).Observe(0)
		healthChecksTotal.WithLabelValues(dep.Name).Add(0)
		healthChecksFailuresTotal.WithLabelValues(dep.Name).Add(0)
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
	ch := make(chan HTTPResult)

	for _, dep := range config.Dependencies.HTTP {
		go checkHealth(dep.HTTPEndpoint, dep.Name, ch)
	}

	var results = resultResponse{}
	var pass = true
	for range config.Dependencies.HTTP {
		res := <-ch

		results.Dependencies.HTTP = append(results.Dependencies.HTTP, res)
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

func checkHealth(e, n string, ch chan<- HTTPResult) {
	start := time.Now()
	// logger.Log("msg", "Starting dep check for", n, "at", start.UnixNano())

	// _timeout := time.Duration(depTimeout)
	client := http.Client{
		Timeout: depTimeout,
	}

	//hit endpoint
	resp, err := client.Get(e)

	//stop timing
	elapsed := float64(time.Since(start).Seconds())

	healthCheckDependencyDuration.WithLabelValues(n).Observe(elapsed)
	healthChecksTotal.WithLabelValues(n).Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "err", err)
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
		HTTP []struct {
			Name       string  `json:"name"`
			Success    bool    `json:"success"`
			Duration   float64 `json:"duration"`
			HTTPStatus string  `json:"httpStatus,omitempty"`
		} `json:"http"`
	} `json:"results"`
}

type HTTPResult struct {
	Name       string  `json:"name"`
	Success    bool    `json:"success"`
	Duration   float64 `json:"duration"`
	HTTPStatus string  `json:"httpStatus,omitempty"`
}

type conf struct {
	Dependencies struct {
		HTTP []struct {
			HTTPEndpoint string `yaml:"http_endpoint"`
			Name         string `yaml:"name"`
		} `yaml:"http"`
	} `yaml:"dependencies"`
}
