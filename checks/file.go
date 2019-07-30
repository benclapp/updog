package checks

import (
	"os"
)

//type created to make the func implement check interface
type CheckFunc func() Result

func (cf CheckFunc) Check() Result {
	return cf()
}

func NewFileChecker(filename string) Checker {
	return CheckFunc(func() Result {
		if _, err := os.Stat(filename); err != nil {
			return Result{Name: "file", Err: err, Success: false}
		}
		return Result{Name: "file", Err: nil, Success: true}
	})
}
