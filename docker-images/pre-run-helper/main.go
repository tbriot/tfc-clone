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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	S3_BUCKET_TF_CONFIGS   string = "tfc-configuration-files"
	TF_CONFIG_REL_DIR_PATH string = "/tf-config"

	CACHE_MOUNPOINT  = "/opt/tfc-cache"
	TF_EXEC_PATH     = "/home/app/.bin/terraform"
	VARIABLES_TABLE  = "vars"
	VAR_CATEGORY_ENV = "env"
	VAR_CATEGORY_TF  = "terraform"
)

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

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %d ms", name, elapsed.Milliseconds())
}

func processSqsMessage(msg RunMessage) error {
	defer timeTrack(time.Now(), "process-sqs-msg")
	log.Printf("Processing message with ID=%v", *msg.MessageId)

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

// Declare variables
var cfg aws.Config
var sqsMessageProvider SqsMessageProvider
var s3TfConfigProvider S3TfConfigProvider

// Configure providers
func init() {
	cfg := mustLoadAwsConfig(context.TODO())

	sqsMessageProvider := newSqsMessageProvider(cfg)
	sqsMessageProvider.WithMaxMessages(5)
	sqsMessageProvider.WithWaitTime(10) // 10 seconds

	s3TfConfigProvider := newS3TfConfigProvider(cfg)
	_ = s3TfConfigProvider
}

func main() {
	prerun_helper(sqsMessageProvider)
}

func prerun_helper(msg_p MessageProvider) {
	for {
		messages, err := msg_p.GetRunMessages(context.TODO())
		if err != nil {
		}
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

// Returns directory where the TF config is made available
func getTfConfigInstallPath() (dirPath string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to get user home directory, %v", err)
	}
	return filepath.Join(homeDir, TF_CONFIG_REL_DIR_PATH), nil
}
