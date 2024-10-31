package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	S3_BUCKET_TF_CONFIGS string = "tfc-configuration-files"

	// Define where terraform config and dotenv file are written on the disk
	SHARED_VOLUME_MOUNT_PATH string = "/opt/shared-volume"
	TF_CONFIG_REL_DIR_PATH   string = "/tfconfig"
	DOTENV_REL_FILE_PATH     string = "/.env"

	RUN_MESSAGE_PROVIDER_WAIT_TIME int32 = 5 // in seconds

	// Workspace variables categories
	VAR_CATEGORY_ENV = "env"
	VAR_CATEGORY_TF  = "terraform"
)

func main() {
	cfg := mustLoadAwsConfig(context.TODO())

	// Run message provider
	sqsMessageProvider := newSqsMessageProvider(cfg)
	sqsMessageProvider.WithMaxMessages(1) //  retrieve only one message
	sqsMessageProvider.WithWaitTime(RUN_MESSAGE_PROVIDER_WAIT_TIME)

	// Terraform configuration provider
	s3TfConfigProvider := newS3TfConfigProvider(cfg)

	// Workspace provider
	tfcWorkspaceVariablesClient := newTfcWorkspaceVariablesClient(cfg)
	// Inject dependencies
	prerun_helper(sqsMessageProvider, s3TfConfigProvider, tfcWorkspaceVariablesClient)
}

func prerun_helper(msgProvider MessageProvider, configProvider TfConfigProvider, wsVarsManager TfcWorkspaceVariablesManager) {
	for {
		// Retrieve one run message
		messages, err := msgProvider.GetRunMessages(context.TODO())
		if err != nil {
			// TODO: if error is NOT critial then log and continue
			log.Fatalf("could not retrieve run message: %v", err)
		}

		switch len(messages) {
		case 0:
			fmt.Printf("No run message received after %d sec wait. Continuing to poll...\n", RUN_MESSAGE_PROVIDER_WAIT_TIME)
			// continue to the next iteraton of the loop
			continue
		case 1:
			fmt.Println("Received one run message!")
		default:
			log.Fatalf("Fetched %d messages from queue while expecting only one.", len(messages))
		}

		defer timeTrack(time.Now(), "prerun-helper-process-msg")
		// Only one message is consumed by the prerun_helper
		msg := messages[0]
		fmt.Printf("Message information: ID=%v, workspace-id=%v, configuration-version-id=%v.\n",
			*msg.MessageId,
			msg.Body.WorkspaceId,
			msg.Body.ConfigVersionId)

		// Download and extract TF config
		err = configProvider.DownloadTfConfig(
			context.TODO(),
			// Passing an S3 object key, but should instead really pass a key id
			// to abstract the s3 technical implementation
			msg.Body.ConfigVersionS3ObjectKey,
			SHARED_VOLUME_MOUNT_PATH+TF_CONFIG_REL_DIR_PATH)
		if err != nil {
			log.Fatalf("Could not download tf config version id=%v: %v", msg.Body.ConfigVersionId, err)
		}

		// Get TFC Workspace Vars
		workspaceVariables, err := wsVarsManager.ListWorkspaceVariables(msg.Body.WorkspaceId)
		if err != nil {
			log.Fatalf("Could not retrieve workspace variables, workspace-id=%v: %v", msg.Body.WorkspaceId, err)
		}

		// Get list of environment variables to be loaded when we run terraform
		// commands at a later stage
		envVariables := getEnvVariablesList(*workspaceVariables)
		// Write .env file
		mustWriteVarsToDotenvFile(
			envVariables,
			SHARED_VOLUME_MOUNT_PATH+DOTENV_REL_FILE_PATH)

		msgProvider.DeleteMessage(context.TODO(), msg.ReceiptHandle)

		// break out of the loop and return from the enclosing function
		return
	}
}

func mustLoadAwsConfig(ctx context.Context) aws.Config {
	defer timeTrack(time.Now(), "load-aws-config")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	return cfg
}

// Return list of environment variables to be loaded in terraform context
func getEnvVariablesList(vars []WorkspaceVariable) map[string]string {
	m := make(map[string]string)
	for _, v := range vars {
		switch v.Category {
		case VAR_CATEGORY_ENV:
			m[v.Key] = v.Value
		case VAR_CATEGORY_TF:
			// Terraform variable must be prefixed by "TF_VAR_"
			m[v.Key] = fmt.Sprintf("TF_VAR_%v", v.Value)
		default:
			fmt.Printf(
				"Unexpected workspace variable category, expecting 'env' or 'terraform', "+
					"got %v, key=%v, skipping variable.\n", v.Category, v.Key)
		}
	}
	return m
}

// Write environment variables in .env file
func mustWriteVarsToDotenvFile(vars map[string]string, filepath string) {
	defer timeTrack(time.Now(), "write-dotenv-file")
	var dotenvString string
	for k, v := range vars {
		dotenvString += fmt.Sprintf("%v=%v\n", k, v)
	}
	// only the owner of the file can read it
	err := os.WriteFile(filepath, []byte(dotenvString), 0400)
	if err != nil {
		log.Fatalf("Could not write env variables to file: %v.\n", err)
	}
}

// Utility to time execution time
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms.\n", name, elapsed.Milliseconds())
}
