package v1

import "time"

type BinaryMessage = Message[[]byte]
type TextMessage = Message[string]

type MessageContents interface {
	[]byte | string
}

type Message[T MessageContents] struct {
	Content    T
	ReceivedAt time.Time
}
