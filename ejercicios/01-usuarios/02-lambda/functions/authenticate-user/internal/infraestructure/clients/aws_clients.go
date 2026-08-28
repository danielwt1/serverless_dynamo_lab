package clients

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Clients struct {
	DynamoDb *dynamodb.Client
}

func NewClients(cfg aws.Config) (*Clients, error) {
	return &Clients{
		DynamoDb: dynamodb.NewFromConfig(cfg),
	}, nil
}
