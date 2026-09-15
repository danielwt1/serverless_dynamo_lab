package utils

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func ConverterDynamo[V any](source []map[string]types.AttributeValue) ([]V, error) {
	response := make([]V, 0, len(source))
	var destination V
	structType := reflect.TypeOf(destination)

	if structType == nil || structType.Kind() != reflect.Struct {
		return nil, errors.New("V debe ser un struct")
	}

	fields, err := getDynamoFields(structType)
	if err != nil {
		return nil, err
	}

	for itemIndex, item := range source {
		var destination V
		destinationValue := reflect.ValueOf(&destination).Elem()

		for _, dynamoField := range fields {
			field := destinationValue.Field(dynamoField.index)
			attributeValue, exists := item[dynamoField.attributeName]
			if !exists {
				return nil, fmt.Errorf(
					"item %d: no existe el atributo Dynamo %q",
					itemIndex,
					dynamoField.attributeName,
				)
			}

			if err := setDynamoField(field, attributeValue); err != nil {
				return nil, fmt.Errorf(
					"item %d, atributo %q: %w",
					itemIndex,
					dynamoField.attributeName,
					err,
				)
			}
		}

		response = append(response, destination)
	}

	return response, nil

}

type dynamoField struct {
	index         int
	attributeName string
}

func getDynamoFields(structType reflect.Type) ([]dynamoField, error) {
	fields := make([]dynamoField, 0, structType.NumField())

	for i := 0; i < structType.NumField(); i++ {
		fieldInfo := structType.Field(i)
		attributeName := fieldInfo.Tag.Get("dynamo")

		if attributeName == "-" || fieldInfo.PkgPath != "" {
			continue
		}

		if attributeName == "" {
			return nil, fmt.Errorf("el campo %q requiere un tag dynamo", fieldInfo.Name)
		}

		fields = append(fields, dynamoField{
			index:         i,
			attributeName: attributeName,
		})
	}

	return fields, nil
}

func setDynamoField(field reflect.Value, attributeValue types.AttributeValue) error {
	switch field.Kind() {
	case reflect.String:
		value, ok := attributeValue.(*types.AttributeValueMemberS)
		if !ok {
			return errors.New("se esperaba un atributo Dynamo de tipo S")
		}
		field.SetString(value.Value)
		return nil

	case reflect.Bool:
		value, ok := attributeValue.(*types.AttributeValueMemberBOOL)
		if !ok {
			return errors.New("se esperaba un atributo Dynamo de tipo BOOL")
		}
		field.SetBool(value.Value)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, ok := attributeValue.(*types.AttributeValueMemberN)
		if !ok {
			return errors.New("se esperaba un atributo Dynamo de tipo N")
		}
		number, err := strconv.ParseInt(value.Value, 10, 64)
		if err != nil {
			return fmt.Errorf("número inválido: %w", err)
		}
		field.SetInt(number)
		return nil

	case reflect.Float32, reflect.Float64:
		value, ok := attributeValue.(*types.AttributeValueMemberN)
		if !ok {
			return errors.New("se esperaba un atributo Dynamo de tipo N")
		}
		number, err := strconv.ParseFloat(value.Value, 64)
		if err != nil {
			return fmt.Errorf("número inválido: %w", err)
		}
		field.SetFloat(number)
		return nil

	case reflect.Ptr:
		timeType := reflect.TypeOf(time.Time{})
		if field.Type().Elem() != timeType {
			return fmt.Errorf("puntero no soportado: %s", field.Type())
		}
		value, ok := attributeValue.(*types.AttributeValueMemberS)
		if !ok {
			return errors.New("para fecha se esperaba un atributo Dynamo de tipo S")
		}
		parsedTime, err := time.Parse(time.RFC3339, value.Value)
		if err != nil {
			return fmt.Errorf("fecha inválida: %w", err)
		}
		field.Set(reflect.ValueOf(&parsedTime))
		return nil

	default:
		return fmt.Errorf("tipo de campo no soportado: %s", field.Type())
	}
}
