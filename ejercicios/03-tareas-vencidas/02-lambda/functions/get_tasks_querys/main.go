package main

import (
	"context"
	"get_task_lambda/internal/application"
	"get_task_lambda/internal/handlers"
	"get_task_lambda/internal/infraestructure"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	//init cfg
	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	//init dynamo
	clientDynamo := dynamodb.NewFromConfig(cfg)
	// get dynamo database from env
	tablename := strings.TrimSpace(os.Getenv("TASK_TABLE_NAME"))
	if tablename == "" {
		tablename = strings.TrimSpace(os.Getenv("TABLENAME"))
	}
	if tablename == "" {
		panic("TASK_TABLE_NAME is required")
	}

	//init repository
	dynamoRepository := infraestructure.NewDynamoAdapter(clientDynamo, tablename)

	//init usecase
	usecase := application.NewTasksQueryApi(dynamoRepository)

	//init handler
	handler := handlers.NewTaskHandlers(usecase)

	router := createRouter(handler)

	proxy := chiadapter.NewV2(router)

	lambda.Start(proxy)

}

func createRouter(handlers *handlers.TaskHandlers) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Get("/tasks/{ownerId}/pending", handlers.GetPendingTasksByUserId)
	router.Get("/tasks/{ownerId}", handlers.GetTasksByUser)
	return router
}
