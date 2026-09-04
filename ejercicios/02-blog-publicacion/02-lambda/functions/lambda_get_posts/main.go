package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"lambda_posts_mutable/internal/application"
	"lambda_posts_mutable/internal/handler"
	"lambda_posts_mutable/internal/infraestructure"
	"log"
	"os"
)

func main() {
	handlerGet, err := buildDependencies()
	if err != nil {
		log.Fatal("Error building dependencies: ", err)
	}
	router, err := buildRouter(handlerGet)
	if err != nil {
		log.Fatal("Error building router: ", err)
	}
	proxy := chiadapter.NewV2(router)
	lambda.Start(proxy.ProxyWithContextV2)
}

func buildDependencies() (*handler.GetInfoHandler, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal("Error loading AWS config.")
		return nil, err
	}
	//dynamoClient
	dynamoClinet := dynamodb.NewFromConfig(cfg)
	//Get table name from Enviroment
	tableName := os.Getenv("TABLENAME")
	//build adapter
	adapter := infraestructure.NewDynamoAdapter(dynamoClinet, tableName)
	//build use case
	usecase := application.NewGetInfoUseCase(adapter)
	//build handler
	handlerGet := handler.NewGetInfoHandler(usecase)
	return handlerGet, nil
}
func buildRouter(getHandler *handler.GetInfoHandler) (*chi.Mux, error) {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	router.Get("/authors/{authorId}/posts", getHandler.GetPublishedPosts)
	router.Get("/authors/{authorId}/drafts", getHandler.GetDraftPosts)
	return router, nil
}
