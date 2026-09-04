package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

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
	leaseSeconds := int64(120)
	if raw := os.Getenv("FANOUT_LEASE_SECONDS"); raw != "" { if parsed, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && parsed > 0 { leaseSeconds = parsed } else { log.Fatal("FANOUT_LEASE_SECONDS must be a positive integer") } }
	useCase := application.NewProcessFanoutUseCase(
		infraestructure.NewFollowerRepository(dynamoClient, os.Getenv("FOLLOWERS_TABLE")),
		infraestructure.NewProgressRepository(dynamoClient, os.Getenv("FANOUT_PROGRESS_TABLE")),
		infraestructure.NewSQSQueue(sqsClient, os.Getenv("FANOUT_JOBS_QUEUE_URL")),
		infraestructure.NewSQSQueue(sqsClient, os.Getenv("NOTIFICATION_JOBS_QUEUE_URL")),
		time.Duration(leaseSeconds)*time.Second,
	)
	lambda.Start(handler.NewSQSHandler(useCase).Handle)
}
