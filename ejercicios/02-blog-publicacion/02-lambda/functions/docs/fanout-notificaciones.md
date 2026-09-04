# Guía paso a paso: publicar un post y notificar a sus seguidores

Esta guía explica el flujo completo con palabras simples. Describe lo que está implementado en:

- fanout_starter: detecta que un post se publicó y crea el trabajo inicial.
- follower_fanout_worker: busca seguidores poco a poco y crea los trabajos de notificación.

La Lambda que entrega el correo o push final no está en esta carpeta. Aun así, se explica la regla que debe cumplir para evitar notificaciones repetidas.

## 1. El problema, explicado simple

Un autor puede tener dos seguidores o dos millones. La petición que publica el post no debe intentar avisarles a todos: tardaría demasiado y, si falla a mitad, no sabríamos a quién se notificó.

Por eso el trabajo se parte:

1. Detectar que el post quedó publicado.
2. Guardar un registro de control en DynamoDB.
3. Enviar una orden a SQS.
4. Buscar seguidores de 100 en 100.
5. Enviar trabajos de notificación de hasta 50 destinatarios.
6. Guardar el punto exacto hasta donde se llegó.

La regla del diseño es:

> El proceso puede repetirse tras un fallo, pero dos workers no deben recorrer al mismo tiempo los seguidores del mismo evento.

## 2. Palabras de esta guía

| Palabra | Significado |
| --- | --- |
| Evento del Stream | Registro emitido por DynamoDB al insertar o modificar un post. Tiene un eventId único. |
| Fanout | Recorrer los seguidores y preparar notificaciones para ellos. |
| META | Ítem de DynamoDB que guarda el estado y avance de un fanout. No es el post. |
| Cursor | Marca que indica desde cuál seguidor debe continuar la consulta. |
| Página | Hasta 100 seguidores devueltos por una consulta. |
| Lote | Hasta 50 destinatarios dentro de un mensaje de notificación. |
| Checkpoint | Cursor guardado de forma duradera después de terminar una página. |
| Lease | Reserva temporal que da permiso a un solo worker para avanzar un fanout. |
| Mensaje duplicado | El mismo trabajo llega más de una vez. Puede ocurrir con SQS y Streams. |

## 3. Mapa del proceso

~~~mermaid
flowchart LR
    A[Post en DynamoDB] -->|cambio de publicación| B[DynamoDB Streams]
    B --> C[fanout_starter]
    C --> D[META: tabla de progreso]
    C --> E[Cola SQS de fanout]
    E --> F[follower_fanout_worker]
    F --> G[Consulta seguidores: páginas de 100]
    F --> H[Cola SQS de notificación: lotes de 50]
    F --> D
    H --> I[Consumidor final: correo o push]
~~~

Hay dos colas porque los trabajos son distintos:

| Cola | Contiene | La consume |
| --- | --- | --- |
| Fanout | “Busca seguidores de este autor para este post”. | follower_fanout_worker |
| Notificación | “Notifica este post a estas personas”. | Consumidor final |

## 4. Cuándo empieza el fanout

El starter sólo acepta dos situaciones:

| Evento | Estado antes | Estado después | ¿Inicia el proceso? |
| --- | --- | --- | --- |
| INSERT | No existía | PUBLISHED | Sí |
| MODIFY | DRAFT | PUBLISHED | Sí |
| INSERT | No existía | DRAFT | No |
| MODIFY | DRAFT | DRAFT | No |
| MODIFY | PUBLISHED | PUBLISHED | No |

Ejemplos:

~~~text
Crear post-900 directamente como PUBLISHED:
INSERT + PUBLISHED -> sí se notifica.

Crear post-900 como DRAFT y después publicarlo:
MODIFY + DRAFT -> PUBLISHED -> sí se notifica.

Cambiar sólo el título de un post ya publicado:
MODIFY + PUBLISHED -> PUBLISHED -> no se vuelve a notificar.
~~~

## 5. La identidad estable del proceso

Cada evento de DynamoDB Streams tiene un eventId. El sistema usa ese mismo valor como identidad del fanout:

