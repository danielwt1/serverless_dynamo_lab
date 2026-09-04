package main

import (
	"context"
	"log"
	"os"

	"authenticate-user/internal/application"
	"authenticate-user/internal/handler"
	"authenticate-user/internal/infraestructure"
	"authenticate-user/internal/infraestructure/clients"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("load AWS configuration: %v", err)
	}

	awsClients, err := clients.NewClients(cfg)
	if err != nil {
		log.Fatalf("create AWS clients: %v", err)
	}

	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		log.Fatal("TABLE_NAME environment variable is required")
	}

	persistenceAdapter := infraestructure.NewUserPersistenceAdapter(awsClients.DynamoDb, tableName)
	userLogin := application.NewUserLogin(persistenceAdapter)
	authenticateUserHandler := handler.NewAuthenticateUserHandler(userLogin)

	lambda.Start(authenticateUserHandler.Handle)
}
