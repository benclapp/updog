package checks

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var logger log.Logger = level.NewFilter(log.NewLogfmtLogger(os.Stdout), level.AllowInfo())

// Checker is a interface used to provide an indication of application health.
type Checker interface {
	Check() Result
}

type Result struct {
	Name     string  `json:"-"`
	Typez    string  `json:"type"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	Err      error   `json:"error,omitempty"`
	Reason   interface{}
}