~~~text
eventId  = E-2026-00017
fanoutId = E-2026-00017
~~~

No se genera un UUID nuevo. Así, si el starter se reintenta, sabe cuál es el mismo proceso y no crea otro.

También existe un identificador de mensaje:

~~~text
Primer trabajo:          E-2026-00017#INITIAL
Trabajo de continuación: E-2026-00017#CURSOR#<hash-del-cursor>
~~~

| Campo | Pregunta que responde |
| --- | --- |
| fanoutId | ¿Cuál es el proceso completo? |
| jobId | ¿Cuál es este mensaje concreto dentro del proceso? |

## 6. META: el lugar donde se guarda el avance

Por cada evento se crea un solo ítem de control:

~~~text
PK = FANOUT#STREAM_EVENT#E-2026-00017
SK = META
~~~

Ejemplo justo después de iniciar:

~~~json
{
  "PK": "FANOUT#STREAM_EVENT#E-2026-00017",
  "SK": "META",
  "fanout_id": "E-2026-00017",
  "stream_event_id": "E-2026-00017",
  "post_id": "post-900",
  "author_id": "author-42",
  "status": "PROCESSING",
  "initial_job_enqueued": false,
  "next_cursor": "",
  "processed_followers": 0,
  "batches_enqueued": 0,
  "notification_jobs_enqueued": 0,
  "started_at": "2026-08-29T15:00:00Z",
  "updated_at": "2026-08-29T15:00:00Z"
}
~~~

| Campo | Uso |
| --- | --- |
| status | PROCESSING mientras falta trabajo; FANOUT_COMPLETED al terminar. |
| initial_job_enqueued | Marca que el starter logró guardar el primer envío a SQS. |
| next_cursor | Punto desde donde seguirá la próxima página. Es la fuente de verdad del avance. |
| Contadores | Permiten observar cuántos seguidores y lotes se procesaron. |
| fanout_id | Identidad estable: el mismo valor que eventId. |

## 7. Parte por parte: qué hace el starter

### 7.1 Valida el evento

Primero comprueba las reglas de la sección 4. Si no es una publicación válida, termina sin crear nada.

También exige post_id y author_id. Sin esos datos no podría saber qué post notificar ni qué lista de seguidores consultar. Si faltan, devuelve error para que DynamoDB Streams pueda reintentar.

### 7.2 Intenta crear META una sola vez

Hace un PutItem con esta regla:

~~~text
Crear META sólo si esa PK y esa SK no existen todavía.
~~~

| Resultado | Qué significa | Siguiente acción |
| --- | --- | --- |
| META creado | Es la primera entrega del evento. | Mandar el trabajo inicial. |
| META ya existía | Streams volvió a entregar el mismo evento. | Leer initial_job_enqueued. |

Si initial_job_enqueued es true, termina. Si es falso o no existe, manda de nuevo el mismo trabajo lógico. Reintenta con el mismo fanoutId y jobId; nunca crea otro fanout.

### 7.3 Envía el primer mensaje a SQS

~~~json
{
  "eventId": "E-2026-00017",
  "fanoutId": "E-2026-00017",
  "jobId": "E-2026-00017#INITIAL",
  "streamEventId": "E-2026-00017",
  "postId": "post-900",
  "authorId": "author-42",
  "cursor": "",
  "batchNumber": 1,
  "createdAt": "2026-08-29T15:00:01Z"
}
~~~

El cursor del mensaje inicial está vacío. En un reintento, el worker no confía en ese valor: vuelve a leer META y toma el next_cursor allí guardado.

### 7.4 Marca el envío inicial

Después de que SQS responde correctamente, actualiza:

~~~text
initial_job_enqueued = true
~~~

SQS y DynamoDB son servicios separados. No hay una operación única que escriba en ambos. La consecuencia se explica con detalle en la sección 13.

## 8. Parte por parte: qué es el lease

SQS puede entregar el mismo mensaje más de una vez. También pueden existir dos mensajes iniciales si SQS aceptó el primero pero falló la actualización de initial_job_enqueued.

Sin protección, dos Lambdas podrían leer el mismo cursor y enviar los mismos lotes a la vez. El lease evita eso.

