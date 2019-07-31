package checks

import (
	"database/sql"
	"time"

	"github.com/go-kit/kit/log/level"

	_ "github.com/denisenkom/go-mssqldb" // MSSQL
	_ "github.com/lib/pq"                // Postgres
)

const SQL_TYPE = "sql"

type SqlChecker struct {
	name       string
	driverName string
	db         *sql.DB
}

func NewSqlChecker(name, driverName, dataSourceName string) *SqlChecker {
	conn, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		level.Error(logger).Log("Opening db connection failed:", err.Error())
	}
	return &SqlChecker{name: name, driverName: driverName, db: conn}
}

func (receiver SqlChecker) Check() Result {
	start := time.Now()
	err := receiver.db.Ping()
	elapsed := time.Since(start).Seconds()

	if err != nil {
		logger.Log("msg", "Error while checking dependency", "type", receiver.driverName, "dependency", receiver.name, "err", err)
		return Result{Name: receiver.name, Typez: SQL_TYPE + receiver.driverName, Duration: elapsed, Success: false}
	} else {
		return Result{Name: receiver.name, Typez: SQL_TYPE + receiver.driverName, Duration: elapsed, Success: true}
	}
}
