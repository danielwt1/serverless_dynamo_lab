package util

import (
	"encoding/base64"
	"encoding/json"
	"get_task_lambda/internal/domain/errors"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func Decode(cursor string) (map[string]types.AttributeValue, error) {
	if cursor == "" {
		return nil, nil
	}
	decodeB64, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errors.InvalidCursor
	}
	//valida que sea un json
	values := make(map[string]string)
	err = json.Unmarshal(decodeB64, &values)
	if err != nil {
		return nil, errors.InvalidCursor
	}
	deserializedValues := make(map[string]types.AttributeValue, len(values))
	for key, value := range values {
		deserializedValues[key] = &types.AttributeValueMemberS{Value: value}
	}
	return deserializedValues, nil
}

func Encode(dynamoKey map[string]types.AttributeValue) (string, error) {
	if len(dynamoKey) == 0 {
		return "", nil
	}
	values := make(map[string]string)
	for key, value := range dynamoKey {
		presentValue, ok := value.(*types.AttributeValueMemberS)
		if !ok {
			return "", errors.InvalidCursor
		}
		values[key] = presentValue.Value
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "", errors.InvalidCursor
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil

}