Antes de buscar seguidores, el worker intenta escribir:

~~~json
{
  "lease_owner": "aws-request-id:sqs-message-id",
  "lease_acquired_at": "2026-08-29T15:00:03Z",
  "lease_expires_at": 1788015723,
  "lease_version": 1
}
~~~

lease_expires_at es una hora de vencimiento en segundos. No mata la Lambda. Sólo permite que otra ejecución recupere el trabajo si el dueño deja de renovarlo.

### 8.1 La regla para tomar el lease

El worker escribe el lease solamente si:

~~~text
status no es FANOUT_COMPLETED
Y
no existe lease_expires_at, o ya está vencido
~~~

~~~mermaid
flowchart TD
    A[Worker recibe FanoutJob] --> B[Lee META]
    B --> C{Terminó?}
    C -->|Sí| D[Termina: mensaje repetido]
    C -->|No| E[Intentar tomar lease con condición]
    E --> F{Ganó?}
    F -->|Sí| G[Puede consultar seguidores]
    F -->|No| H[Otro worker trabaja: devolver fallo parcial a SQS]
~~~

Dos workers pueden leer META casi al mismo tiempo. Aun así, sólo uno gana la escritura condicional. El perdedor no consulta followers ni publica lotes; SQS lo volverá a intentar después.

### 8.2 Renovación y liberación

El worker renueva el lease antes de cada página. La duración está en FANOUT_LEASE_SECONDS, con 120 segundos por defecto.

Cuando necesita crear una continuación, primero manda el siguiente mensaje a SQS. Sólo cuando SQS lo acepta libera el lease. Si falla ese envío, conserva el lease hasta su vencimiento; otro intento retoma desde el último checkpoint seguro.

Al terminar todo el fanout, cambia el estado a FANOUT_COMPLETED. En ese momento cualquier lease restante deja de importar: el estado terminado gana.

## 9. Parte por parte: cómo se buscan los seguidores

Después de tomar el lease, el worker lee META una vez más y usa next_cursor como punto de partida.

Consulta hasta 100 seguidores:

~~~text
PK = FOLLOWING#author-42
SK empieza por FOLLOWER#
Limit = 100
~~~

Para 230 seguidores el recorrido es:

| Página | Seguidores | Cursor guardado después |
| --- | --- | --- |
| 1 | user-001 a user-100 | Inicio de user-101 |
| 2 | user-101 a user-200 | Inicio de user-201 |
| 3 | user-201 a user-230 | Vacío: ya no hay más |

## 10. Parte por parte: cómo se crean los lotes de notificación

Una página puede traer 100 seguidores, pero se divide en lotes de hasta 50. Para la primera página se producen dos mensajes:

~~~json
{
  "eventId": "E-2026-00017",
  "fanoutBatchId": "E-2026-00017#CURSOR#abc123#PART#01",
  "postId": "post-900",
  "authorId": "author-42",
  "recipientIds": ["user-001", "...", "user-050"]
}
~~~

~~~json
{
  "eventId": "E-2026-00017",
  "fanoutBatchId": "E-2026-00017#CURSOR#abc123#PART#02",
  "postId": "post-900",
  "authorId": "author-42",
  "recipientIds": ["user-051", "...", "user-100"]
}
~~~

El fanoutBatchId identifica la página y la parte. Ayuda a rastrear un lote concreto.

## 11. Parte por parte: el checkpoint

Después de enviar todos los lotes de una página, el worker guarda el checkpoint. Lo hace con una transacción DynamoDB que:

1. Crea un registro BATCH por cada lote enviado.
2. Actualiza META: cursor siguiente, contadores y estado.
3. Exige que lease_owner continúe siendo este worker.

Ejemplo al completar seguidores 1–100:

~~~json
{
  "status": "PROCESSING",
  "next_cursor": "<cursor-para-user-101>",
  "processed_followers": 100,
  "batches_enqueued": 2,
  "notification_jobs_enqueued": 2,
  "lease_owner": "request-A:message-1"
}
~~~

La regla es:

> El cursor avanza sólo después de enviar todos los lotes de esa página y de guardar correctamente su checkpoint.

