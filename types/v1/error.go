package v1

import "encoding/json"

type CommandError struct {
	IncomingMessageID any
	Error             error
	IncomingCommand   Command
}

func (ce CommandError) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m["type"] = "error"
	m["relatedTo"] = ce.IncomingMessageID
	m["error"] = ce.Error.Error()
	m["receivedCommand"] = ce.IncomingCommand

	return json.Marshal(m)
}
