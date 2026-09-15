package config

import (
	"errors"
	"os"
	"strings"
)

func TaskTableName() (string, error) {
	name := strings.TrimSpace(os.Getenv("TASK_TABLE_NAME"))
	if name == "" {
		return "", errors.New("TASK_TABLE_NAME es obligatorio")
	}
	return name, nil
}