Si el worker ya perdió el lease, la condición falla. No puede mover el cursor de un proceso que ya pertenece a otra ejecución; el mensaje SQS se reintenta.

## 12. Fin o continuación

Una invocación procesa como máximo 1.000 seguidores, normalmente diez páginas.

| Situación | Qué hace |
| --- | --- |
| No queda cursor después de una página | Guarda FANOUT_COMPLETED y termina. |
| Llegó a 1.000 pero hay cursor | Envía un FanoutJob de continuación, luego libera el lease. |

La continuación conserva el mismo fanoutId. Sólo cambia jobId, porque es otro mensaje del mismo proceso.

## 13. Super ejemplo completo: 2.030 seguidores

Datos de partida:

~~~text
postId = post-900
authorId = author-42
eventId = E-2026-00017
seguidores = 2.030
página = 100
lote = 50
límite por ejecución = 1.000
~~~

### Parte A: publicación

1. Se crea post-900 directamente con status=PUBLISHED.
2. DynamoDB Streams entrega INSERT y eventId=E-2026-00017.
3. Starter crea META en PROCESSING con cursor vacío.
4. Starter manda E-2026-00017#INITIAL a la cola de fanout.
5. Starter marca initial_job_enqueued=true.

Estado:

~~~text
META: PROCESSING, next_cursor="", processed_followers=0
Cola de fanout: un mensaje INITIAL
~~~

### Parte B: worker A, seguidores 1–1.000

6. Worker A recibe el mensaje y toma el lease.
7. Lee seguidores 1–100, manda dos lotes y guarda cursor 101.
8. Repite el patrón con 101–200, 201–300, hasta 901–1.000.
9. META ahora dice processed_followers=1000 y apunta al seguidor 1.001.
10. Worker A manda una continuación a SQS y libera su lease.

Resultado parcial:

~~~text
20 lotes de notificación
META continúa PROCESSING
next_cursor apunta al seguidor 1.001
~~~

### Parte C: worker B, seguidores 1.001–2.000

11. Worker B recibe la continuación.
12. Lee META y toma el cursor 1.001; no usa un cursor viejo como fuente de verdad.
13. Toma el lease, procesa diez páginas y genera veinte lotes.
14. Aún faltan 30 seguidores, por lo que manda otra continuación y libera lease.

### Parte D: worker C, últimos 30 seguidores

15. Worker C toma lease.
16. Lee 2.001–2.030 y crea un lote de 30.
17. Como no hay siguiente cursor, el checkpoint cambia META a FANOUT_COMPLETED.
18. Un mensaje viejo que llegue después sólo lee ese estado y termina.

Resultado final:

~~~text
2.030 seguidores
41 lotes de notificación
3 invocaciones worker
1 META con todo el avance
~~~

## 14. Fallos: qué ocurre exactamente

| Fallo | Qué queda guardado | Recuperación | ¿Puede haber repetición? |
| --- | --- | --- | --- |
| Falla crear META | META no existe. | Streams reintenta starter. | No. |
| SQS rechaza job inicial | META existe sin marca. | Starter reintenta mismo job. | No hay mensaje aceptado. |
| SQS acepta job inicial y falla la marca posterior | Puede haber mensaje, META aún sin marca. | Starter reenvía el mismo jobId. | Sí, dos mensajes; lease evita dos recorridos paralelos. |
| Llega un duplicado mientras hay lease vigente | Lease sigue en META. | El segundo worker devuelve fallo parcial a SQS. | No procesa en paralelo. |
| Worker muere antes del checkpoint | META conserva cursor anterior. | Al vencer lease se repite desde esa página. | Sí, pueden repetirse lotes de esa página. |
| Envía lotes pero falla checkpoint | Cursor anterior permanece. | Se vuelve a enviar la página. | Sí. |
| Pierde lease antes de checkpoint | Condición de DynamoDB falla. | SQS reintenta. | No mueve cursor ajeno. |
| Falla enviar continuación | Checkpoint de 1.000 ya existe, lease queda hasta vencer. | Reintento inicia desde cursor guardado. | Puede repetirse el intento de continuación. |

