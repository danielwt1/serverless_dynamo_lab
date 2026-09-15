package infraestructure

import (
	"context"
	"fmt"
	"lambda_find_expired_task/domain"
	"lambda_find_expired_task/domain/ports/out"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClientProcessBatch interface {
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
}

type dynamoClientProcessUpdater interface {
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type dynamoClientProcessQueryer interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type notificationMarker struct {
	EntityType string `dynamo:"entity_type"`
	EventID    string `dynamo:"event_id"`
	Status     string `dynamo:"status"`
	BatchRunID string `dynamo:"batch_run_id"`
}

type batchEvent struct {
	EntityType string `dynamo:"entity_type"`
	Status     string `dynamo:"status"`
	domain.TaskOverdueEvent
}

type DynamoProcessAdapter[V any] struct {
	client    DynamoClientProcessBatch
	tableName string
}

func NewDynamoProcessAdapter[V any](client DynamoClientProcessBatch, tableName string) *DynamoProcessAdapter[V] {
	return &DynamoProcessAdapter[V]{client: client, tableName: tableName}
}

func (dc *DynamoProcessAdapter[V]) FindProcess(context context.Context, keys map[string]string) (V, error) {
	var empty V
	if keys == nil || len(keys) == 0 {
		return empty, fmt.Errorf("keys or keys must not be empty")
	}
	if _, ok := keys["id"]; !ok {
		return empty, fmt.Errorf("id not found in keys")
	}
	if _, ok := keys["sk"]; !ok {
		return empty, fmt.Errorf("sk not found in keys")
	}

	item, err := dc.client.GetItem(context, &dynamodb.GetItemInput{
		TableName:      &dc.tableName,
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: keys["id"]},
			"SK": &types.AttributeValueMemberS{Value: keys["sk"]},
		},
	})
	if err != nil {
		return empty, err
	}

	if len(item.Item) == 0 {
		return empty, out.ErrProcessNotFound
	}
	var process V
	err = attributevalue.UnmarshalMapWithOptions(item.Item, &process, func(options *attributevalue.DecoderOptions) {
		options.TagKey = "dynamo"
	})
	return process, err
}

func (dc *DynamoProcessAdapter[V]) FindOpenProcess(ctx context.Context) (domain.ProcessModel, error) {
	queryer, ok := dc.client.(dynamoClientProcessQueryer)
	if !ok {
		return domain.ProcessModel{}, fmt.Errorf("el cliente DynamoDB no permite consultar procesos abiertos")
	}
	result, err := queryer.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(dc.tableName),
		IndexName:              aws.String("process_gsi_open"),
		KeyConditionExpression: aws.String("GSI1PK = :process"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":process": &types.AttributeValueMemberS{Value: "PROCESS#OVERDUE"},
		},
		Limit: aws.Int32(2),
	})
	if err != nil {
		return domain.ProcessModel{}, fmt.Errorf("consultar proceso abierto: %w", err)
	}
	if len(result.Items) == 0 {
		return domain.ProcessModel{}, out.ErrProcessNotFound
	}
	if len(result.Items) > 1 {
		return domain.ProcessModel{}, fmt.Errorf("se encontró más de un proceso abierto")
	}
	var process domain.ProcessModel
	if err := attributevalue.UnmarshalMapWithOptions(result.Items[0], &process, func(options *attributevalue.DecoderOptions) {
		options.TagKey = "dynamo"
	}); err != nil {
		return domain.ProcessModel{}, fmt.Errorf("mapear proceso abierto: %w", err)
	}
	return process, nil
}

