package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/denisenkom/go-mssqldb" // MSSQL
	_ "github.com/lib/pq"                // Postgres
)

var sqlClients = sqlClientsList{}

func initSQL() {
	for _, db := range config.Dependencies.SQL {
		conn, err := sql.Open(db.Type, db.ConnectionString)
		if err != nil {
			log.Fatal("Opening db connection failed:", err.Error())
		}

		sqlCli := sqlClient{
			Name: db.Name,
			Db:   conn,
			Type: db.Type,
		}

		sqlClients = append(sqlClients, sqlCli)

		logger.Log("dependency_type", "sql", "dependency_name", string(db.Name), "db_type", db.Type)

		healthCheckDependencyDuration.WithLabelValues(db.Name, db.Type).Observe(0)
		healthChecksTotal.WithLabelValues(db.Name, db.Type).Add(0)
		healthChecksFailuresTotal.WithLabelValues(db.Name, db.Type).Add(0)
	}
}

func checkSQL(n, t string, db *sql.DB, ch chan<- sqlResult) {
	start := time.Now()
	err := db.Ping()
	elapsed := time.Since(start).Seconds()

	healthCheckDependencyDuration.WithLabelValues(n, t).Observe(elapsed)
	healthChecksTotal.WithLabelValues(n, t).Inc()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "type", t, "dependency", n, "err", err)
		healthChecksFailuresTotal.WithLabelValues(n, t).Inc()
		ch <- sqlResult{Name: n, Success: false, Duration: elapsed, Err: err}
	} else {
		ch <- sqlResult{Name: n, Success: true, Duration: elapsed}
	}
}

type sqlResult struct {
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	Err      error   `json:"error,omitempty"`
}

type sqlClientsList []sqlClient

type sqlClient struct {
	Name string
	Db   *sql.DB
	Type string
}
