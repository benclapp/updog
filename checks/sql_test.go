package checks

import (
	"testing"

	"gotest.tools/assert"
)

func TestSqlServerChecker(t *testing.T) {
	sqlchecker := NewSqlChecker("sampledb", "sqlserver", "sqlserver://username:password@host:port/instance&connection+timeout=30")
	result := sqlchecker.Check()

	assert.Assert(t, result.Success == true)
	assert.Assert(t, result.Err == nil)
}
