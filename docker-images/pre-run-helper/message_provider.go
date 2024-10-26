package main

import "context"

type RunMessage struct {
	MessageId *string
	Body      *RunMessageBody
	// An identifier associated with the act of receiving the message. A new receipt
	// handle is returned every time you receive a message. When deleting a message,
	// you provide the last received receipt handle to delete the message.
	ReceiptHandle *string
	Attributes    map[string]string
}

type RunMessageBody struct {
	ConfigVersionId          string `json:"configVersionId"`
	ConfigVersionS3ObjectKey string `json:"configVersionS3ObjectKey"`
	WorkspaceId              string `json:"workspaceId"`
}

type MessageProvider interface {
	GetRunMessages(ctx context.Context) ([]RunMessage, error)
	DeleteMessage(ctx context.Context, receipthandle *string) error
	WithMaxMessages(int32)
	WithWaitTime(int32)
}
