package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/denisenkom/go-mssqldb" // MSSQL
	_ "github.com/lib/pq"                // Postgres
)

func initSQL() {
	for _, db := range config.Dependencies.SQL {

		logger.Log("dependency_type", "sql", "dependency_name", string(db.Name), "db_type", db.Type)

		healthCheckDependencyDuration.WithLabelValues(db.Name, db.Type).Observe(0)
		healthChecksTotal.WithLabelValues(db.Name, db.Type).Add(0)
		healthChecksFailuresTotal.WithLabelValues(db.Name, db.Type).Add(0)

		// checkSQL(db.Name, db.Type, db.ConnectionString)
	}
}

func checkSQL(n, t, cs string, ch chan<- sqlResult) {
	start := time.Now()
	db, err := sql.Open(t, cs)
	if err != nil {
		log.Fatal("Opening db connection failed:", err.Error())
	}
	defer db.Close()

	stmt, err := db.Prepare("SELECT 1")
	if err != nil {
		log.Fatal("prepare failed:", err.Error())
	}
	defer stmt.Close()

	row := stmt.QueryRow()
	var number int64
	err = row.Scan(&number)
	if err != nil {
		log.Fatal("Scan failed:", err.Error())
	}

	elapsed := time.Since(start).Seconds()

	healthCheckDependencyDuration.WithLabelValues(n, t).Observe(elapsed)
	healthChecksTotal.WithLabelValues(n, t).Inc()

	if err != nil {
		logger.Log("msg", "Errpr while checking dependency", "type", t, "dependency", n, "err", err)
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
