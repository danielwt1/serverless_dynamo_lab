# Modelo DynamoDB explicado

## 1. Diseñar desde las consultas

DynamoDB necesita conocer primero las preguntas que debe responder. Este ejercicio implementa ocho access patterns:

| ID | Pregunta | Operación |
| --- | --- | --- |
| AP-01 | ¿Cómo creo una tarea y todas sus vistas? | Transacción de dos `Put` |
| AP-02 | ¿Cómo completo una tarea sin dejarla visible como pendiente? | `GetItem` fuerte + transacción |
| AP-03 | ¿Cuáles son todas las tareas de un dueño? | `Query` por dueño y prefijo `TASK_ID#` |
| AP-04 | ¿Cuáles son sus tareas pendientes ordenadas por vencimiento? | `Query` por dueño y prefijo `STATUS#PENDING#` |
| AP-05 | ¿Cuáles pendientes vencieron globalmente? | Una `Query` por shard en el GSI |
| AP-06 | ¿Hay una corrida abierta que deba retomarse? | `Query` a `process_gsi_open` |
| AP-07 | ¿Hasta dónde llegó cada shard y qué eventos faltan? | `GetItem`, `UpdateItem` y `Query` por `run_id` |
| AP-08 | ¿Cómo cierro la corrida sin perder publicaciones? | Estados del proceso y outbox persistente |

## 2. Tabla de tareas

La tabla tiene clave primaria compuesta:

```text
PK  string
SK  string
```

Todos los items de una persona comparten `PK = OWNER_ID#<owner_id>`.

### 2.1 Item principal `TASK`

Ejemplo conceptual:

```json
{
  "PK": "OWNER_ID#alice",
  "SK": "TASK_ID#6f1d...",
  "entity_type": "TASK",
  "task_id": "6f1d...",
  "owner_id": "alice",
  "description": "Pagar la factura",
  "created_at": "2026-09-14T15:00:00Z",
  "expired_at": "2026-09-15T13:00:00Z",
  "status": "PENDING",
  "GSI1PK": "SHARD#03#STATUS#PENDING",
  "GSI1SK": "EXPIRED_AT#2026-09-15T13:00:00Z#TASK_ID#6f1d..."
}
```

`SK = TASK_ID#...` permite listar items principales sin mezclar las proyecciones. El UUID hace única la tarea dentro de la partición.

### 2.2 Proyección pendiente

```json
{
  "PK": "OWNER_ID#alice",
  "SK": "STATUS#PENDING#EXPIRED_AT#2026-09-15T13:00:00Z#TASK_ID#6f1d...",
  "entity_type": "TASK_PENDING_PROJECTION",
  "task_id": "6f1d...",
  "owner_id": "alice",
  "description": "Pagar la factura",
  "created_at": "2026-09-14T15:00:00Z",
  "expired_at": "2026-09-15T13:00:00Z"
}
```

La fecha está dentro de `SK`, así que DynamoDB devuelve pendientes en orden de vencimiento. `task_id` rompe empates si dos tareas vencen al mismo instante. La proyección duplica los atributos que necesita la respuesta HTTP y evita otra lectura por tarea.

### 2.3 GSI global de pendientes

Nombre: `task_gsi_pending`.

```text
partition key = GSI1PK
sort key      = GSI1SK
projection    = ALL
```

La partición se elige de manera estable:

```text
shard = primer byte de SHA-256(task_id) módulo 6
GSI1PK = SHARD#<00..05>#STATUS#PENDING
```

El hash distribuye tareas entre seis particiones lógicas. El valor se calcula una vez al crear y no cambia.

Para una corrida con límite `2026-09-16T05:00:00Z`, cada shard usa:

```text
GSI1PK = SHARD#03#STATUS#PENDING
GSI1SK BETWEEN "EXPIRED_AT#"
           AND "EXPIRED_AT#2026-09-16T05:00:00Z#\uFFFF"
```

El prefijo inferior incluye todas las fechas. El sufijo alto `\uFFFF` incluye cualquier `task_id` que siga a una fecha exactamente igual al límite.

El índice es disperso: solo items con `GSI1PK` y `GSI1SK` aparecen. Al completar la tarea se remueven ambos atributos, por lo que desaparece del índice sin borrar su historial.

## 3. Tabla de procesos

Esta tabla guarda coordinación durable del batch. También usa `PK` y `SK`, tiene TTL en `ttl` y el GSI `process_gsi_open`.

### 3.1 `BatchRun` o `META`

```text
PK             = OVERDUE_RUN#<execution_time>
SK             = META
entity_type    = BATCH_RUN
status         = RUNNING | PUBLISHING | COMPLETED | FAILED_RETRYABLE
execution_time = fecha fija de la corrida
active_run_id  = igual a PK
shard_count    = 6
last_error     = último error o cadena vacía
updated_at     = fecha del cambio más reciente
GSI1PK         = PROCESS#OVERDUE            # mientras está abierto
GSI1SK         = OPEN#<updated_at>#<execution_time>
```

