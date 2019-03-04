package main

import (
	"fmt"
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

var (
	httpDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:	"http_request_duration_seconds",
		Help:	"HTTP Latency histogram",
	})
)

func init() {
	prometheus.MustRegister(httpDurationsHistogram)
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
	resp, err := http.Get("https://monitor.home.bencl.app/prometheus1/-/healthy")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("HTTP Response Status:", resp.Status)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		fmt.Println("It's up")
	} else {
		fmt.Println("It's down")
	}
}