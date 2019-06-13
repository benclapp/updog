## Next Release

- [CHANGE] Remove RabbitMQ dependency checks

## 0.4.1

- [FIX] Fix typo in RabbitMQ init log

## 0.4.0

- [CHANGE] Use 503 response for failures rather than 502
- [FEATURE] Add `/updog` endpoint

## 0.3.0

- [FEATURE] Add `dependency_type` label to dependency metrics
- [FEATURE] Add support for SQL checks. Postgres and MSSQL 

## 0.2.0

- [CHANGE] Refactor config file and `/health` for nicer handling of multiple dependency types
- [CHANGE] Return more detailed error for dependency failures
- [FEATURE] Add Redis health checks, both secure and insecure


## 0.1.1

- Fix typo for dependency label name

## 0.1.0

Initial release! 

Docker images available at [Docker Hub](https://hub.docker.com/r/benclapp/updog)

- Aggregate http health checks by hitting `/health` endpoint
- Expose metrics for golang runtime, http request latency, and health check latency
