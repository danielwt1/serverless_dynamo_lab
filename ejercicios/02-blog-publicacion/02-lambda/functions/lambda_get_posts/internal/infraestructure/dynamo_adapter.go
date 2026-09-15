package infraestructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"lambda_posts_mutable/internal/domain"
	"lambda_posts_mutable/internal/domain/ports/out"
)

type DynamoClientI interface {
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}
type DynamoAdapter struct {
	client    DynamoClientI
	tableName string
}

func NewDynamoAdapter(client DynamoClientI, tableName string) *DynamoAdapter {
	return &DynamoAdapter{
		client:    client,
		tableName: tableName,
	}
}

func (dn *DynamoAdapter) GetPublishedPosts(ctx context.Context, author_id string, status string, cursor string, limit int32) (domain.PostsPage, error) {
	startKey, err := decodeCursor(cursor)
	if err != nil {
		return domain.PostsPage{}, err
	}

	result, err := dn.client.Query(ctx, &dynamodb.QueryInput{
		TableName:      aws.String(dn.tableName),
		ConsistentRead: aws.Bool(true),
		// 1. Usamos #pk para el nombre de la columna y :pk_val para el valor
		KeyConditionExpression: aws.String("#pk = :pk_val AND begins_with(#status, :status)"),

		// 2. Aquí mapeas los nombres de tus columnas (comienzan con #)
		ExpressionAttributeNames: map[string]string{
			"#pk":     "PK", // Reemplaza #pk por el nombre real de tu columna PK
			"#status": "SK", // Si vas a usar status en un FilterExpression más adelante
		},

		// 3. Aquí mapeas los valores reales (comienzan con :)
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk_val": &types.AttributeValueMemberS{Value: "AUTHOR_ID#" + author_id},
			":status": &types.AttributeValueMemberS{Value: "STATUS#" + status},
		},
		ExclusiveStartKey: startKey,
		Limit:             aws.Int32(limit),
		ScanIndexForward:  aws.Bool(false),
	})
	if err != nil {
		return domain.PostsPage{}, err
	}
	postsPublised := make([]domain.PostModel, 0, len(result.Items))
	for _, item := range result.Items {
		postsPublised = append(postsPublised, domain.PostModel{
			AuthorId:    item["PK"].(*types.AttributeValueMemberS).Value[len("AUTHOR_ID#"):],
			PublishedAt: item["published_at"].(*types.AttributeValueMemberS).Value,
			PostId:      item["post_id"].(*types.AttributeValueMemberS).Value,
			CreatedAt:   item["created_at"].(*types.AttributeValueMemberS).Value,
			UpdatedAt:   item["updated_at"].(*types.AttributeValueMemberS).Value,
			Description: item["description"].(*types.AttributeValueMemberS).Value,
		})
	}

	nextCursor, err := encodeCursor(result.LastEvaluatedKey)
	if err != nil {
		return domain.PostsPage{}, err
	}

	return domain.PostsPage{Items: postsPublised, NextCursor: nextCursor}, nil

}

func (dn *DynamoAdapter) GetDraftPosts(ctx context.Context, authorID string, cursor string, limit int32) (domain.PostsPage, error) {
	startKey, err := decodeCursor(cursor)
	if err != nil {
		return domain.PostsPage{}, err
	}

	result, err := dn.client.Query(ctx, &dynamodb.QueryInput{
		TableName: aws.String(dn.tableName),
		IndexName: aws.String("draft_gsi"),

		// draft_gsi is sparse: only draft items have GSI1PK and GSI1SK.
		// GSI1PK = STATUS#DRAFT#AUTHOR_ID#<authorId>
		// GSI1SK = CREATED_AT#<createdAt>#POST_ID#<postId>
		KeyConditionExpression: aws.String("#gsiPK = :draftAuthor AND begins_with(#gsiSK, :createdAt)"),

		ExpressionAttributeNames: map[string]string{
			"#gsiPK": "GSI1PK",
			"#gsiSK": "GSI1SK",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":draftAuthor": &types.AttributeValueMemberS{Value: "STATUS#DRAFT#AUTHOR_ID#" + authorID},
			":createdAt": &types.AttributeValueMemberS{Value: "CREATED_AT#"},
		},
		ExclusiveStartKey: startKey,
		Limit:             aws.Int32(limit),
		ScanIndexForward:  aws.Bool(false),
	})
	if err != nil {
		return domain.PostsPage{}, err
	}
	postsPublised := make([]domain.PostModel, 0, len(result.Items))
	for _, item := range result.Items {
		postsPublised = append(postsPublised, domain.PostModel{
			AuthorId:    item["PK"].(*types.AttributeValueMemberS).Value[len("AUTHOR_ID#"):],
			PublishedAt: "",
			PostId:      item["post_id"].(*types.AttributeValueMemberS).Value,
			CreatedAt:   item["created_at"].(*types.AttributeValueMemberS).Value,
			UpdatedAt:   item["updated_at"].(*types.AttributeValueMemberS).Value,
			Description: item["description"].(*types.AttributeValueMemberS).Value,
		})
	}

	nextCursor, err := encodeCursor(result.LastEvaluatedKey)
	if err != nil {
		return domain.PostsPage{}, err
	}

	return domain.PostsPage{Items: postsPublised, NextCursor: nextCursor}, nil
}

func decodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, domain.ErrInvalidCursor
	}

	values := map[string]string{}
	if err := json.Unmarshal(payload, &values); err != nil || len(values) == 0 {
		return nil, domain.ErrInvalidCursor
	}

	key := make(map[string]types.AttributeValue, len(values))
	for name, value := range values {
		key[name] = &types.AttributeValueMemberS{Value: value}
	}
	return key, nil
}

func encodeCursor(lastEvaluatedKey map[string]types.AttributeValue) (string, error) {
	if len(lastEvaluatedKey) == 0 {
		return "", nil
	}

	values := make(map[string]string, len(lastEvaluatedKey))
	for name, attribute := range lastEvaluatedKey {
		stringAttribute, ok := attribute.(*types.AttributeValueMemberS)
		if !ok {
			return "", errors.New("cursor contains an unsupported attribute type")
		}
		values[name] = stringAttribute.Value
	}

	payload, err := json.Marshal(values)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

var _ out.GetInfoPort = (*DynamoAdapter)(nil)
