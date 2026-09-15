package infraestructure

import (
	"context"
	"get_task_lambda/internal/domain"
	"get_task_lambda/internal/domain/errors"
	"get_task_lambda/internal/infraestructure/util"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type DynamoClientStub struct {
	input       *dynamodb.QueryInput
	output      *dynamodb.QueryOutput
	errorDynamo error
}

func (stub *DynamoClientStub) Query(context context.Context, input *dynamodb.QueryInput, options ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	stub.input = input
	return stub.output, stub.errorDynamo
}

func Test_Get_users_tasks(t *testing.T) {
	usecases := []struct {
		name             string
		userId           string
		cursor           string
		output           *dynamodb.QueryOutput
		outputExpected   domain.QueryResult
		errorWanted      error
		errorThrown      error
		expectedStartKey map[string]types.AttributeValue
	}{
		{
			name:        "retorna error cuando el cursor es invalido",
			userId:      "1",
			cursor:      "1",
			errorWanted: errors.InvalidCursor,
		},
		{
			name:           "consulta la primera pagina sin cursor",
			userId:         "1",
			cursor:         "",
			output:         &dynamodb.QueryOutput{},
			outputExpected: domain.QueryResult{Task: []domain.TaskModel{}},
		},
		{
			name:   "consulta la siguiente pagina con cursor valido",
			userId: "1",
			cursor: mustCursor(t, map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			}),
			output:         &dynamodb.QueryOutput{},
			outputExpected: domain.QueryResult{Task: []domain.TaskModel{}},
			expectedStartKey: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			},
		},
		{
			name:   "Convieerte objetos devueltos por dynamo",
			userId: "1",
			cursor: mustCursor(t, map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			}),
			output: &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{
						"PK":          &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
						"SK":          &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
						"created_at":  &types.AttributeValueMemberS{Value: "2013-08-22T10:11:10"},
						"task_id":     &types.AttributeValueMemberS{Value: "task-3"},
						"owner_id":    &types.AttributeValueMemberS{Value: "1"},
						"updated_at":  &types.AttributeValueMemberS{Value: "2013-08-22T10:11:10"},
						"description": &types.AttributeValueMemberS{Value: "Tarea de test numero 1"},
						"status":      &types.AttributeValueMemberS{Value: "PENDING"},
					},
					{
						"PK":          &types.AttributeValueMemberS{Value: "OWNER_ID#2"},
						"SK":          &types.AttributeValueMemberS{Value: "TASK_ID#task-5"},
						"created_at":  &types.AttributeValueMemberS{Value: "2013-08-22T10:11:10"},
						"task_id":     &types.AttributeValueMemberS{Value: "task-5"},
						"owner_id":    &types.AttributeValueMemberS{Value: "2"},
						"updated_at":  &types.AttributeValueMemberS{Value: "2013-08-22T10:11:10"},
						"description": &types.AttributeValueMemberS{Value: "Tarea de test numero 2"},
						"status":      &types.AttributeValueMemberS{Value: "PENDING"},
					},
				},
			},
			outputExpected: domain.QueryResult{Task: []domain.TaskModel{
				{
					TaskId:      "task-3",
					OwnerId:     "1",
					Description: "Tarea de test numero 1",
					CreatedAt:   nil,
					Expire_at:   nil,
					Status:      "PENDING",
				},
				{
					TaskId: "task-5", OwnerId: "2",
					Description: "Tarea de test numero 2",
					CreatedAt:   nil,
					Expire_at:   nil,
					Status:      "PENDING",
				},
			},
				Cursor: ""},
			expectedStartKey: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			},
		},
		{
			name:   "Error en dynamo",
			userId: "1",
			cursor: mustCursor(t, map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			}),
			output:         &dynamodb.QueryOutput{},
			outputExpected: domain.QueryResult{Task: []domain.TaskModel{}},
			expectedStartKey: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK": &types.AttributeValueMemberS{Value: "TASK_ID#task-3"},
			},
			errorWanted: errors.DynamoError,
			errorThrown: errors.DynamoError,
		},
	}
	for _, uc := range usecases {
		t.Run(uc.name, func(t *testing.T) {
			stub := &DynamoClientStub{output: uc.output, errorDynamo: uc.errorThrown}
			client := NewDynamoAdapter(stub, "task")
			response, err := client.GetUserTasks(context.TODO(), uc.userId, uc.cursor)

			if uc.errorWanted != nil {
				assert.ErrorIs(t, err, uc.errorWanted)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, uc.outputExpected, response)
			assert.Equal(t, uc.expectedStartKey, stub.input.ExclusiveStartKey)
		})
	}
}

func Test_Get_pending_tasks(t *testing.T) {
	t.Run("convierte la proyeccion pendiente sin requerir status", func(t *testing.T) {
		stub := &DynamoClientStub{output: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{
			{
				"PK":          &types.AttributeValueMemberS{Value: "OWNER_ID#1"},
				"SK":          &types.AttributeValueMemberS{Value: "STATUS#PENDING#EXPIRED_AT#2026-09-06T00:00:00Z#TASK_ID#task-1"},
				"task_id":     &types.AttributeValueMemberS{Value: "task-1"},
				"owner_id":    &types.AttributeValueMemberS{Value: "1"},
				"description": &types.AttributeValueMemberS{Value: "Pagar factura"},
			}}}}
		adapter := NewDynamoAdapter(stub, "tasks")

		result, err := adapter.GetPendingTasksByUser(context.TODO(), "1", "")

		assert.NoError(t, err)
		assert.Equal(t, domain.QueryResult{Task: []domain.TaskModel{{
			TaskId: "task-1", OwnerId: "1", Description: "Pagar factura", Status: "PENDING",
		}}}, result)
		assert.Equal(t, "OWNER_ID#1", attributeString(stub.input.ExpressionAttributeValues[":PK"]))
		assert.Equal(t, "STATUS#PENDING#", attributeString(stub.input.ExpressionAttributeValues[":SK"]))
	})

	t.Run("devuelve error y no panic si falta un atributo requerido", func(t *testing.T) {
		stub := &DynamoClientStub{output: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{
			{"task_id": &types.AttributeValueMemberS{Value: "task-1"}, "owner_id": &types.AttributeValueMemberS{Value: "1"}},
		}}}
		adapter := NewDynamoAdapter(stub, "tasks")

		_, err := adapter.GetPendingTasksByUser(context.TODO(), "1", "")

		assert.ErrorIs(t, err, errors.InvalidTask)
	})
}

func attributeString(value types.AttributeValue) string {
	stringValue, ok := value.(*types.AttributeValueMemberS)
	if !ok {
		return ""
	}
	return stringValue.Value
}

func mustCursor(t *testing.T, key map[string]types.AttributeValue) string {
	// usar para que go sepa que es una funcion de ayuda para testing
	t.Helper()

	cursor, err := util.Encode(key)
	if err != nil {
		t.Fatalf("util.Encode() error = %v", err)
	}
	return cursor
}
