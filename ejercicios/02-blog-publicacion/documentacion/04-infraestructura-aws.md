# 4. Infraestructura AWS

## Recursos a levantar

| Recurso | Nombre sugerido | Configuración esencial |
|---|---|---|
| Tabla principal | `author_post` | PK/SK String, GSI `draft_gsi`, Stream `NEW_AND_OLD_IMAGES` |
| Progreso | `FanoutProgress` | PK/SK String |
| Cola de fanout | `fanout-jobs` + DLQ | Standard y redrive policy |
| Cola de notificaciones | `notification-jobs` + DLQ | Standard y redrive policy |
| Funciones | cuatro Lambdas actuales | runtime `provided.al2023` y arquitectura del binario |
| API | HTTP API Gateway | rutas create y get |

## Índices que debes crear en DynamoDB

La tabla principal `author_post` usa `PK` (String) como partition key y `SK` (String) como sort key. Agrega un único índice secundario global:

| Índice | Tipo | Partition key | Sort key | Proyección |
|---|---|---|---|---|
| `draft_gsi` | GSI | `GSI1PK` (String) | `GSI1SK` (String) | `ALL` |

Además, activa DynamoDB Streams en `author_post` con la vista `NEW_AND_OLD_IMAGES`. La tabla `FanoutProgress` no lleva GSI: usa `PK` (String) y `SK` (String) como su clave primaria.

## Qué aporta cada servicio

**DynamoDB** guarda datos por patrón de acceso y Stream observa cambios. **Lambda** ejecuta pequeñas unidades de lógica sin servidores. **SQS** desacopla, persiste trabajo durante fallos y deriva casos agotados a una DLQ. **API Gateway** expone HTTP. **IAM** limita permisos y **CloudWatch** concentra logs, métricas y alarmas.

## Variables de entorno

| Lambda | Variables |
|---|---|
| create / get | `TABLENAME=author_post` |
| fanout-starter | `FANOUT_PROGRESS_TABLE`, `FANOUT_JOBS_QUEUE_URL` |
| worker | `FOLLOWERS_TABLE=author_post`, `FANOUT_PROGRESS_TABLE`, `FANOUT_JOBS_QUEUE_URL`, `NOTIFICATION_JOBS_QUEUE_URL` |

## IAM mínimo

| Rol | Acciones necesarias |
|---|---|
| create | `dynamodb:PutItem` en `author_post` |
| get | `dynamodb:Query` en tabla e índice |
| starter | lectura del Stream, `PutItem`/`UpdateItem` en progreso, `sqs:SendMessage` en fanout |
| worker | trigger SQS, `Query` followers, `TransactWriteItems` progreso, `SendMessage` a ambas colas |
| writer futuro | trigger SQS y `PutItem` en Inbox/Notifications |

Limita las políticas por ARN de tabla, índice, Stream y cola; no uses `Resource: "*"`. El visibility timeout de `fanout-jobs` debe superar el timeout del worker. Empieza con batch de 1 a 10 mensajes y concurrencia reservada para proteger DynamoDB.

## Capturas útiles

Las evidencias de consola deben mostrar: tabla/GSI/Stream, ambas colas y sus DLQ, roles y variables Lambda, event source mappings, `META` completado y mensajes en `notification-jobs`.
