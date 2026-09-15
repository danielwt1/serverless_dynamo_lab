package main

import (
	"context"
	"lambda_task_overdue_consumer/internal/application"
	appconfig "lambda_task_overdue_consumer/internal/config"
	"lambda_task_overdue_consumer/internal/handler"
	"lambda_task_overdue_consumer/internal/infraestructure"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	tableName, err := appconfig.ProcessTableName()
	if err != nil {
		log.Fatal("cargar configuración: ", err)
	}
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("cargar configuración AWS: ", err)
	}
	repository := infraestructure.NewDynamoNotificationRepository(dynamodb.NewFromConfig(awsConfig), tableName)
	lambda.Start(handler.NewSNSHandler(application.NewConsumeTaskOverdueUseCase(repository)).Handle)
}
