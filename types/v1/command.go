package v1

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
)

type Command struct {
	Command string          `json:"command" validate:"required,gt=0"`
	ID      string          `json:"id"`
	Data    json.RawMessage `json:"data"    validate:"required"`
}

func (c Command) Validate() error {
	v := validator.New()
	return v.Struct(c)
}
