package types

import (
	"google.golang.org/protobuf/proto"
)

type Message[T proto.Message] struct {
	Key     *string
	Value   T
	Headers map[string]string
	Ack     func() error
}

type Consumer[T proto.Message] interface {
	Open() (<-chan *Message[T], error)
	Close() error
}

type Publisher[T proto.Message] interface {
	Open() (chan<- *Message[T], error)
	Close() error
}
