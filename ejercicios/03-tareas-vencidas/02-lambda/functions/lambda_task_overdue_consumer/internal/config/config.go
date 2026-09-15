package config

import (
	"errors"
	"os"
	"strings"
)

func ProcessTableName() (string, error) {
	name := strings.TrimSpace(os.Getenv("PROCESS_TABLE_NAME"))
	if name == "" {
		return "", errors.New("PROCESS_TABLE_NAME es obligatorio")
	}
	return name, nil
}
