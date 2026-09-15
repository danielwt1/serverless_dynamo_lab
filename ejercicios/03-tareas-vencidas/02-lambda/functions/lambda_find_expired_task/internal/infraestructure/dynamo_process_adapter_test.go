package infraestructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"lambda_find_expired_task/domain"
)

type processClientStub struct {
	input       *dynamodb.TransactWriteItemsInput
	getOutput   *dynamodb.GetItemOutput
	queryInput  *dynamodb.QueryInput
	queryOutput *dynamodb.QueryOutput
	updateInput *dynamodb.UpdateItemInput
	err         error
}

func (s *processClientStub) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if s.getOutput != nil {
		return s.getOutput, s.err
	}
	return &dynamodb.GetItemOutput{}, s.err
}

func (s *processClientStub) TransactWriteItems(_ context.Context, input *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	s.input = input
	return &dynamodb.TransactWriteItemsOutput{}, s.err
}

func (s *processClientStub) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	s.queryInput = input
	if s.queryOutput != nil {
		return s.queryOutput, s.err
	}
	return &dynamodb.QueryOutput{}, s.err
}

func (s *processClientStub) UpdateItem(_ context.Context, input *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	s.updateInput = input
	return &dynamodb.UpdateItemOutput{}, s.err
}

func TestInitProcessAndShards(t *testing.T) {
	client := &processClientStub{}
	repo := NewDynamoProcessAdapter[domain.ProcessModel](client, "batch_notification")
	process := domain.ProcessModel{Status: "RUNNING", ExecutionTime: "2026-09-08T00:00:00Z", UpdatedAt: "2026-09-08T00:00:00Z"}
	shards := []domain.ProcessShardModel{{ShardId: "00", Status: "PENDING", Ttl: 2000000000}, {ShardId: "01", Status: "PENDING"}}
	if err := repo.InitProcessAndShardProcess(context.Background(), process, shards); err != nil {
		t.Fatal(err)
	}
	if len(client.input.TransactItems) != 3 {
		t.Fatal("se esperaba META y dos checkpoints")
	}
	for i, sk := range []string{"META", "SHARD#00", "SHARD#01"} {
		put := client.input.TransactItems[i].Put
		if *put.TableName != "batch_notification" || put.ConditionExpression == nil {
			t.Fatal("tabla o condición ausente")
		}
		if put.Item["PK"].(*types.AttributeValueMemberS).Value != "OVERDUE_RUN#"+process.ExecutionTime || put.Item["SK"].(*types.AttributeValueMemberS).Value != sk {
			t.Fatal("claves incorrectas")
		}
	}
	meta := client.input.TransactItems[0].Put.Item
	checkpoint := client.input.TransactItems[1].Put.Item
	if meta["entity_type"].(*types.AttributeValueMemberS).Value != "BATCH_RUN" || checkpoint["entity_type"].(*types.AttributeValueMemberS).Value != "SHARD_CHECKPOINT" {
		t.Fatal("tipo de entidad incorrecto")
	}
	if meta["shard_count"].(*types.AttributeValueMemberN).Value != "2" || meta["GSI1PK"].(*types.AttributeValueMemberS).Value != "PROCESS#OVERDUE" {
		t.Fatal("metadatos incorrectos")
	}
	if checkpoint["pages_completed"].(*types.AttributeValueMemberN).Value != "0" || checkpoint["ttl"].(*types.AttributeValueMemberN).Value != "2000000000" {
		t.Fatal("campos numéricos incorrectos")
	}
	if _, ok := checkpoint["execution_time"]; ok {
		t.Fatal("atributo del proceso filtrado al checkpoint")
	}
	if _, ok := client.input.TransactItems[2].Put.Item["ttl"]; ok {
		t.Fatal("TTL cero debe omitirse")
	}
	if shards[0].RunId != "" || process.ActiveRunId != "" {
		t.Fatal("se modificaron los argumentos")
	}

	client.err = &types.TransactionCanceledException{}
	if err := repo.InitProcessAndShardProcess(context.Background(), process, shards); !errors.Is(err, client.err) {
		t.Fatalf("se perdió el error AWS: %v", err)
	}
	for name, invalid := range map[string][]domain.ProcessShardModel{
		"sin shards":         nil,
		"demasiados":         make([]domain.ProcessShardModel, 100),
		"duplicados":         {shards[0], shards[0]},
		"otra corrida":       {{ShardId: "00", Status: "PENDING", RunId: "otra"}},
		"sin id":             {{Status: "PENDING"}},
		"progreso existente": {{ShardId: "00", Status: "PENDING", Cursor: "cursor"}},
	} {
		t.Run(name, func(t *testing.T) {
			client.input = nil
			if err := repo.InitProcessAndShardProcess(context.Background(), process, invalid); err == nil || client.input != nil {
				t.Fatal("debe rechazar la entrada antes de escribir")
			}
		})
	}
}

func TestReserveTaskOverdueCreatesMarkerAndOutboxAtomically(t *testing.T) {
	client := &processClientStub{}
	repo := NewDynamoProcessAdapter[domain.ProcessModel](client, "batch_notification")
	dueAt := time.Date(2026, 9, 13, 10, 30, 0, 0, time.UTC)
	event := domain.TaskOverdueEvent{
		EventId:       "task-1#2026-09-13T10:30:00Z",
		TaskId:        "task-1",
		OwnerId:       "owner-1",
		Description:   "pagar factura",
		ExpiredAt:     dueAt,
		ExecutionTime: time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC),
	}
	if err := repo.ReserveTaskOverdue(context.Background(), "OVERDUE_RUN#2026-09-14T20:00:00Z", event); err != nil {
		t.Fatal(err)
	}
	if len(client.input.TransactItems) != 2 {
		t.Fatalf("acciones = %d, quiere marker y evento", len(client.input.TransactItems))
	}
	marker := client.input.TransactItems[0].Put.Item
	outbox := client.input.TransactItems[1].Put.Item
	if marker["PK"].(*types.AttributeValueMemberS).Value != "TASK_OVERDUE#"+event.EventId || marker["status"].(*types.AttributeValueMemberS).Value != "RESERVED" {
		t.Fatalf("marker incorrecto: %v", marker)
	}
	if outbox["PK"].(*types.AttributeValueMemberS).Value != "OVERDUE_RUN#2026-09-14T20:00:00Z" || outbox["status"].(*types.AttributeValueMemberS).Value != "READY_TO_PUBLISH" {
		t.Fatalf("outbox incorrecto: %v", outbox)
	}
	if outbox["event_id"].(*types.AttributeValueMemberS).Value != event.EventId || outbox["task_id"].(*types.AttributeValueMemberS).Value != event.TaskId {
		t.Fatalf("payload incompleto: %v", outbox)
	}
}

func TestSetProcessStatusCompletesAndRemovesOpenIndexKeys(t *testing.T) {
	client := &processClientStub{}
	repo := NewDynamoProcessAdapter[domain.ProcessModel](client, "batch_notification")
	if err := repo.SetProcessStatus(context.Background(), "OVERDUE_RUN#run", "COMPLETED", "", "2026-09-14T20:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if client.updateInput == nil || *client.updateInput.UpdateExpression != "SET #status = :status, updated_at = :updated_at, last_error = :last_error REMOVE GSI1PK, GSI1SK" {
		t.Fatalf("update de cierre incorrecto: %+v", client.updateInput)
	}
}
