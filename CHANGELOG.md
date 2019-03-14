## 0.2.0

- [CHANGE] Refactor config file and `/health` for nicer handling of multiple dependency types
- [FEATURE] Add Redis health checks, both secure and insecure

## 0.1.1

- Fix typo for dependency label name

## 0.1.0

Initial release! 

Docker images available at [Docker Hub](https://hub.docker.com/r/benclapp/updog)

- Aggregate http health checks by hitting `/health` endpoint
- Expose metrics for golang runtime, http request latency, and health check latency
