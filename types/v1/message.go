package v1

type BinaryMessage = Message[[]byte]
type TextMessage = Message[string]

type MessageContents interface {
	[]byte | string
}

type Message[T MessageContents] struct {
	Type    int
	Payload T
}
