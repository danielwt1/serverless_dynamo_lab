# 4. Infraestructura AWS

## Recursos

| Recurso | Nombre sugerido | Configuración |
|---|---|---|
| DynamoDB | `users` | PK `PK`, SK `SK`, GSI `GSI1` con `GSI1PK/GSI1SK`, pago por demanda |
| Lambdas | create-user, authenticate-user, list-active-users | runtime/imagen según Dockerfile; `USERS_TABLE_NAME=users` en create-user y `TABLE_NAME=users` en las otras dos |
| API Gateway | HTTP API | POST `/users`, POST `/auth/login`, GET `/users/active` |
| IAM | un rol por Lambda | mínimo privilegio |
| CloudWatch | logs, métricas y alarmas | retención definida para logs |

## Índices que debes crear en DynamoDB

Al crear la tabla `users` en consola, primero define la clave primaria de la tabla y luego agrega exactamente este índice secundario global:

| Índice | Tipo | Partition key | Sort key | Proyección |
|---|---|---|---|---|
| `GSI1` | GSI | `GSI1PK` (String) | `GSI1SK` (String) | `INCLUDE`: `userId`, `name`, `lastName`, `email`, `state`, `created_at` |

La tabla base usa `PK` (String) como partition key y `SK` (String) como sort key. No hay más GSIs ni LSIs en este lab.

## Qué aporta cada servicio

DynamoDB resuelve lectura por email y listado activo mediante claves/GSI. Lambda encapsula cada caso de uso. API Gateway expone HTTP y transforma la petición en el evento de handler. IAM autoriza únicamente cada acción necesaria. CloudWatch permite investigar 4xx/5xx, throttling y latencia.

## IAM mínimo

| Lambda | Acciones DynamoDB |
|---|---|
| create-user | `dynamodb:TransactWriteItems` en `users` |
| authenticate-user | `dynamodb:GetItem` en `users` |
| list-active-users | `dynamodb:Query` en tabla/índice `GSI1` |

Añade los permisos administrados básicos de logging de Lambda. Evita `Resource: "*"` en políticas propias: usa ARN de tabla e índice.

## Configuración importante

El GSI `GSI1` debe proyectar los atributos que `list-active-users` devuelve: `userId`, `name`, `lastName`, `email`, `state` y `created_at`. La proyección evita una lectura adicional de cada perfil. No se requieren Streams, SQS ni DLQ en el alcance actual.

Antes de producción, resuelve la deuda de passwords y alinea login 401/estado inactivo descritos en las guías 1 a 3.
