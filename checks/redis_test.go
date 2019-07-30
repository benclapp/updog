package checks

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestRedisChecker(t *testing.T) {
	redischecker := NewRedisChecker("docker_redis", "127.0.0.1:6379", "", 5*time.Second, false)
	result := redischecker.Check()

	assert.Assert(t, result.Success == true)
	assert.Assert(t, result.Err == nil)
}
