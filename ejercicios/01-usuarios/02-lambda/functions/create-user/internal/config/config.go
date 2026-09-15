package config

import (
	"fmt"
	"os"
)

// Config contiene la configuracion propia de la aplicacion.
// AWS_REGION y las credenciales pertenecen al SDK; el nombre de la tabla
// pertenece a esta Lambda y se entrega por variable de entorno.
type Config struct {
	UsersTableName string
}

func Load() (Config, error) {
	tableName := os.Getenv("USERS_TABLE_NAME")
	if tableName == "" {
		return Config{}, fmt.Errorf("USERS_TABLE_NAME is required")
	}

	return Config{UsersTableName: tableName}, nil
}
