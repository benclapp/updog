package main

import (
	"net/http"
	"time"
)

var httpClient = http.Client{}

func initHTTP() {
	httpClient = http.Client{
		Timeout: depTimeout,
	}

	for _, dep := range config.Dependencies.HTTP {
		logger.Log("dependency_type", "http", "dependency_name", dep.Name, "dependency_endpoint", dep.HTTPEndpoint)

		healthCheckDependencyDuration.WithLabelValues(dep.Name, "http").Observe(0)
		healthChecksTotal.WithLabelValues(dep.Name, "http").Add(0)
		healthChecksFailuresTotal.WithLabelValues(dep.Name, "http").Add(0)
	}
}

func checkHTTP(e, n string, ch chan<- HTTPResult) {
	start := time.Now()
	resp, err := httpClient.Get(e)
	elapsed := float64(time.Since(start).Seconds())

	healthCheckDependencyDuration.WithLabelValues(n, "http").Observe(elapsed)
	healthChecksTotal.WithLabelValues(n, "http").Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "dependency", n, "err", err)
		healthChecksFailuresTotal.WithLabelValues(n, "http").Inc()
		ch <- HTTPResult{Name: n, Success: false, Duration: elapsed, Err: err}
	} else if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		ch <- HTTPResult{Name: n, Success: true, Duration: elapsed}
	} else {
		healthChecksFailuresTotal.WithLabelValues(n, "http").Inc()
		logger.Log("msg", "health check dependency failed", "dependency_name", n, "response_code", resp.Status, "duration", elapsed)
		ch <- HTTPResult{Name: n, Success: false, Duration: elapsed, HTTPStatus: resp.Status}
	}
}

type HTTPResult struct {
	Name       string  `json:"name"`
	Success    bool    `json:"success"`
	Duration   float64 `json:"duration"`
	HTTPStatus string  `json:"httpStatus,omitempty"`
	Err        error   `json:"error,omitempty"`
}
