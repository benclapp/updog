package checks

import (
	"net/http"
	"time"
)

const HTTP_TYPE = "http"

type HttpChecker struct {
	name       string
	endpoint   string
	httpClient *http.Client
}

func NewHttpChecker(name, endpoint string, timeout time.Duration) *HttpChecker {
	httpClient := http.Client{Timeout: timeout}
	return &HttpChecker{name: name, endpoint: endpoint, httpClient: &httpClient}
}

func (receiver HttpChecker) Check() Result {
	start := time.Now()
	res, err := receiver.httpClient.Get(receiver.endpoint)
	elapsed := float64(time.Since(start).Seconds())

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "dependency", receiver.name, "err", err)
		return Result{Name: receiver.name, Typez: HTTP_TYPE, Success: false, Duration: elapsed, Err: err}
	} else if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return Result{Name: receiver.name, Typez: HTTP_TYPE, Success: true, Duration: elapsed, Err: nil}
	} else {
		logger.Log("msg", "health check dependency failed", "dependency_name", receiver.name, "response_code", res.Status, "duration", elapsed)
		return Result{Name: receiver.name, Typez: HTTP_TYPE, Success: false, Duration: elapsed, Reason: res.Status, Err: nil}
	}
}
