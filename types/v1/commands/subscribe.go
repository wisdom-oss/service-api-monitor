package commands

import (
	"github.com/go-playground/validator/v10"
	"github.com/sosodev/duration"
)

type Subscribe struct {
	Paths    []string          `json:"paths"          validate:"required,dive,gt=0"`
	Interval duration.Duration `json:"updateInterval"`
}

func (s Subscribe) Validate() error {
	v := validator.New()
	return v.Struct(s)
}
