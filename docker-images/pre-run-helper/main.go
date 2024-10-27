package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	S3_BUCKET_TF_CONFIGS           string = "tfc-configuration-files"
	TF_CONFIG_REL_DIR_PATH         string = "/tf-config"
	RUN_MESSAGE_PROVIDER_WAIT_TIME int32  = 5 // in seconds

	CACHE_MOUNPOINT  = "/opt/tfc-cache"
	VAR_CATEGORY_ENV = "env"
	VAR_CATEGORY_TF  = "terraform"
)

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms", name, elapsed.Milliseconds())
}

func processSqsMessage(msg RunMessage) error {
	defer timeTrack(time.Now(), "process-sqs-msg")
	log.Printf("Processing message with ID=%v", *msg.MessageId)

	return nil
}

//func setWorkspaceVars(wsId string) {
//	defer timeTrack(time.Now(), "set-workspace-id")
//	// get workspace variables
//	dynamoDBActions := *newDynamoDBActions()
//	vars, err := dynamoDBActions.GetVariables(context.TODO(), wsId, VARIABLES_TABLE)
//	if err != nil {
//		fmt.Printf("Could not retrieve variables from DynamoDB: %s\n", err.Error())
//		return
//	}
//
//	fmt.Printf("Retrieved %d variables from DynamoDB\n", len(vars))
//
//	// set environment variables
//	for _, v := range vars {
//		switch v.Attributes.Category {
//		case VAR_CATEGORY_ENV:
//			os.Setenv(v.Attributes.Key, v.Attributes.Value)
//		case VAR_CATEGORY_TF:
//			os.Setenv("TF_VAR_"+v.Attributes.Key, v.Attributes.Value)
//		default:
//			fmt.Printf("Unexpected variable category='%v'.", v.Attributes.Category)
//		}
//	}
//
//	return
//}

// Declare variables
var cfg aws.Config
var sqsMessageProvider *SqsMessageProvider
var s3TfConfigProvider *S3TfConfigProvider
var tfcWorkspaceVariablesClient *TfcWorkspaceVariablesClient

// Configure providers
func init() {
	cfg := mustLoadAwsConfig(context.TODO())

	// Run message provider
	sqsMessageProvider := newSqsMessageProvider(cfg)
	sqsMessageProvider.WithMaxMessages(1) //  retrieve only one message
	sqsMessageProvider.WithWaitTime(RUN_MESSAGE_PROVIDER_WAIT_TIME)

	// Terraform configuration provider
	s3TfConfigProvider := newS3TfConfigProvider(cfg)
	_ = s3TfConfigProvider

	// Workspace provider
	tfcWorkspaceVariablesClient := newTfcWorkspaceVariablesClient(cfg)
	_ = tfcWorkspaceVariablesClient
}

func main() {
	// Inject dependencies
	prerun_helper(*sqsMessageProvider, *s3TfConfigProvider, *tfcWorkspaceVariablesClient)
}

func prerun_helper(msgProvider MessageProvider, configProvider TfConfigProvider, wsVarsManager TfcWorkspaceVariablesManager) {
	tfConfigInstallPath := mustGetTfConfigInstallPath()

	for {
		messages, err := msgProvider.GetRunMessages(context.TODO())
		if err != nil {
			// If critial then panic
			// else continue
		}

		switch len(messages) {
		case 0:
			fmt.Printf("No run message received after %d wait. Continuing to poll...\n", RUN_MESSAGE_PROVIDER_WAIT_TIME)
		case 1:
			fmt.Println("Received one run message.")
		default:
			log.Fatalf("Fetched %d messages from queue while expecting only one.", len(messages))
		}

		// Only one message is consumed by the prerun_helper
		msg := messages[0]
		log.Printf("Message information: ID=%v, workspace-id=%v, configuration-version-id=%v.",
			*msg.MessageId,
			msg.Body.WorkspaceId,
			msg.Body.ConfigVersionId)

		// Download and extract TF config
		configProvider.DownloadTfConfig(context.TODO(), msg.Body.ConfigVersionId, tfConfigInstallPath)

		// Get TFC Workspace Vars
		// Build list of env vars from ws vars
		// Write .env file
		_ = wsVarsManager

		msgProvider.DeleteMessage(context.TODO(), msg.ReceiptHandle)
	}
}

func mustLoadAwsConfig(ctx context.Context) aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	return cfg
}

// Returns directory where the TF config is made available
func mustGetTfConfigInstallPath() (dirPath string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to get user home directory, %v", err)
	}
	return filepath.Join(homeDir, TF_CONFIG_REL_DIR_PATH)
}
