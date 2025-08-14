package v1

type CommandError struct {
	IncomingMessageID string `json:"relatedTo,omitempty"`
	Error             string `json:"error"`
	IncomingData      any    `json:"receivedData,omitempty"`
}
