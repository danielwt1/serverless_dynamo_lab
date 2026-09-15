# 03. Tareas vencidas

## Contexto

Una aplicación de productividad permite a cada persona crear y completar tareas con fecha límite. Operaciones necesita detectar tareas vencidas sin recorrer toda la tabla.

## Problema

Construye una API para crear, completar y consultar tareas. Una ejecución programada diaria debe identificar tareas pendientes vencidas y emitir un evento de negocio para que otros consumidores reaccionen.

## Reglas

- Una tarea pertenece a una sola persona.
- Una tarea completada no puede aparecer como pendiente ni vencida.
- La vista global de vencidas no puede usar `Scan`.
- Ejecutar dos veces el proceso diario no debe generar efectos de negocio duplicados.

## Criterios de aceptación

- Lista todas las tareas de una persona con paginación.
- Encuentra vencidas globales mediante una consulta dirigida.
- La ejecución programada publica `TaskOverdue` solo para el cambio que corresponda.

## Alcance de servicios

DynamoDB, API Gateway, Lambda, CloudWatch Logs y EventBridge Scheduler. Define tú qué Lambda publica el evento y qué consumidor inicial demuestra que el flujo funciona.

## Nivel de ayuda

Semi-guiado: documenta primero access patterns, claves, GSI y contrato de evento. No añadas índices ni servicios hasta justificar qué consulta o fallo resuelven.

## Estructura del ejercicio

```text
00-despliegue/             # Guías y artefactos del despliegue manual
01-dynamodb/               # Modelo y recursos de DynamoDB
02-lambda/functions/       # Una carpeta por función Lambda
03-api/contracts/          # Contratos OpenAPI
04-cdk/                     # Infraestructura como código con CDK
04-eventos/notificaciones/ # Eventos y notificaciones
05-resiliencia/            # Patrones de tolerancia a fallos
documentacion/policy/      # Documentación y políticas IAM de referencia
postman/                   # Colecciones y entornos de prueba HTTP
scripts/aws/               # Scripts operativos para AWS CLI
scripts/local/             # Scripts para pruebas locales
```

La guía de estudio conecta estos artefactos con el código implementado.

## Implementación

El ejercicio implementa los patrones AP-01 a AP-08 del modelo DynamoDB:

- `POST /tasks/{ownerId}` crea la tarea y su proyección pendiente en una transacción.
- `PATCH /tasks/{ownerId}/{taskId}/complete` completa la tarea, quita el GSI disperso y elimina la proyección.
- `GET /tasks/{ownerId}` y `GET /tasks/{ownerId}/pending` consultan páginas de 10 elementos sin `Scan`.
- EventBridge ejecuta diariamente la búsqueda de vencidas mediante seis shards.
- El batch persiste checkpoints y un outbox, publica `TaskOverdue` y un consumidor demuestra deduplicación mediante `event_id`.

La infraestructura desplegable está en `04-cdk`; el contrato HTTP está en `03-api/contracts/openapi.yaml`.

## Mapa de las Lambdas

| Lambda | Disparador | Objetivo |
| --- | --- | --- |
| [`lambda_create_task`](02-lambda/functions/lambda_create_task/README.md) | `POST /tasks/{ownerId}` | Crear atómicamente la tarea, su proyección pendiente y su entrada al GSI global. |
| [`get_tasks_querys`](02-lambda/functions/get_tasks_querys/README.md) | Dos rutas `GET` | Consultar páginas de todas las tareas o solo las pendientes de un dueño. |
| [`lambda_complete_task`](02-lambda/functions/lambda_complete_task/README.md) | `PATCH .../complete` | Completar de forma idempotente y retirar todas las vistas pendientes. |
| [`lambda_find_expired_task`](02-lambda/functions/lambda_find_expired_task/README.md) | EventBridge diario | Recorrer seis shards con checkpoints, preparar un outbox y publicar `TaskOverdue`. |
| [`lambda_task_overdue_consumer`](02-lambda/functions/lambda_task_overdue_consumer/README.md) | SNS | Reclamar cada evento una sola vez y demostrar un efecto idempotente. |

## Documentación para estudiar el código

Empieza por el [índice de estudio](documentacion/README.md). Allí encontrarás una explicación de la arquitectura completa, el modelo DynamoDB, un recorrido recomendado por las capas y una guía detallada dentro de cada Lambda.