func (dc *DynamoProcessAdapter[V]) ReserveTaskOverdue(ctx context.Context, runID string, event domain.TaskOverdueEvent) error {
	if dc.tableName == "" || runID == "" || event.EventId == "" || event.TaskId == "" || event.ExpiredAt.IsZero() {
		return fmt.Errorf("tableName, run_id, event_id, task_id y expired_at son obligatorios")
	}
	markerPK := "TASK_OVERDUE#" + event.EventId
	markerOutput, err := dc.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(dc.tableName),
		ConsistentRead: aws.Bool(true),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: markerPK},
			"SK": &types.AttributeValueMemberS{Value: "EVENT#TASK_OVERDUE"},
		},
	})
	if err != nil {
		return fmt.Errorf("buscar marker: %w", err)
	}
	if len(markerOutput.Item) > 0 {
		var marker notificationMarker
		if err := attributevalue.UnmarshalMapWithOptions(markerOutput.Item, &marker, func(options *attributevalue.DecoderOptions) {
			options.TagKey = "dynamo"
		}); err != nil {
			return fmt.Errorf("mapear marker: %w", err)
		}
		if marker.Status == "PUBLISHED" || marker.Status == "RESERVED" && marker.BatchRunID == runID {
			return nil
		}
		return fmt.Errorf("el evento %s está reservado por otra corrida", event.EventId)
	}

	markerPut, err := newProcessPut(dc.tableName, markerPK, "EVENT#TASK_OVERDUE", notificationMarker{
		EntityType: "NOTIFICATION_MARKER",
		EventID:    event.EventId,
		Status:     "RESERVED",
		BatchRunID: runID,
	})
	if err != nil {
		return fmt.Errorf("mapear marker: %w", err)
	}
	eventSK := "EVENT#" + event.ExpiredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00") + "#" + event.TaskId
	eventPut, err := newProcessPut(dc.tableName, runID, eventSK, batchEvent{
		EntityType:       "BATCH_EVENT",
		Status:           "READY_TO_PUBLISH",
		TaskOverdueEvent: event,
	})
	if err != nil {
		return fmt.Errorf("mapear evento: %w", err)
	}
	_, err = dc.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{markerPut, eventPut},
	})
	if err != nil {
		return fmt.Errorf("reservar evento: %w", err)
	}
	return nil
}

func (dc *DynamoProcessAdapter[V]) ListReadyTaskOverdue(ctx context.Context, runID, cursor string) ([]domain.TaskOverdueEvent, string, error) {
	queryer, ok := dc.client.(dynamoClientProcessQueryer)
	if !ok {
		return nil, "", fmt.Errorf("el cliente DynamoDB no permite consultar el outbox")
	}
	startKey, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", fmt.Errorf("decodificar cursor del outbox: %w", err)
	}
	result, err := queryer.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(dc.tableName),
		KeyConditionExpression: aws.String("PK = :run_id AND begins_with(SK, :event)"),
		FilterExpression:       aws.String("#status = :ready"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":run_id": &types.AttributeValueMemberS{Value: runID},
			":event":  &types.AttributeValueMemberS{Value: "EVENT#"},
			":ready":  &types.AttributeValueMemberS{Value: "READY_TO_PUBLISH"},
		},
		ExclusiveStartKey: startKey,
		Limit:             aws.Int32(25),
	})
	if err != nil {
		return nil, "", fmt.Errorf("consultar outbox: %w", err)
	}
	nextCursor, err := encodeCursor(result.LastEvaluatedKey)
	if err != nil {
		return nil, "", fmt.Errorf("codificar cursor del outbox: %w", err)
	}
	events := make([]domain.TaskOverdueEvent, 0, len(result.Items))
	for _, item := range result.Items {
		var stored batchEvent
		if err := attributevalue.UnmarshalMapWithOptions(item, &stored, func(options *attributevalue.DecoderOptions) {
			options.TagKey = "dynamo"
		}); err != nil {
			return nil, "", fmt.Errorf("mapear evento del outbox: %w", err)
		}
		events = append(events, stored.TaskOverdueEvent)
	}
	return events, nextCursor, nil
}

