package main

import (
	"context"
	"log"
	"os"

	"follower_fanout_worker/internal/application"
	"follower_fanout_worker/internal/handler"
	"follower_fanout_worker/internal/infraestructure"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil { log.Fatal("load AWS config: ", err) }
	dynamoClient, sqsClient := dynamodb.NewFromConfig(cfg), sqs.NewFromConfig(cfg)
	useCase := application.NewProcessFanoutUseCase(
		infraestructure.NewFollowerRepository(dynamoClient, os.Getenv("FOLLOWERS_TABLE")),
		infraestructure.NewProgressRepository(dynamoClient, os.Getenv("FANOUT_PROGRESS_TABLE")),
		infraestructure.NewSQSQueue(sqsClient, os.Getenv("FANOUT_JOBS_QUEUE_URL")),
		infraestructure.NewSQSQueue(sqsClient, os.Getenv("NOTIFICATION_JOBS_QUEUE_URL")),
	)
	lambda.Start(handler.NewSQSHandler(useCase).Handle)
}
