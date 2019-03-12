# updog

[![Go Report Card](https://goreportcard.com/badge/github.com/benclapp/updog)](https://goreportcard.com/report/github.com/benclapp/updog)
[![Docker Pulls](https://img.shields.io/docker/pulls/benclapp/updog.svg?maxAge=604800)](https://hub.docker.com/r/benclapp/updog)

Updog is a health check aggregator for scenarios where you have multiple micro services running in a group. For example you may have many data centres running the same services. Your geo-steered loadbalancer could hit updog's `/health` endpoint for the status of each data centre.

Checks of dependencies are executed in parallel, to ensure one slow dependency doesn't risk causing an upstream timeout. The status of each dependency check is returned for adhoc debugging. Any response between `200 ≤ x ≤ 299` will succeed. `/health` returns a 502 if any dependencies fail. Sample response:

```json
{
  "results": {
    "http": [
      {
        "name": "Google",
        "success": true,
        "duration": 1.003521066
      },
      {
        "name": "GitHub",
        "success": true,
        "duration": 1.040340163
      },
      {
        "name": "Failure Dependency",
        "success": false,
        "duration": 1.242335411,
        "httpStatus": "404 Not Found"
      }
    ]
  }
}
```

## Configuration

Configuration of the downstream dependencies is done with a simple YAML file.

```yaml
dependencies:
  http:
  - name: Google
    http_endpoint: https://google.com
  - name: GitHub
    http_endpoint: https://github.com
  - name: Prometheus
    http_endpoint: http://demo.robustperception.io:9090/-/healthy
```

### Flags

The following flags can be supplied on startup.

```
Flags:
      --help                    Show context-sensitive help (also try --help-long and --help-man).
  -t, --timeout=5s              Timeout for dependency checks
  -c, --config.path="updog.yaml"
                                Path of configuration file
      --listen.address=":1111"  Address to listen on for HTTP requests
      --version                 Show application version.
```

## Metrics

Metrics are exposed in the Prometheus format, at the standard `/metrics` endpoint.

Name | Description | Type | Labels
-----|-------------|------|-------
`updog_http_request_duration_seconds` | Inbound request latency. | Histogram | `path`
`updog_dependency_duration_seconds` | Latency of the outbound dependency check. | Histogram | `dependency`
`updog_dependency_checks_total` | Count of total health checks per dependency. | Counter | `dependency`
`updog_dependency_check_failures_total` | Count of total health check failures per dependency. | Counter | `dependency`
