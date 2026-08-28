package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"lambda_create_post/internal/application"
	"lambda_create_post/internal/handler"
	"lambda_create_post/internal/infraestructure"
)

func main() {
	createHandler, err := buildDependencies()
	if err != nil {
		log.Fatal("error building dependencies: ", err)
	}

	proxy := chiadapter.NewV2(buildRouter(createHandler))
	lambda.Start(proxy.ProxyWithContextV2)
}

func buildDependencies() (*handler.CreatePostHandler, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	adapter := infraestructure.NewDynamoAdapter(dynamodb.NewFromConfig(cfg), os.Getenv("TABLENAME"))
	useCase := application.NewCreatePostUseCase(adapter)
	return handler.NewCreatePostHandler(useCase), nil
}

func buildRouter(createHandler *handler.CreatePostHandler) *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Post("/authors/{authorId}/posts", createHandler.CreatePost)
	return router
}
