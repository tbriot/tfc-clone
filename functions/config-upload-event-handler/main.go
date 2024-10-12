package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// input: "cv-cs1f56el089s714shag0-1728246425601.tar.gz"
// output: "cv-cs1f56el089s714shag0"
func ExtractConfigVerionId(s3ObjectKey string) (string, error) {
	lastHyphenPos := strings.LastIndex(s3ObjectKey, "-")
	if lastHyphenPos == -1 {
		return "", errors.New("Could not extract configuration version id from s3 object key=" + s3ObjectKey)
	}
	return s3ObjectKey[:lastHyphenPos], nil
}

func HandleRequest(ctx context.Context, s3Event events.S3Event) {
	for _, record := range s3Event.Records {
		s3 := record.S3
		fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3.Bucket.Name, s3.Object.Key)

		configurationVersionId, err := ExtractConfigVerionId(s3.Object.Key)
		if err != nil {
			panic(err)
		}

		// Using the SDK's default configuration, loading additional config
		// and credentials values from the environment variables, shared
		// credentials, and shared configuration files
		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithRegion("ca-central-1"),
		)
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		// Create the DynamoDB client
		svc := dynamodb.NewFromConfig(cfg)

		// Marshall DynamoDB primary key
		// Returns {"S": "cv-cs1f56el089s714shag0"}
		mConfigurationVersionId, err := attributevalue.Marshal(configurationVersionId)
		if err != nil {
			panic(err)
		}

		// Build DynamoDB update expression
		update := expression.Set(expression.Name("Status"), expression.Value("uploaded"))
		expr, err := expression.NewBuilder().WithUpdate(update).Build()
		if err != nil {
			log.Fatalf("Couldn't build expression for update. Here's why: %v\n", err)
		}

		_, err = svc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName:                 aws.String("configuration-versions"),
			Key:                       map[string]types.AttributeValue{"ID": mConfigurationVersionId},
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			UpdateExpression:          expr.Update(),
		})
		if err != nil {
			log.Fatalf("Couldn't update configuration version %v. Here's why: %v\n", configurationVersionId, err)
		}

		log.Printf("Uploaded successfully config version=%v status to 'uploaded'.", configurationVersionId)
	}
}

func main() {
	lambda.Start(HandleRequest)
}
