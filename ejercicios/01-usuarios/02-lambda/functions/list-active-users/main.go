package main

import (
	"context"
	"log"
	"os"

	"list-active-users/internal/application"
	"list-active-users/internal/handler"
	"list-active-users/internal/infrastructure"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}

	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatal("TABLE_NAME environment variable is required")
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	activeUsersAdapter := infrastructure.NewActiveUsersAdapter(dynamoClient, tableName)
	listActiveUsers := application.NewListActiveUsersUseCase(activeUsersAdapter)
	listActiveUsersHandler := handler.NewListActiveUsersHandler(listActiveUsers)

	lambda.Start(listActiveUsersHandler.List)
}
