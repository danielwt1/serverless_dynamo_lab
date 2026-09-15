package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("TASK_TABLE_NAME", " task_data ")
	t.Setenv("PROCESS_TABLE_NAME", "batch_notification")
	t.Setenv("TASK_OVERDUE_TOPIC_ARN", "arn:aws:sns:us-east-1:123456789012:task-overdue")

	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.TaskTableName != "task_data" || config.ProcessTableName != "batch_notification" || config.OverdueTopicARN == "" {
		t.Fatalf("configuración incorrecta: %+v", config)
	}
}

func TestLoadReportsAllMissingVariables(t *testing.T) {
	t.Setenv("TASK_TABLE_NAME", "")
	t.Setenv("PROCESS_TABLE_NAME", "")
	t.Setenv("TASK_OVERDUE_TOPIC_ARN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("se esperaba error")
	}
	for _, variable := range []string{"TASK_TABLE_NAME", "PROCESS_TABLE_NAME", "TASK_OVERDUE_TOPIC_ARN"} {
		if !strings.Contains(err.Error(), variable) {
			t.Fatalf("el error no incluye %s: %v", variable, err)
		}
	}
}
