package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	CACHE_MOUNPOINT      = "/opt/tfc-cache"
	TF_EXEC_PATH         = "/home/app/.bin/terraform"
	S3_BUCKET_TF_CONFIGS = "tfc-configuration-files"
	TF_CONFIG_DIRNAME    = "/tf-config"
	VARIABLES_TABLE      = "vars"
	VAR_CATEGORY_ENV     = "env"
	VAR_CATEGORY_TF      = "terraform"
)

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

type DynamoDBActions struct {
	DymamoDBClient *dynamodb.Client
}

func newDynamoDBActions() *DynamoDBActions {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("ca-central-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return &DynamoDBActions{DymamoDBClient: dynamodb.NewFromConfig(cfg)}
}

func (actor DynamoDBActions) GetVariables(ctx context.Context, wsId string, table string) ([]Variable, error) {
	defer timeTrack(time.Now(), "query-variables-dynamodb")
	var (
		err       error
		variables []Variable
		response  *dynamodb.QueryOutput
	)
	keyEx := expression.Key("workspace-id").Equal(expression.Value(wsId))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("Couldn't build expression to query variables from DynamoDB: %w\n", err)
	} else {
		queryPaginator := dynamodb.NewQueryPaginator(actor.DymamoDBClient, &dynamodb.QueryInput{
			TableName:                 aws.String(table),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		})
		for queryPaginator.HasMorePages() {
			response, err = queryPaginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("Couldn't query variables for workspaceId=%s: %w\n", wsId, err)
			} else {
				var variablePage []Variable
				err = attributevalue.UnmarshalListOfMaps(response.Items, &variablePage)
				if err != nil {
					return nil, fmt.Errorf("Couldn't unmarshal query response for workspaceId=%s: %w\n", wsId, err)
				} else {
					variables = append(variables, variablePage...)
				}
			}
		}
	}
	return variables, nil
}

type RunInputMsg struct {
	ConfigVersionId          string `json:"configVersionId"`
	ConfigVersionS3ObjectKey string `json:"configVersionS3ObjectKey"`
	WorkspaceId              string `json:"workspaceId"`
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

func processSqsMessage(msg Message) error {
	defer timeTrack(time.Now(), "process-sqs-msg")
	log.Printf("Processing message with ID=%v", *msg.MessageId)

	// Unmarshall SQS message JSON payload
	runInputMsg, err := unmarshalRunInputMsg(*msg.Body)
	if err != nil {
		return err
	}
	configVersionId := runInputMsg.ConfigVersionId
	configVersionS3ObjectKey := runInputMsg.ConfigVersionS3ObjectKey
	workspaceId := runInputMsg.WorkspaceId

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

	// Set workspace variables
	setWorkspaceVars(workspaceId)

	// Clean config files
	cleanConfig()

	return nil
}

type Variable struct {
	Id          string             `dynamodbav:"id"`
	WorkspaceId string             `dynamodbav:"workspace-id"`
	VarType     string             `dynamodbav:"type"`
	Attributes  VariableAttributes `dynamodbav:"attributes"`
}

type VariableAttributes struct {
	Key       string `dynamodbav:"key"`
	Value     string `dynamodbav:"value"`
	Category  string `dynamodbav:"category"`
	Sensitive bool   `dynamodbav:"sensitive"`
}

func setWorkspaceVars(wsId string) {
	defer timeTrack(time.Now(), "set-workspace-id")
	// get workspace variables
	dynamoDBActions := *newDynamoDBActions()
	vars, err := dynamoDBActions.GetVariables(context.TODO(), wsId, VARIABLES_TABLE)
	if err != nil {
		fmt.Printf("Could not retrieve variables from DynamoDB: %s\n", err.Error())
		return
	}

	fmt.Printf("Retrieved %d variables from DynamoDB\n", len(vars))

	// set environment variables
	for _, v := range vars {
		switch v.Attributes.Category {
		case VAR_CATEGORY_ENV:
			os.Setenv(v.Attributes.Key, v.Attributes.Value)
		case VAR_CATEGORY_TF:
			os.Setenv("TF_VAR_"+v.Attributes.Key, v.Attributes.Value)
		default:
			fmt.Printf("Unexpected variable category='%v'.", v.Attributes.Category)
		}
	}

	return
}

func cleanConfig() {
	defer timeTrack(time.Now(), "cleanConfig")
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll(filepath.Join(dirname, TF_CONFIG_DIRNAME))
	if err != nil {
		log.Println("Error while deleting all files of tf config: " + err.Error())
	}
}

func mustGetTFConfigDir() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(dirname, TF_CONFIG_DIRNAME)
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

var s3Actions = newS3Actions()

// Declare variables
var cfg aws.Config
var sqsMessageProvider SqsMessageProvider

// Configure providers
func init() {
	cfg := mustLoadAwsConfig(context.TODO())

	sqsMessageProvider := newSqsMessageProvider(cfg)
	sqsMessageProvider.WithMaxMessages(5)
	sqsMessageProvider.WithWaitTime(10) // 10 seconds
}

func main() {
	prerun_helper(sqsMessageProvider)
}

func prerun_helper(msg_p MessageProvider) {
	for {
		messages, _ := msg_p.GetRunMessages(context.TODO())
		if len(messages) > 0 {
			log.Printf("Fetched %d messages from queue", len(messages))
		}

		for _, msg := range messages {
			if err := processSqsMessage(msg); err != nil {
				log.Printf("Error when processing sqs msg: %v", err.Error())
			}
			msg_p.DeleteMessage(context.TODO(), msg.ReceiptHandle)
		}
	}
}

func mustLoadAwsConfig(ctx context.Context) aws.Config {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to load SDK config, %v", err)
	}
	return cfg
}
