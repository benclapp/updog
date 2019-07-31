package checks

import (
	"fmt"
	"log"
	"testing"

	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

func TestFileChecker(t *testing.T) {
	var filechecker Checker
	filechecker = NewFileChecker("banana.txt")
	result := filechecker.Check()
	assert.Assert(t, result.Err != nil)
}

func TestMarshalResult(t *testing.T) {
	var result = Result{Name: "name", Success: true, Duration: 2.0, Reason: "200"}
	d, err := yaml.Marshal(&result)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- t dump:\n%s\n\n", string(d))
	assert.Assert(t, err == nil)
}
