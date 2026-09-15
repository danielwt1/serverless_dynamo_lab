package main

import (
	"context"
	"lambda_create_task/internal/application"
	appconfig "lambda_create_task/internal/config"
	"lambda_create_task/internal/handler"
	"lambda_create_task/internal/infraestructure"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	tableName, err := appconfig.TaskTableName()
	if err != nil {
		log.Fatal("cargar configuración: ", err)
	}
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("cargar configuración AWS: ", err)
	}
	repository := infraestructure.NewDynamoTaskRepository(dynamodb.NewFromConfig(awsConfig), tableName)
	lambda.Start(handler.NewHTTPHandler(application.NewCreateTaskUseCase(repository)).Handle)
}