func (dc *DynamoProcessAdapter[V]) MarkTaskOverduePublished(ctx context.Context, runID string, event domain.TaskOverdueEvent, publishedAt string) error {
	if runID == "" || event.EventId == "" || publishedAt == "" {
		return fmt.Errorf("run_id, event_id y published_at son obligatorios")
	}
	eventSK := "EVENT#" + event.ExpiredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00") + "#" + event.TaskId
	updates := []types.TransactWriteItem{
		{Update: &types.Update{
			TableName:           aws.String(dc.tableName),
			Key:                 map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: runID}, "SK": &types.AttributeValueMemberS{Value: eventSK}},
			ConditionExpression: aws.String("#status = :ready"),
			UpdateExpression:    aws.String("SET #status = :published, published_at = :published_at"),
			ExpressionAttributeNames: map[string]string{
				"#status": "status",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":ready":        &types.AttributeValueMemberS{Value: "READY_TO_PUBLISH"},
				":published":    &types.AttributeValueMemberS{Value: "PUBLISHED"},
				":published_at": &types.AttributeValueMemberS{Value: publishedAt},
			},
		}},
		{Update: &types.Update{
			TableName: aws.String(dc.tableName),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "TASK_OVERDUE#" + event.EventId},
				"SK": &types.AttributeValueMemberS{Value: "EVENT#TASK_OVERDUE"},
			},
			ConditionExpression: aws.String("#status = :reserved AND batch_run_id = :run_id"),
			UpdateExpression:    aws.String("SET #status = :published, published_at = :published_at"),
			ExpressionAttributeNames: map[string]string{
				"#status": "status",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":reserved":     &types.AttributeValueMemberS{Value: "RESERVED"},
				":published":    &types.AttributeValueMemberS{Value: "PUBLISHED"},
				":published_at": &types.AttributeValueMemberS{Value: publishedAt},
				":run_id":       &types.AttributeValueMemberS{Value: runID},
			},
		}},
	}
	if _, err := dc.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: updates}); err != nil {
		return fmt.Errorf("confirmar evento publicado: %w", err)
	}
	return nil
}

func (dc *DynamoProcessAdapter[V]) SaveShardCheckpoint(ctx context.Context, runID string, shard domain.ProcessShardModel) error {
	if dc.tableName == "" || runID == "" || shard.ShardId == "" || shard.Updated_at == "" {
		return fmt.Errorf("tableName, run_id, shard_id y updated_at son obligatorios")
	}
	if shard.RunId != "" && shard.RunId != runID {
		return fmt.Errorf("el checkpoint pertenece a otra corrida")
	}
	if shard.Status != "IN_PROGRESS" && shard.Status != "COMPLETED" {
		return fmt.Errorf("estado de checkpoint no soportado: %q", shard.Status)
	}
	updater, ok := dc.client.(dynamoClientProcessUpdater)
	if !ok {
		return fmt.Errorf("el cliente DynamoDB no permite actualizar checkpoints")
	}
	_, err := updater.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(dc.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: runID},
			"SK": &types.AttributeValueMemberS{Value: "SHARD#" + shard.ShardId},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
		UpdateExpression:    aws.String("SET #status = :status, #cursor = :cursor, pages_completed = :pages, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
			"#cursor": "cursor",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: shard.Status},
			":cursor":     &types.AttributeValueMemberS{Value: shard.Cursor},
			":pages":      &types.AttributeValueMemberN{Value: fmt.Sprint(shard.Pages_completed)},
			":updated_at": &types.AttributeValueMemberS{Value: shard.Updated_at},
		},
	})
	if err != nil {
		return fmt.Errorf("actualizar checkpoint: %w", err)
	}
	return nil
}

func (dc *DynamoProcessAdapter[V]) SetProcessStatus(ctx context.Context, runID, status, lastError, updatedAt string) error {
	if dc.tableName == "" || runID == "" || updatedAt == "" {
		return fmt.Errorf("tableName, run_id y updated_at son obligatorios")
	}
	switch status {
	case "RUNNING", "PUBLISHING", "COMPLETED", "FAILED_RETRYABLE":
	default:
		return fmt.Errorf("estado de proceso no soportado: %q", status)
	}
	updater, ok := dc.client.(dynamoClientProcessUpdater)
	if !ok {
		return fmt.Errorf("el cliente DynamoDB no permite actualizar procesos")
	}
	updateExpression := "SET #status = :status, updated_at = :updated_at, last_error = :last_error"
	if status == "COMPLETED" {
		updateExpression += " REMOVE GSI1PK, GSI1SK"
	}
	_, err := updater.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(dc.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: runID},
			"SK": &types.AttributeValueMemberS{Value: "META"},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
		UpdateExpression:    aws.String(updateExpression),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: status},
			":updated_at": &types.AttributeValueMemberS{Value: updatedAt},
			":last_error": &types.AttributeValueMemberS{Value: lastError},
		},
	})
	if err != nil {
		return fmt.Errorf("actualizar proceso: %w", err)
	}
	return nil
}

