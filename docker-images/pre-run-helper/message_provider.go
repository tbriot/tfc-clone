package main

import "context"

type Message struct {
	MessageId *string
	Body      *string
	// An identifier associated with the act of receiving the message. A new receipt
	// handle is returned every time you receive a message. When deleting a message,
	// you provide the last received receipt handle to delete the message.
	ReceiptHandle *string
	Attributes    map[string]string
}

type MessageProvider interface {
	GetRunMessages(ctx context.Context) ([]Message, error)
	DeleteMessage(ctx context.Context, receipthandle *string) error
	WithMaxMessages(int32)
	WithWaitTime(int32)
}
