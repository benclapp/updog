package checks

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestHttpChecker(t *testing.T) {
	httpchecker := NewHttpChecker("google", "http://demo.robustperception.io:9090/-/healthy", 5*time.Second)
	result := httpchecker.Check()

	assert.Assert(t, result.Success == true)
	assert.Assert(t, result.Err == nil)
}
