package clients

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewClients_CreatesDynamoDBClient(t *testing.T) {
	clients, err := NewClients(aws.Config{Region: "us-east-1"})

	if err != nil {
		t.Fatalf("NewClients() error = %v, want nil", err)
	}
	if clients == nil {
		t.Fatal("NewClients() = nil, want a Clients instance")
	}
	if clients.DynamoDb == nil {
		t.Error("NewClients().DynamoDb = nil, want a DynamoDB client")
	}
}