`process_gsi_open` es disperso. El cierre remueve sus dos claves. Consultar `GSI1PK = PROCESS#OVERDUE` encuentra la corrida recuperable sin conocer su fecha.

### 3.2 `ShardCheckpoint`

```text
PK              = OVERDUE_RUN#<execution_time>
SK              = SHARD#<00..05>
entity_type     = SHARD_CHECKPOINT
run_id          = igual a PK
shard_id        = 00..05
status          = PENDING | IN_PROGRESS | COMPLETED
cursor          = LastEvaluatedKey codificado o cadena vacía
pages_completed = contador informativo
updated_at      = última confirmación
ttl             = epoch seconds, siete días después de crear la corrida
```

El cursor representa la página ya confirmada. Guardarlo después de reservar todos los eventos de la página impide saltarse una tarea ante un fallo parcial.

### 3.3 `BatchEvent` u outbox

```text
PK             = OVERDUE_RUN#<execution_time>
SK             = EVENT#<expired_at>#<task_id>
entity_type    = BATCH_EVENT
status         = READY_TO_PUBLISH | PUBLISHED
event_id       = <task_id>#<expired_at>
task_id        = ...
owner_id       = ...
description    = ...
expired_at     = ...
execution_time = ...
published_at   = ...                      # después de publicar
```

Compartir `PK` con la corrida permite obtener todos sus eventos con `begins_with(SK, "EVENT#")`. La implementación aplica un filtro por `READY_TO_PUBLISH`; como `FilterExpression` se evalúa después de leer la página, el código debe seguir el cursor incluso cuando una página devuelva cero items filtrados.

### 3.4 `NotificationMarker`

```text
PK           = TASK_OVERDUE#<event_id>
SK           = EVENT#TASK_OVERDUE
entity_type  = NOTIFICATION_MARKER
event_id     = <task_id>#<expired_at>
status       = RESERVED | PUBLISHED
batch_run_id = corrida que lo reservó
published_at = ...                        # tras confirmar SNS
consumed_at  = ...                        # tras el primer consumo
consumer     = TASK_OVERDUE_LOG
```

El marker tiene identidad independiente de la corrida. Si una tarea aparece en otra corrida con el mismo `task_id` y `expired_at`, encuentra el mismo marker y no crea un segundo evento de negocio.

## 4. Escrituras atómicas

### Crear tarea

La transacción contiene:

1. `Put` condicional del item principal.
2. `Put` condicional de la proyección.

Ambos usan `attribute_not_exists(PK) AND attribute_not_exists(SK)`.

### Completar tarea

La transacción contiene:

1. `Update` del principal condicionado a `status = PENDING`.
2. `Delete` de la proyección pendiente calculada con su `expired_at` y `task_id`.

### Reservar evento

La transacción contiene:

1. `Put` condicional del marker `RESERVED`.
2. `Put` condicional del `BatchEvent READY_TO_PUBLISH`.

### Confirmar publicación

La transacción contiene:

1. Cambio del outbox de `READY_TO_PUBLISH` a `PUBLISHED`.
2. Cambio del marker de `RESERVED` a `PUBLISHED`, verificando `batch_run_id`.

Las transacciones mantienen consistentes los pares de items. SNS queda fuera de DynamoDB, por lo que se acepta una posible entrega repetida y se conserva un `event_id` estable para deduplicarla.

## 5. Cursores

Un cursor de DynamoDB no es un número de página. Es la clave completa del último item evaluado. Esto evita que inserciones o borrados obliguen a contar offsets.

Las consultas HTTP convierten valores string a un mapa JSON y luego usan Base64 URL-safe sin padding. El batch conserva además el tipo DynamoDB (`S`, `N` o `B`) antes de codificar. En ambos casos el cursor se trata como opaco fuera del adaptador.

## 6. Consistencia

- Las lecturas del item principal y de checkpoints usan `ConsistentRead=true` cuando consultan la tabla base.
- Las consultas al GSI son eventualmente consistentes por definición de DynamoDB.
- Las transacciones ofrecen atomicidad para las escrituras agrupadas.
- Las condiciones convierten carreras en fallos explícitos que el caso de uso puede tratar como idempotencia o conflicto.

## 7. Relación entre modelo y costo

Las consultas por dueño leen solo una partición y hasta 10 items por página. La consulta vencida lee seis rangos del GSI y hasta 25 items por página. El costo depende principalmente de tareas relevantes y páginas recorridas, no de todas las tareas históricas completadas.

La duplicación de la proyección aumenta escrituras y almacenamiento. Ese costo compra una lectura directa, ordenada y sin `Scan` para pendientes por dueño.
