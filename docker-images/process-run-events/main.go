package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const QUEUE_URL = "https://sqs.ca-central-1.amazonaws.com/253789223556/tfc-run-events"
const CACHE_MOUNPOINT = "/opt/tfc-cache"
const S3_BUCKET_TF_CONFIGS = "tfc-configuration-files"
const TF_CONFIG_DIRNAME = "/tf-config"

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

type S3Actions struct {
	S3Client *s3.Client
}

func newS3Actions() *S3Actions {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ca-central-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return &S3Actions{S3Client: s3.NewFromConfig(cfg)}
}

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

func (actor SqsActions) DeleteMessage(ctx context.Context, queueurl string, receipthandle string) {
	defer timeTrack(time.Now(), "delete-sqs-message")
	_, err := actor.SqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueurl),
		ReceiptHandle: &receipthandle,
	})
	if err != nil {
		log.Printf("couldn't delete message from queue %v. here's why: %v\n", queueurl, err)
	}
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
	log.Printf("Processing message with ID=%v", *msg.MessageId)

	// Unmarshall SQS message JSON payload
	runInputMsg, err := unmarshalRunInputMsg(*msg.Body)
	if err != nil {
		return err
	}
	configVersionId := runInputMsg.ConfigVersionId
	configVersionS3ObjectKey := runInputMsg.ConfigVersionS3ObjectKey

	log.Printf(
		"configVersionId=%v, configVersionS3ObjectKey=%v",
		configVersionId,
		configVersionS3ObjectKey,
	)

	// Fetch configuration version package
	downloadFilePath := getTfConfigDownloadFilePath(configVersionS3ObjectKey)
	err = s3Actions.downloadTfConfig(configVersionS3ObjectKey, S3_BUCKET_TF_CONFIGS, downloadFilePath)

	// Unzip terraform configuration
	unzipTfConfigPackage(downloadFilePath)

	// Set proper Terraform binary version
	switchTfVersion("1.9.7", true)

	// Run terraform init
	tfInit()

	return nil
}

func tfInit() {
	defer timeTrack(time.Now(), "tf-init")

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("terraform", "init", "-no-color")
	cmd.Dir = filepath.Join(dirname, TF_CONFIG_DIRNAME)
	stdout, err := cmd.Output()

	if err != nil {
		log.Println("Error while applying terraform init: " + err.Error())
	}
	// Print the output
	log.Println("Ouput of tf init: " + string(stdout))
}

func unzipTfConfigPackage(filepath string) {
	defer timeTrack(time.Now(), "unzip-tf-config")

	// create target directory if not existing
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	tfConfigDir := dirname + TF_CONFIG_DIRNAME
	_ = os.Mkdir(tfConfigDir, 0755)

	cmd := exec.Command("tar", "-xf", filepath, "--strip-components=1", "-C", tfConfigDir)
	_, err = cmd.Output()

	if err != nil {
		log.Println("Error while unzipping tf config package: " + err.Error())
	}
}

func getTfConfigDownloadFilePath(s3ObjectKey string) string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(dirname, s3ObjectKey)
}

func (actor S3Actions) downloadTfConfig(objectKey string, bucketName string, fileName string) error {
	defer timeTrack(time.Now(), "downloadTfConfig")

	result, err := actor.S3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		return err
	}
	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objectKey, err)
	}
	_, err = file.Write(body)
	return err
}

var sqsActions = newSqsActions()
var s3Actions = newS3Actions()

func main() {
	//listDir(CACHE_MOUNPOINT + "/terraform/.terraform.versions")

	for {
		messages, _ := sqsActions.GetMessages(context.TODO(), QUEUE_URL, 5, 10)
		if len(messages) > 0 {
			log.Printf("Fetched %d messages from queue", len(messages))
		}

		for _, msg := range messages {
			if err := processSqsMessage(msg); err != nil {
				log.Printf("Error when processing sqs msg: %v", err.Error())
			}
			sqsActions.DeleteMessage(context.TODO(), QUEUE_URL, *msg.ReceiptHandle)
		}
	}
}
