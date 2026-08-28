# 2. Implementación de Lambdas

## Estructura usada

Cada Lambda es un módulo Go independiente con `main.go`, `Dockerfile`, evento de ejemplo, tests y capas `domain`, `application`, puertos, handler e infraestructura. El recorrido es:

```text
API Gateway -> handler -> caso de uso -> puerto -> adaptador DynamoDB
```

El adaptador se programa contra una interfaz pequeña del SDK; los tests usan stubs y prueban reglas sin depender de AWS.

## `create-user`

Recibe nombre, apellido, email, password y fecha de nacimiento. El handler traduce JSON y responde 201, 400, 409 o 500; el adaptador normaliza el correo, genera UUID/timestamp y ejecuta la transacción de reserva + perfil. La condición de la reserva se traduce a `ErrUserAlreadyExists`.

Mejora pendiente: validar campos antes de llamar al caso de uso y reemplazar la contraseña de ejemplo por un hash.

## `authenticate-user`

Busca `EMAIL#<correo>/UNIQUE` con `GetItem` consistente y compara credenciales en el caso de uso. Devuelve el `userId`; no emite JWT todavía. Para evitar enumeración de cuentas, en una API pública conviene responder 401 uniforme para credenciales inválidas; el handler actual responde 404 y debe ajustarse si se expone a internet.

## `list-active-users`

Valida `limit` entre 1 y 100, consulta `GSI1PK=ACTIVE_USERS`, ordena descendente y traduce `LastEvaluatedKey` a `nextCursor`. La respuesta expone solo campos seguros del perfil, no password.

## Ejecución local y pruebas

Desde la raíz del laboratorio:

```bash
cd ejercicios/01-usuarios
Consulta la guía manual de AWS CLI en `documentacion/comandos-aws-cli.md` para crear la tabla en AWS. Este laboratorio ya no incluye scripts locales.
```

Variables: `USERS_TABLE_NAME=users` para create-user, `TABLE_NAME=users` para authenticate-user y list-active-users; además `AWS_REGION=us-east-1` y, donde el adaptador local lo soporte, `DYNAMODB_ENDPOINT=http://localhost:8000`. Ejecuta `go test ./...` dentro de la carpeta de cada Lambda; no comparten `go.mod`.
