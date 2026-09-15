package main

import (
	"context"
	"lambda_find_expired_task/domain"
	"lambda_find_expired_task/internal/application"
	appconfig "lambda_find_expired_task/internal/config"
	"lambda_find_expired_task/internal/handler"
	"lambda_find_expired_task/internal/infraestructure"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func main() {
	applicationConfig, err := appconfig.Load()
	if err != nil {
		log.Fatal("cargar configuración: ", err)
	}
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("cargar configuración AWS: ", err)
	}

	dynamoClient := dynamodb.NewFromConfig(awsConfig)
	processRepository := infraestructure.NewDynamoProcessAdapter[domain.ProcessModel](dynamoClient, applicationConfig.ProcessTableName)
	shardRepository := infraestructure.NewDynamoProcessAdapter[domain.ProcessShardModel](dynamoClient, applicationConfig.ProcessTableName)
	useCase := application.NewFindExpiredTasksUseCase(
		infraestructure.NewDynamoQueryOverDueAdapter(dynamoClient, applicationConfig.TaskTableName),
		processRepository,
		shardRepository,
		processRepository,
		infraestructure.NewSNSRecipient(sns.NewFromConfig(awsConfig), applicationConfig.OverdueTopicARN),
	)

	lambda.Start(handler.NewScheduledHandler(useCase).Handle)
}
