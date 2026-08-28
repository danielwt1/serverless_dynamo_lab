package main

import (
	"context"
	"create-user/internal/application"
	appconfig "create-user/internal/config"
	handler2 "create-user/internal/handler"
	"create-user/internal/infrastructure"
	clientsApi "create-user/internal/infrastructure/clients"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func main() {
	awsConfig, err := loadAWSConfig(context.Background())
	if err != nil {
		log.Fatal("load AWS configuration: ", err)
	}
	applicationConfig, err := appconfig.Load()
	if err != nil {
		log.Fatal("load application configuration: ", err)
	}
	//init client
	clients := clientsApi.NewClients(awsConfig)
	//init dynamo client
	dynamoClient := clients.DynamoDB
	//init adapter
	dynamoAdapter := infrastructure.NewCreateUserAdapter(
		dynamoClient,
		applicationConfig.UsersTableName,
	)
	//use cases
	useCase := application.NewCreateUserUsecase(dynamoAdapter)
	//handler
	handler := handler2.NewCreateUserHandler(useCase)
	lambda.Start(handler.CreateUser)

}

func loadAWSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx)
}
