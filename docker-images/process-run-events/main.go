package main

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const QUEUE_URL = "https://sqs.ca-central-1.amazonaws.com/253789223556/tfc-run-events"
const CACHE_MOUNPOINT = "/opt/tfc-cache"

// SqsActions encapsulates the Amazon Simple Queue Service (Amazon SQS) actions
type SqsActions struct {
	SqsClient *sqs.Client
}

func newSqsActions() *SqsActions {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ca-central-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return &SqsActions{SqsClient: sqs.NewFromConfig(cfg)}
}

// GetMessages uses the ReceiveMessage action to get messages from an Amazon SQS queue.
func (actor SqsActions) GetMessages(ctx context.Context, queueUrl string, maxMessages int32, waitTime int32) ([]types.Message, error) {
	var messages []types.Message
	result, err := actor.SqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueUrl),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitTime,
	})
	if err != nil {
		log.Printf("Couldn't get messages from queue %v. Here's why: %v\n", queueUrl, err)
	} else {
		messages = result.Messages
	}
	return messages, err
}

func switchTfVersion(version string, cache bool) {
	var cmd *exec.Cmd
	if cache {
		cmd = exec.Command("tfswitch", "-i", CACHE_MOUNPOINT+"/terraform", version)
	} else {
		cmd = exec.Command("tfswitch", version)
	}
	stdout, err := cmd.Output()

	if err != nil {
		log.Println("tfswitch. error:" + err.Error())
	}

	log.Println("tfswitch. stdout:" + string(stdout))
}

func displayTfVersion() {
	app := "terraform"
	arg0 := "-version"

	cmd := exec.Command(app, arg0)
	stdout, err := cmd.Output()

	if err != nil {
		log.Println(err.Error())
	}

	// Print the output
	log.Println("displayTfVersion:" + string(stdout))
}

func listDir(path string) {
	log.Printf("Listing content of dir=%v", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		log.Println(e.Name())
	}
}

func main() {
	listDir(CACHE_MOUNPOINT + "/terraform/.terraform.versions")
	// tfswitch to 1.9.7 using custom install dir
	switchTfVersion("1.9.7", true)
	displayTfVersion()
	// tfswitch to 1.9.7 using custom install dir
	switchTfVersion("1.9.7", false)
	displayTfVersion()
	// tfswitch to 1.8.5 custom install dir
	switchTfVersion("1.8.5", true)
	displayTfVersion()
	// tfswitch to 1.8.5 custom install dir
	switchTfVersion("1.8.5", false)
	displayTfVersion()
	// tfswitch to 1.8.3 custom install dir
	switchTfVersion("1.8.3", true)
	displayTfVersion()
	// tfswitch to 1.8.3 custom install dir
	switchTfVersion("1.8.3", false)
	displayTfVersion()

	sqsActions := newSqsActions()
	for {
		messages, _ := sqsActions.GetMessages(context.TODO(), QUEUE_URL, 2, 5)

		log.Printf("Received %d messages", len(messages))
		for _, msg := range messages {
			log.Printf("Received message with ID=%v, Body=%v", *msg.MessageId, *msg.Body)
		}
	}
}
