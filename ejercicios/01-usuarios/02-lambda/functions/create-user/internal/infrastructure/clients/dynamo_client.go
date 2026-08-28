package clients

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// Clients agrupa los clientes AWS que necesita esta Lambda.
// Cuando hagan falta otros servicios, agregalos aqui (por ejemplo, S3 o EventBridge).
type Clients struct {
	DynamoDB *dynamodb.Client
}

// New construye los clientes a partir de una configuracion AWS ya cargada.
// La configuracion se carga en main.go, que es el punto de composicion.
func NewClients(cfg aws.Config) *Clients {
	return &Clients{
		DynamoDB: dynamodb.NewFromConfig(cfg),
	}
}
