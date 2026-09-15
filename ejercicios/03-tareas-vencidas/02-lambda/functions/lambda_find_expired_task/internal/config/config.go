package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	TaskTableName    string
	ProcessTableName string
	OverdueTopicARN  string
}

func Load() (Config, error) {
	config := Config{
		TaskTableName:    strings.TrimSpace(os.Getenv("TASK_TABLE_NAME")),
		ProcessTableName: strings.TrimSpace(os.Getenv("PROCESS_TABLE_NAME")),
		OverdueTopicARN:  strings.TrimSpace(os.Getenv("TASK_OVERDUE_TOPIC_ARN")),
	}
	missing := make([]string, 0, 3)
	if config.TaskTableName == "" {
		missing = append(missing, "TASK_TABLE_NAME")
	}
	if config.ProcessTableName == "" {
		missing = append(missing, "PROCESS_TABLE_NAME")
	}
	if config.OverdueTopicARN == "" {
		missing = append(missing, "TASK_OVERDUE_TOPIC_ARN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("faltan variables de entorno: %s", strings.Join(missing, ", "))
	}
	return config, nil
}
