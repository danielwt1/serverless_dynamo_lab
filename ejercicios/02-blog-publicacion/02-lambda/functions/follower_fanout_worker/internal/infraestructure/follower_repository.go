package infraestructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	"follower_fanout_worker/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type QueryClient interface { Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) }
type FollowerRepository struct { client QueryClient; tableName string }
func NewFollowerRepository(client QueryClient, tableName string) *FollowerRepository { return &FollowerRepository{client: client, tableName: tableName} }
func (r *FollowerRepository) FindPage(ctx context.Context, authorID, cursor string, limit int32) (domain.FollowerPage, error) { startKey, err := decodeCursor(cursor); if err != nil { return domain.FollowerPage{}, err }; output, err := r.client.Query(ctx, &dynamodb.QueryInput{TableName: aws.String(r.tableName), KeyConditionExpression: aws.String("#pk = :pk AND begins_with(#sk, :prefix)"), ExpressionAttributeNames: map[string]string{"#pk":"PK", "#sk":"SK"}, ExpressionAttributeValues: map[string]types.AttributeValue{":pk": &types.AttributeValueMemberS{Value:"FOLLOWING#"+authorID}, ":prefix": &types.AttributeValueMemberS{Value:"FOLLOWER#"}}, ExclusiveStartKey: startKey, Limit: aws.Int32(limit)}); if err != nil { return domain.FollowerPage{}, err }; ids := make([]string, 0, len(output.Items)); for _, item := range output.Items { sk, ok := item["SK"].(*types.AttributeValueMemberS); if !ok || len(sk.Value) <= len("FOLLOWER#") { return domain.FollowerPage{}, errors.New("invalid follower item") }; ids = append(ids, sk.Value[len("FOLLOWER#"):]) }; next, err := encodeCursor(output.LastEvaluatedKey); return domain.FollowerPage{RecipientIDs: ids, NextCursor: next}, err }
func decodeCursor(cursor string) (map[string]types.AttributeValue, error) { if cursor == "" { return nil, nil }; payload, err := base64.RawURLEncoding.DecodeString(cursor); if err != nil { return nil, errors.New("invalid cursor") }; values := map[string]string{}; if err := json.Unmarshal(payload, &values); err != nil || len(values) == 0 { return nil, errors.New("invalid cursor") }; key := make(map[string]types.AttributeValue, len(values)); for name, value := range values { key[name] = &types.AttributeValueMemberS{Value:value} }; return key, nil }
func encodeCursor(key map[string]types.AttributeValue) (string, error) { if len(key) == 0 { return "", nil }; values := map[string]string{}; for name, attribute := range key { stringAttribute, ok := attribute.(*types.AttributeValueMemberS); if !ok { return "", errors.New("unsupported cursor attribute") }; values[name] = stringAttribute.Value }; payload, err := json.Marshal(values); if err != nil { return "", err }; return base64.RawURLEncoding.EncodeToString(payload), nil }
