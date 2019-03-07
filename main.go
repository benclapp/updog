package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	yaml "gopkg.in/yaml.v2"

	logg "github.com/go-kit/kit/log"
)

var logger logg.Logger

var config = conf{}

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

var (
	httpDurationsHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP Latency histogram",
		},
		[]string{"handler"},
	)

	healthCheckDependencyDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "updog_health_check_dependency_duration_seconds",
			Help: "Duration of a health check dependency in seconds",
		},
		[]string{"depenency"},
	)

	healthChecksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_health_checks_total",
			Help: "Count of total health checks per dependency",
		},
		[]string{"dependency"},
	)

	healthChecksFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "updog_dependency_health_check_failures_total",
			Help: "Count of total health check failures per dependency",
		},
		[]string{"dependency"},
	)
)

func init() {

	logger = logg.NewLogfmtLogger(logg.NewSyncWriter(os.Stderr))
	logger = logg.With(logger, "ts", logg.DefaultTimestampUTC, "caller", logg.DefaultCaller)

	logger.Log("msg", "Loading config")
	config.getConf()

	prometheus.MustRegister(
		httpDurationsHistogram,
		healthCheckDependencyDuration,
		healthChecksTotal,
		healthChecksFailuresTotal,
	)

	logger.Log("msg", "Dependencies:")
	for _, dep := range config.Dependencies {
		logger.Log("dependency_name", dep.Name, "dependency_type", dep.Type, "dependency_endpoint", dep.HTTPEndpoint)

		healthCheckDependencyDuration.WithLabelValues(dep.Name).Observe(0)
		healthChecksTotal.WithLabelValues(dep.Name).Add(0)
		healthChecksFailuresTotal.WithLabelValues(dep.Name).Add(0)
	}
}

func main() {
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", handlePing)
	http.HandleFunc("/health", handleHealth)

	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Fprintf(w, "pong")
	httpDurationsHistogram.WithLabelValues("ping").Observe(time.Since(start).Seconds())
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ch := make(chan Result)

	for _, dep := range config.Dependencies {
		go checkHealth(dep.HTTPEndpoint, dep.Name, dep.Type, ch)
	}

	var results = resultResponse{}
	var pass = true
	for range config.Dependencies {
		res := <-ch

		results.Result = append(results.Result, res)
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

	httpDurationsHistogram.WithLabelValues("health").Observe(time.Since(start).Seconds())
}

func checkHealth(e, n, t string, ch chan<- Result) {
	start := time.Now()
	// logger.Log("msg", "Starting dep check for", n, "at", start.UnixNano())

	timeout := time.Duration(9500 * time.Millisecond)
	client := http.Client{
		Timeout: timeout,
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
		ch <- Result{Name: n, Type: t, Success: false, Duration: elapsed}
		return
	}

	//check response code
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		ch <- Result{Name: n, Type: t, Success: true, Duration: elapsed}
	} else {
		healthChecksFailuresTotal.WithLabelValues(n).Inc()
		logger.Log("msg", "health check dependency failed", "dependency_name", n, "response_code", resp.Status, "duration", elapsed)
		ch <- Result{Name: n, Type: t, Success: false, Duration: elapsed, HTTPStatus: resp.Status}
	}
}

type resultResponse struct {
	Result []struct {
		Name       string  `json:"name"`
		Type       string  `json:"type"`
		Success    bool    `json:"success"`
		Duration   float64 `json:"duration"`
		HTTPStatus string  `json:"httpStatus,omitempty"`
	} `json:"results"`
}

type Result struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Success    bool    `json:"success"`
	Duration   float64 `json:"duration"`
	HTTPStatus string  `json:"httpStatus,omitempty"`
}

type conf struct {
	Dependencies []struct {
		HTTPEndpoint string `yaml:"http_endpoint"`
		Name         string `yaml:"name"`
		Type         string `yaml:"type"`
	} `yaml:"dependencies"`
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile("sample-configuration.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
