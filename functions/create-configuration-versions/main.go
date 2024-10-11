package main

import (
	"bytes"
	"context"
	//"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	tfe "github.com/hashicorp/go-tfe"

	"github.com/google/jsonapi"
	"github.com/rs/xid"
)

type ConfigurationVersion struct {
	Id   string `dynamodbav:"id"`
	Type string `dynamodbav:"type"`
}

func UnmarshalRequestPayload(json *string) (*tfe.ConfigurationVersionCreateOptions, error) {
	configurationVersionCreateOptions := new(tfe.ConfigurationVersionCreateOptions)
	jsonStringReader := strings.NewReader(*json)

	if err := jsonapi.UnmarshalPayload(jsonStringReader, configurationVersionCreateOptions); err != nil {
		parsingError := fmt.Errorf("Error while parsing request json payload: %w", err)
		return configurationVersionCreateOptions, parsingError
	}

	return configurationVersionCreateOptions, nil
}

func MarshalResponsePayload(cv *tfe.ConfigurationVersion) (*string, error) {
	jsonBuf := bytes.NewBufferString("")
	if err := jsonapi.MarshalPayload(jsonBuf, cv); err != nil {
		log.Fatalf("unable to marshall new item to json response, %v", err)
	}
	json := jsonBuf.String()
	return &json, nil
}

type Presigner struct {
	PresignClient *s3.PresignClient
}

func (presigner Presigner) PutObject(
	ctx context.Context, bucketName string, objectKey string, lifetimeSecs int64) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		log.Printf("Couldn't get a presigned request to put %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return request, err
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("Processing request data for request %s.\n", request.RequestContext.RequestID)
	fmt.Printf("Body size = %d.\n", len(request.Body))

	// Umarshall request payload
	configurationVersionCreateOptions, err := UnmarshalRequestPayload(&request.Body)
	if err != nil {
		fmt.Print(err.Error())
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
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

	// Create S3 presign client
	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)
	presigner := Presigner{PresignClient: presignClient}

	ID := "cv-" + xid.New().String()

	// Build S3 object presigned upload url
	msec := time.Now().UnixMilli()
	s3ObjectKey := fmt.Sprintf("%s-%d.tar.gz", ID, msec)
	presignedPutRequest, err := presigner.PutObject(ctx, "tfc-configuration-files", s3ObjectKey, 15*60)
	if err != nil {
		panic(err)
	}

	newConfigurationVersion := tfe.ConfigurationVersion{
		ID:            ID,
		AutoQueueRuns: *configurationVersionCreateOptions.AutoQueueRuns,
		Source:        tfe.ConfigurationSourceAPI,
		Status:        tfe.ConfigurationPending,
		UploadURL:     presignedPutRequest.URL,
	}

	item, err := attributevalue.MarshalMap(newConfigurationVersion)
	if err != nil {
		panic(err)
	}

	// Create the DynamoDB client
	svc := dynamodb.NewFromConfig(cfg)

	// Build the request with its input parameters
	_, err = svc.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String("configuration-versions"),
		Item:      item,
	})
	if err != nil {
		log.Fatalf("failed to put item, %v", err)
	}

	// Unmarshall response
	json, err := MarshalResponsePayload(&newConfigurationVersion)
	if err != nil {
		log.Fatalf("failed to marshall response payload, %v", err)
	}

	return events.APIGatewayProxyResponse{Body: *json, StatusCode: 200}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
