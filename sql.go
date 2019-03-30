package main

var mssqlClients = mssqlClientList{}

func initMSSQL() {
	// logger.Log("msg", "init mssql")
	for _, db := range config.Dependencies.MSSQL {


		// logger.Log("dependency_type", "mssql", "dependency_name", string(db.Name))

		healthCheckDependencyDuration.WithLabelValues(db.Name, "mssql").Observe(0)
		healthChecksTotal.WithLabelValues(db.Name, "mssql").Add(0)
		healthChecksFailuresTotal.WithLabelValues(db.Name, "mssql").Add(0)
	}
}

func checkMSSQL(n, cs string, ch, chan<- MSSQLResult) {
	
}

type MSSQLResult struct {
	Name     string  `json:"name"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	Err      error   `json:"error,omitempty"`
}

type mssqlClientList []struct {
	name    string
	connStr string
}
