# Infraestructura del ejercicio de tareas vencidas

Este stack crea las dos tablas del modelo, cuatro operaciones HTTP, el batch diario con concurrencia reservada, SNS, un consumidor idempotente y una DLQ para el Scheduler.

## Recursos creados

| Recurso | Configuración relevante |
| --- | --- |
| Tabla de tareas | `PK` + `SK`, on demand, GSI `task_gsi_pending` |
| Tabla de procesos | `PK` + `SK`, on demand, TTL `ttl`, GSI `process_gsi_open` |
| Cinco Lambdas | Imágenes Docker Linux ARM64 y logs JSON con retención de una semana |
| HTTP API | Stage `$default`, auto deploy, payload v2 y access logs |
| Topic SNS | Publicación de `TaskOverdue` y suscripción del consumidor |
| Regla EventBridge | Cron diario a las `05:00 UTC`, dos reintentos y edad máxima de dos horas |
| SQS DLQ | Retención de 14 días para invocaciones programadas agotadas |

`TaskLambda` es el construct compartido que empaqueta cada carpeta como imagen Docker. Las Lambdas HTTP usan por defecto 256 MiB y 10 segundos. El batch usa 512 MiB, 900 segundos y concurrencia reservada en 1.

Los permisos se asignan por función. Crear solo puede escribir transacciones; consultar solo puede hacer `Query`; completar puede leer y escribir; el batch consulta el GSI, administra la tabla de procesos y publica en SNS; el consumidor solo lee y actualiza markers.

## Archivos

| Archivo | Responsabilidad |
| --- | --- |
| [`lib/tasks-overdue-lab-stack.ts`](lib/tasks-overdue-lab-stack.ts) | Conecta tablas, Lambdas, SNS, schedule, DLQ y API |
| [`lib/constructs/tasks-tables.ts`](lib/constructs/tasks-tables.ts) | Tablas e índices |
| [`lib/constructs/task-lambda.ts`](lib/constructs/task-lambda.ts) | Configuración común de Lambdas Docker |
| [`lib/constructs/tasks-http-api.ts`](lib/constructs/tasks-http-api.ts) | Rutas, integraciones, stage y logs de acceso |
| [`lib/config/tasks-config.ts`](lib/config/tasks-config.ts) | Nombres de índices, rutas y valores del laboratorio |
| [`test/tasks-overdue-lab-stack.test.ts`](test/tasks-overdue-lab-stack.test.ts) | Aserciones sobre la plantilla sintetizada |

## Validación y despliegue

```code
npm install
npm test
npm run synth
npm run deploy
```

`npm test` verifica la forma de la infraestructura y `npm run synth` produce CloudFormation localmente. `npm run deploy` sí modifica la cuenta AWS configurada.

Los outputs son:

- `ApiEndpoint`: URL base de las rutas definidas en [`openapi.yaml`](../03-api/contracts/openapi.yaml).
- `TasksTableName`: tabla de tareas y proyecciones.
- `ProcessesTableName`: tabla de corridas, checkpoints y eventos.
- `TaskOverdueTopicArn`: ARN del topic.
- `ScheduleDLQUrl`: cola para fallos definitivos del trigger diario.

Consulta la [guía global](../documentacion/00-arquitectura-y-flujo-global.md) para seguir el flujo entre estos recursos.
