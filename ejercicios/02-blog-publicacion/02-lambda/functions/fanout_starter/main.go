package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"fanout_starter/internal/application"
	"fanout_starter/internal/handler"
	"fanout_starter/internal/infraestructure"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("load AWS config: ", err)
	}

	useCase := application.NewStartFanoutUseCase(
		infraestructure.NewProgressRepository(dynamodb.NewFromConfig(cfg), os.Getenv("FANOUT_PROGRESS_TABLE")),
		infraestructure.NewFanoutQueue(sqs.NewFromConfig(cfg), os.Getenv("FANOUT_JOBS_QUEUE_URL")),
	)
	lambda.Start(handler.NewDynamoStreamHandler(useCase).Handle)
}