// InitProcessAndShardProcess crea un proceso y sus checkpoints de forma atómica.
func (dc *DynamoProcessAdapter[V]) InitProcessAndShardProcess(ctx context.Context, process domain.ProcessModel, shards []domain.ProcessShardModel) error {
	if dc.tableName == "" || process.ExecutionTime == "" || process.UpdatedAt == "" {
		return fmt.Errorf("tableName, execution_time y updated_at son obligatorios")
	}
	if process.Status != "RUNNING" {
		return fmt.Errorf("el proceso debe iniciar RUNNING")
	}
	// Una acción para META y hasta 99 para los checkpoints.
	if len(shards) == 0 || len(shards) > 99 {
		return fmt.Errorf("se requieren entre 1 y 99 shards")
	}
	runID := "OVERDUE_RUN#" + process.ExecutionTime
	if process.ActiveRunId != "" && process.ActiveRunId != runID {
		return fmt.Errorf("active_run_id no coincide con execution_time")
	}
	process.ActiveRunId = runID
	process.EntityTpe = "BATCH_RUN"
	put, err := newProcessPut(dc.tableName, runID, "META", process)
	if err != nil {
		return fmt.Errorf("mapear proceso: %w", err)
	}
	put.Put.Item["shard_count"] = &types.AttributeValueMemberN{Value: fmt.Sprint(len(shards))}
	put.Put.Item["GSI1PK"] = &types.AttributeValueMemberS{Value: "PROCESS#OVERDUE"}
	put.Put.Item["GSI1SK"] = &types.AttributeValueMemberS{Value: "OPEN#" + process.UpdatedAt + "#" + process.ExecutionTime}
	items := make([]types.TransactWriteItem, 0, len(shards)+1)
	items = append(items, put)
	seen := make(map[string]bool, len(shards))
	for _, shard := range shards {
		if shard.ShardId == "" || seen[shard.ShardId] {
			return fmt.Errorf("shard_id vacío o duplicado: %q", shard.ShardId)
		}
		seen[shard.ShardId] = true
		if shard.RunId != "" && shard.RunId != runID {
			return fmt.Errorf("shard %q pertenece a otra corrida", shard.ShardId)
		}
		if shard.Status != "PENDING" || shard.Cursor != "" || shard.Pages_completed != 0 || shard.Ttl < 0 {
			return fmt.Errorf("shard %q requiere estado PENDING, cursor vacío, cero páginas y TTL no negativo", shard.ShardId)
		}
		shard.RunId = runID
		shard.EntityTpe = "SHARD_CHECKPOINT"
		if shard.Updated_at == "" {
			shard.Updated_at = process.UpdatedAt
		}
		put, err := newProcessPut(dc.tableName, runID, "SHARD#"+shard.ShardId, shard)
		if err != nil {
			return fmt.Errorf("mapear shard %q: %w", shard.ShardId, err)
		}
		// Un TTL cero significa que no se solicitó expiración.
		if shard.Ttl == 0 {
			delete(put.Put.Item, "ttl")
		}
		items = append(items, put)
	}
	_, err = dc.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	if err != nil {
		return fmt.Errorf("inicializar proceso y shards: %w", err)
	}
	return nil
}

// Cada tipo define sus atributos con tags; las claves pertenecen al adaptador.
func newProcessPut[T any](tableName, pk, sk string, value T) (types.TransactWriteItem, error) {
	item, err := attributevalue.MarshalMapWithOptions(value, func(options *attributevalue.EncoderOptions) {
		options.TagKey = "dynamo"
	})
	if err != nil {
		return types.TransactWriteItem{}, err
	}
	item["PK"] = &types.AttributeValueMemberS{Value: pk}
	item["SK"] = &types.AttributeValueMemberS{Value: sk}
	return types.TransactWriteItem{Put: &types.Put{
		TableName:           aws.String(tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
	}}, nil
}

var _ out.ProcessRepository[domain.ProcessModel] = (*DynamoProcessAdapter[domain.ProcessModel])(nil)
var _ out.ProcessRepository[domain.ProcessShardModel] = (*DynamoProcessAdapter[domain.ProcessShardModel])(nil)
var _ out.ProcessProgressRepository = (*DynamoProcessAdapter[domain.ProcessModel])(nil)
