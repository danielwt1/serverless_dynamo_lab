# `lambda_find_expired_task`: proceso y recuperación

## Decisión

No hay lock ni lease en DynamoDB. Esta Lambda usa
`ReservedConcurrentExecutions = 1`: AWS nunca ejecuta dos invocaciones activas
de `lambda_find_expired_task` al mismo tiempo.

```text
EventBridge Scheduler
        |
        v
lambda_find_expired_task (ReservedConcurrency = 1)
        |
        +-- worker Go: shard 00
        +-- worker Go: shard 01
        '-- worker Go: shard 05
```

Los workers sí consultan shards en paralelo, pero todos pertenecen a una sola
Lambda. Si Scheduler dispara otra invocación mientras una sigue activa, Lambda
la limita/throttles; el trigger debe tener reintentos y DLQ configurados.

DynamoDB persiste progreso e idempotencia, no coordinación: batch abierto,
cursor por shard y eventos pendientes de SNS.

## Items en `batch_notification`

### `BatchRun` (`META`)

```text
PK             = OVERDUE_RUN#<execution_time>
SK             = META
entity_type    = BATCH_RUN
status         = RUNNING | PUBLISHING | COMPLETED | FAILED_RETRYABLE
execution_time = <UTC RFC3339>
shard_count    = 6
last_error     = <opcional>

# Solo mientras el batch está abierto
GSI1PK         = PROCESS#OVERDUE
GSI1SK         = OPEN#<updated_at>#<execution_time>
```

`GSI1PK` y `GSI1SK` solo existen mientras el batch no terminó. Así la Lambda
puede consultar si hay una corrida pendiente. Al llegar a `COMPLETED`, elimina
ambos atributos y el batch desaparece del índice. La concurrencia reservada
impide que existan dos batches abiertos creados por esta Lambda.

### `ShardCheckpoint`

```text
PK                = OVERDUE_RUN#<execution_time>
SK                = SHARD#<00..05>
entity_type       = SHARD_CHECKPOINT
status            = PENDING | IN_PROGRESS | COMPLETED
cursor            = <LastEvaluatedKey serializado>
pages_completed   = <número>
updated_at        = <UTC RFC3339>
```

Indica desde qué página continúa el shard; no bloquea nada.

### `BatchEvent` (outbox)

```text
PK                = OVERDUE_RUN#<execution_time>
SK                = EVENT#<dueAt>#<task_id>
entity_type       = BATCH_EVENT
event_id          = <task_id>#<dueAt>
status            = READY_TO_PUBLISH | PUBLISHED
```

### `NotificationMarker` (idempotencia)

```text
PK                = TASK_OVERDUE#<task_id>#<dueAt>
SK                = EVENT#TASK_OVERDUE
entity_type       = NOTIFICATION_MARKER
event_id          = <task_id>#<dueAt>
status            = RESERVED | PUBLISHED
batch_run_id      = OVERDUE_RUN#<execution_time>
```

## Al iniciar una Lambda

La Lambda consulta el índice con `GSI1PK = PROCESS#OVERDUE`.

```text
¿Hay BatchRun abierto?
├─ Sí: RUNNING | PUBLISHING | FAILED_RETRYABLE
│  └─ Retoma ese mismo run_id. No crea batch nuevo.
└─ No
   └─ Crea BatchRun nuevo con el execution_time del trigger,
      seis checkpoints PENDING y las claves GSI1 OPEN.
```

No hay carrera entre buscar y crear porque AWS no deja dos ejecuciones activas
de esta función. Solo esta Lambda debe tener permisos para escribir los items
del proceso.

```go
openRun := repository.FindOpenBatch(ctx)
if openRun != nil {
    runID = openRun.ID
} else {
    runID = "OVERDUE_RUN#" + triggerExecutionTime
    repository.CreateRunWithCheckpoints(ctx, runID)
}
```

## Consulta de shards y recuperación

Por cada página del shard: consulta tareas con el cursor, persiste markers y
`BatchEvent`, y solo después guarda el cursor siguiente. Si no hay
`LastEvaluatedKey`, marca el shard `COMPLETED`.

Si falla un worker o vence un deadline interno menor que el timeout de Lambda:

1. Cancela los workers.
2. Marca `META` como `FAILED_RETRYABLE` y guarda `last_error`.
3. No confirma el cursor de una página incompleta.
4. Retorna error para que actúen los reintentos configurados.

La siguiente Lambda encuentra el mismo batch abierto y aplica:

| Checkpoint | Acción |
| --- | --- |
| `COMPLETED` | Lo omite. |
| `PENDING` | Inicia con cursor vacío. |
| `IN_PROGRESS` | Repite desde el último cursor confirmado. |

Ejemplo: si `#00`, `#01` y `#03` terminaron, pero `#02` quedó pegado, la
siguiente Lambda solo trabaja `#02`, `#04` y `#05`.

## Idempotencia, SNS y cierre

Para una tarea vencida se hace un `TransactWrite`: crea el marker `RESERVED` y
el `BatchEvent READY_TO_PUBLISH`. Si el marker ya está `PUBLISHED`, se omite;
si está `RESERVED` para este mismo `run_id`, la página se está repitiendo y se
continúa.

Cuando los seis checkpoints están `COMPLETED`, el flujo es:

```text
META PUBLISHING -> Query READY_TO_PUBLISH -> SNS PublishBatch (máx. 10)
-> BatchEvent y Marker PUBLISHED -> META COMPLETED
-> eliminar GSI1PK/GSI1SK OPEN
```

Al eliminar las claves GSI el cron siguiente crea un batch nuevo. Si SNS falla,
el batch queda `FAILED_RETRYABLE`. SNS entrega al menos una vez, por eso el
consumidor deduplica usando `event_id`.
