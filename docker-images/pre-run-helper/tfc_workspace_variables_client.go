package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	DYNAMODB_WORKSPACE_VARIABLES_TABLE_NAME = "vars"
)

type TfcWorkspaceVariablesClient struct {
	ctx            context.Context
	dynamodbClient *dynamodb.Client
	tableName      string
}

type dynamodbVariable struct {
	Id          string                     `dynamodbav:"id"`
	WorkspaceId string                     `dynamodbav:"workspace-id"`
	VarType     string                     `dynamodbav:"type"`
	Attributes  dynamodbVariableAttributes `dynamodbav:"attributes"`
}

type dynamodbVariableAttributes struct {
	Key       string `dynamodbav:"key"`
	Value     string `dynamodbav:"value"`
	Category  string `dynamodbav:"category"`
	Sensitive bool   `dynamodbav:"sensitive"`
}

// --------------------------------------------------------------------------------
// Constructor
// --------------------------------------------------------------------------------
func newTfcWorkspaceVariablesClient(cfg aws.Config) *TfcWorkspaceVariablesClient {
	return &TfcWorkspaceVariablesClient{
		ctx:            context.TODO(),
		dynamodbClient: dynamodb.NewFromConfig(cfg),
		tableName:      DYNAMODB_WORKSPACE_VARIABLES_TABLE_NAME,
	}
}

// --------------------------------------------------------------------------------
// Implement TfcWorkspaceVariablesManager interface
// --------------------------------------------------------------------------------
func (c *TfcWorkspaceVariablesClient) ListWorkspaceVariables(wsId string) (*[]WorkspaceVariable, error) {
	defer timeTrack(time.Now(), "list-workspace-variables")
	var (
		err               error
		dynamodbVariables []dynamodbVariable
		response          *dynamodb.QueryOutput
	)

	keyEx := expression.Key("workspace-id").Equal(expression.Value(wsId))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, fmt.Errorf("couldn't build expression to query variables from DynamoDB: %w\n", err)
	} else {
		queryPaginator := dynamodb.NewQueryPaginator(c.dynamodbClient, &dynamodb.QueryInput{
			TableName:                 aws.String(c.tableName),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			KeyConditionExpression:    expr.KeyCondition(),
		})
		for queryPaginator.HasMorePages() {
			response, err = queryPaginator.NextPage(c.ctx)
			if err != nil {
				return nil, fmt.Errorf("couldn't query variables for workspace-id=%s: %w\n", wsId, err)
			} else {
				var variablePage []dynamodbVariable
				err = attributevalue.UnmarshalListOfMaps(response.Items, &variablePage)
				if err != nil {
					return nil, fmt.Errorf("couldn't unmarshal query response for workspace-id=%s: %w\n", wsId, err)
				} else {
					dynamodbVariables = append(dynamodbVariables, variablePage...)
				}
			}
		}
	}
	return mapDynamodbVars(&dynamodbVariables), nil
}

// --------------------------------------------------------------------------------
// Utility functions
// --------------------------------------------------------------------------------
// Converts Dynamodb variables into abstracted workspace variables type
func mapDynamodbVars(dynamodbVars *[]dynamodbVariable) *[]WorkspaceVariable {
	var workspaceVars []WorkspaceVariable
	for _, dynamodbVar := range *dynamodbVars {
		workspaceVar := WorkspaceVariable{
			Id:        dynamodbVar.Id,
			Key:       dynamodbVar.Attributes.Key,
			Value:     dynamodbVar.Attributes.Value,
			Category:  dynamodbVar.Attributes.Category,
			Sensitive: dynamodbVar.Attributes.Sensitive}
		workspaceVars = append(workspaceVars, workspaceVar)
	}
	return &workspaceVars
}
