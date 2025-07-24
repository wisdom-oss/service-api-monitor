package v1

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Command struct {
	Command string `json:"command" validate:"required"`
	Data    any    `json:"data"`
}

const (
	commandSubscribe       = "subscribe"
	commandUnsubscribe     = "unsubscribe"
	commandAddSubscription = "addSubscription"
)

func (c Command) Valid() (bool, []error) {
	err := validate.Struct(c)
	if err != nil {

	}
	return true, nil
}
