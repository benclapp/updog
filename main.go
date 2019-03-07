package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	yaml "gopkg.in/yaml.v2"
)

var config = conf{}

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

var (
	httpDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP Latency histogram",
	})

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

	config.getConf()

	prometheus.MustRegister(httpDurationsHistogram)
	prometheus.MustRegister(healthCheckDependencyDuration)
	prometheus.MustRegister(healthChecksTotal)
	prometheus.MustRegister(healthChecksFailuresTotal)

	fmt.Println("Dependencies:")
	for _, dep := range config.Dependencies {
		println(dep.Name, ":", dep.HTTPEndpoint)
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
	fmt.Fprintf(w, "pong")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	//declare status and duration channels. Perhaps a channel as a struct?
	ch := make(chan Result)

	for _, dep := range config.Dependencies {
		go checkHealth(dep.HTTPEndpoint, dep.Name, dep.Type, ch)
	}

	var results = resultResponse{}
	for range config.Dependencies {
		res := <-ch
		fmt.Println("Returned result:", res)

		results.Result = append(results.Result, res)
		if res.Success == false {
			w.WriteHeader(http.StatusBadGateway)
		}
	}
	fmt.Println("Returned results:", results)

	re := &results
	response, _ := json.MarshalIndent(re, "", "    ")

	fmt.Fprintf(w, string(response))

}

func checkHealth(e, n, t string, ch chan<- Result) {
	start := time.Now()
	// fmt.Println("Starting dep check for", n, "at", start.UnixNano())

	//hit endpoint
	resp, err := http.Get(e)
	if err != nil {
		log.Fatal(err)
	}

	//stop timing
	elapsed := float64(time.Since(start).Seconds())
	// fmt.Println("Ending dep check for", n, "after", elapsed)

	healthCheckDependencyDuration.WithLabelValues(n).Observe(elapsed)
	healthChecksTotal.WithLabelValues(n).Inc()

	//check response code
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		ch <- Result{Name: n, Type: t, Success: true, Duration: elapsed}
	} else {
		healthChecksFailuresTotal.WithLabelValues(n).Inc()
		fmt.Println("Check failed for", n, "response code: ", resp.Status)
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
