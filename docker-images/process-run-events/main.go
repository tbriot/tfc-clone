package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"

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
	defer timeTrack(time.Now(), "switchtfversion")
	var cmd *exec.Cmd
	if cache {
		cmd = exec.Command("tfswitch", "-i", CACHE_MOUNPOINT+"/terraform", version)
	} else {
		cmd = exec.Command("tfswitch", version)
	}
	_, err := cmd.Output()

	if err != nil {
		log.Println("tfswitch. error:" + err.Error())
	}
}

//func displayTfVersion() {
//	app := "terraform"
//	arg0 := "-version"
//
//	cmd := exec.Command(app, arg0)
//	stdout, err := cmd.Output()
//
//	if err != nil {
//		log.Println(err.Error())
//	}
//
//	// Print the output
//	log.Println("displayTfVersion:" + string(stdout))
//}

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

type RunInputMsg struct {
	ConfigVersionId          string `json:"configVersionId"`
	ConfigVersionS3ObjectKey string `json:"configVersionS3ObjectKey"`
}

func unmarshalRunInputMsg(payload string) (RunInputMsg, error) {
	defer timeTrack(time.Now(), "unmarshal-msg-payload")
	var runInputMsg RunInputMsg
	//if err := json.Unmarshal([]byte(payload), &runInputMsg); err =! nil {
	err := json.Unmarshal([]byte(payload), &runInputMsg)
	if err != nil {
		panic(err)
	}
	return runInputMsg, nil
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms", name, elapsed.Milliseconds())
}

func processSqsMessage(msg types.Message) error {
	defer timeTrack(time.Now(), "process-sqs-msg")
	log.Printf("Processing message with ID=%v, Body=%v", *msg.MessageId, *msg.Body)

	// Unmarshall SQS message JSON payload
	runInputMsg, err := unmarshalRunInputMsg(*msg.Body)
	if err != nil {
		return err
	}
	log.Printf(
		"configVersionId=%v, configVersionS3ObjectKey=%v",
		runInputMsg.ConfigVersionId,
		runInputMsg.ConfigVersionS3ObjectKey,
	)

	// Fetch configuration version package
	// TODO

	// Set proper Terraform binary version
	switchTfVersion("1.9.7", true)

	return nil
}

func main() {
	//listDir(CACHE_MOUNPOINT + "/terraform/.terraform.versions")

	sqsActions := newSqsActions()
	for {
		messages, _ := sqsActions.GetMessages(context.TODO(), QUEUE_URL, 5, 10)
		if len(messages) > 0 {
			log.Printf("Fetched %d messages from queue", len(messages))
		}

		for _, msg := range messages {
			if err := processSqsMessage(msg); err != nil {
				log.Printf("Error when processing sqs msg: %v", err.Error())
			}
		}
	}
}
