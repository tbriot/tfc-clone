package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	QUEUE_URL string = "https://sqs.ca-central-1.amazonaws.com/253789223556/tfc-run-events"
)

type SqsMessageProvider struct {
	sqsClient   *sqs.Client
	queueUrl    *string
	maxMessages int32
	waitTime    int32 // in seconds
}

// Constructor
func newSqsMessageProvider(cfg aws.Config) *SqsMessageProvider {
	queueUrl := QUEUE_URL // create addressable variable from unadressable constant
	return &SqsMessageProvider{
		sqsClient:   sqs.NewFromConfig(cfg),
		queueUrl:    &queueUrl,
		maxMessages: 5,
		waitTime:    10, // 10 seconds
	}
}

// Setters
func (p SqsMessageProvider) WithMaxMessages(no_msg int32) {
	p.maxMessages = no_msg
}

func (p SqsMessageProvider) WithWaitTime(time int32) {
	p.waitTime = time
}

// Implementing the MessageProvider interface
func (p SqsMessageProvider) GetRunMessages(ctx context.Context) ([]Message, error) {
	var messages []types.Message
	result, err := p.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(*p.queueUrl),
		MaxNumberOfMessages: p.maxMessages,
		WaitTimeSeconds:     p.waitTime,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get messages from queue %v: %w\n", p.queueUrl, err)
	} else {
		messages = result.Messages
	}

	return mapSqsMessages(messages), err
}

func (p SqsMessageProvider) DeleteMessage(ctx context.Context, receipthandle *string) error {
	defer timeTrack(time.Now(), "delete-sqs-message")
	_, err := p.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(*p.queueUrl),
		ReceiptHandle: receipthandle,
	})
	if err != nil {
		return fmt.Errorf("couldn't delete message from queue %v: %w\n", p.queueUrl, err)
	}
	return nil
}

// Utility function that Converts SQS messages into abstracted messages type
func mapSqsMessages(sqsMsgs []types.Message) []Message {
	var msgs []Message
	for _, m := range sqsMsgs {
		msgs = append(msgs, Message{
			MessageId:     m.MessageId,
			Body:          m.Body,
			ReceiptHandle: m.ReceiptHandle,
			Attributes:    m.Attributes,
		})
	}
	return msgs
}