### Caso clave: SQS aceptó, pero DynamoDB falló después

~~~mermaid
sequenceDiagram
    participant S as Starter
    participant D as META DynamoDB
    participant Q as SQS fanout
    participant A as Worker A
    participant B as Worker B
    S->>D: Crear META
    S->>Q: Enviar E-900#INITIAL
    Q-->>S: Aceptado
    S->>D: Marcar initial_job_enqueued=true
    D-->>S: Falla
    Note over S: Streams reintenta Starter
    S->>Q: Reenvía E-900#INITIAL
    A->>D: Toma lease
    B->>D: Intenta lease y pierde
    Note over B: No procesa; SQS lo reintentará
    A->>D: Guarda checkpoints y completa
    B->>D: Reintento ve FANOUT_COMPLETED
~~~

No se puede hacer atómica una llamada a SQS y otra a DynamoDB. La solución no pretende eliminar el mensaje duplicado: hace que el duplicado no produzca dos recorridos a la vez.

## 15. Por qué el consumidor final también debe deduplicar

Existe este orden posible:

1. Worker envía un lote a SQS de notificaciones.
2. SQS acepta el lote.
3. Worker falla antes de guardar checkpoint.
4. Otro worker repite la página para no saltarse personas.

Por eso el consumidor final debe registrar, antes de mandar el correo o push:

~~~text
PK = NOTIFICATION#E-2026-00017
SK = RECIPIENT#user-001
Condición: esa llave aún no existe
~~~

| Resultado | Acción |
| --- | --- |
| La llave se crea | Enviar notificación real. |
| La llave ya existía | No enviar: es una repetición. |

El lease evita paralelismo en el recorrido. Esta última condición evita que una persona vea dos notificaciones por una página recuperada.

## 16. Qué garantiza y qué no

### Sí garantiza

- Inicia fanout para INSERT más PUBLISHED y DRAFT a PUBLISHED.
- Reutiliza la misma identidad al reintentar starter.
- Sólo un worker puede avanzar un META mientras tenga el lease.
- Un worker viejo no puede guardar checkpoint tras perder lease.
- El proceso retoma desde el último cursor confirmado.
- Los trabajos repetidos después de completar no vuelven a buscar seguidores.

### No garantiza por sí solo

- Una transacción única entre DynamoDB y SQS.
- Que un lote nunca se repita tras una caída entre SQS y checkpoint.
- Que el proveedor final no reciba dos llamadas si su consumidor no deduplica.

El modelo es “al menos una vez”: se prefiere repetir de forma controlada antes que perder una notificación.

## 17. Configuración y operación

| Elemento | Recomendación |
| --- | --- |
| FANOUT_LEASE_SECONDS | 120 segundos por defecto; debe cubrir el peor tiempo de una página. |
| Visibilidad de SQS | Mayor que el máximo tiempo de ejecución del worker. |
| DLQ | Revisarla: muestra trabajos que no pudieron completar sus reintentos. |
| META en PROCESSING sin cambios | Revisar worker, lease, timeout y visibilidad de SQS. |
| Contadores META | Úsalos para observar avance y tamaño del fanout. |

## 18. Relación con el código

| Responsabilidad | Archivo |
| --- | --- |
| Detectar publicación válida | fanout_starter/internal/handler/dynamodb_stream.go |
| Crear META y marca inicial | fanout_starter/internal/infraestructure/progress_repository.go |
| Crear identidad y job inicial | fanout_starter/internal/application/start_fanout.go |
| Tomar, renovar y liberar lease | follower_fanout_worker/internal/infraestructure/progress_repository.go |
| Páginas, lotes, checkpoint y continuación | follower_fanout_worker/internal/application/process_fanout.go |
| Fallo parcial hacia SQS | follower_fanout_worker/internal/handler/sqs.go |

## 19. Resumen final

Cada publicación tiene una identidad fija. Un worker obtiene permiso temporal para recorrer seguidores. El avance se guarda página por página. Si hay una caída, se vuelve desde el último avance confirmado. Y el consumidor final bloquea cualquier repetición visible para el usuario.
