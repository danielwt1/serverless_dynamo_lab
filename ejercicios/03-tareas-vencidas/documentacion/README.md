# Guía de estudio del ejercicio 03: tareas vencidas

Esta carpeta explica la implementación actual del ejercicio. El objetivo es que puedas leer el sistema desde la necesidad de negocio hasta las expresiones concretas de DynamoDB y entender por qué existe cada archivo.

## Ruta de lectura recomendada

1. [Arquitectura y flujo global](00-arquitectura-y-flujo-global.md): actores, responsabilidades, recorridos completos e invariantes.
2. [Modelo DynamoDB](01-modelo-dynamodb.md): items, claves, índices, access patterns, transacciones y estados.
3. [Cómo leer el código](02-recorrido-del-codigo.md): arquitectura por puertos, orden de lectura y ejercicios de seguimiento.
4. [Crear una tarea](../02-lambda/functions/lambda_create_task/README.md).
5. [Consultar tareas](../02-lambda/functions/get_tasks_querys/README.md).
6. [Completar una tarea](../02-lambda/functions/lambda_complete_task/README.md).
7. [Buscar y publicar tareas vencidas](../02-lambda/functions/lambda_find_expired_task/README.md).
8. [Consumir `TaskOverdue`](../02-lambda/functions/lambda_task_overdue_consumer/README.md).
9. [Reintentos e idempotencia](../05-resiliencia/reintentos-e-idempotencia.md).
10. [Validación contra el XLSX](validacion-modelo.md).

## Fuentes de verdad

| Tema | Archivo |
| --- | --- |
| Comportamiento HTTP | [`openapi.yaml`](../03-api/contracts/openapi.yaml) |
| Evento `TaskOverdue` | [`task-overdue.schema.json`](../04-eventos/notificaciones/task-overdue.schema.json) |
| Recursos y permisos AWS | [`tasks-overdue-lab-stack.ts`](../04-cdk/lib/tasks-overdue-lab-stack.ts) |
| Diseño DynamoDB esperado | `Model Dynamo Task Expired.xlsx` y [validación del modelo](validacion-modelo.md) |
| Comportamiento ejecutable | Código Go y sus pruebas dentro de `02-lambda/functions` |

Si una explicación y el código difieren, las pruebas y el código actual describen lo que realmente se ejecuta. La documentación debe actualizarse junto con cualquier cambio de contrato, claves o flujo.

## Vocabulario mínimo

- **Access pattern:** pregunta concreta que debe responder DynamoDB, por ejemplo “listar las tareas pendientes de un dueño”.
- **Item principal:** registro que representa la tarea completa.
- **Proyección:** segundo item con datos duplicados y una clave ordenada para resolver otra consulta.
- **GSI:** índice secundario global. Permite consultar por claves distintas de `PK` y `SK`.
- **Índice disperso:** GSI al que solo entran items que contienen sus claves. Al quitar `GSI1PK` y `GSI1SK`, una tarea completada deja de ser visible como pendiente.
- **Shard:** partición lógica usada para repartir una consulta global. Este ejercicio usa seis: `00` a `05`.
- **Cursor:** representación opaca de `LastEvaluatedKey`; indica dónde continuar una consulta paginada.
- **Checkpoint:** progreso confirmado de un shard dentro de una corrida.
- **Outbox:** eventos persistidos antes de publicarlos. Permite retomar una publicación fallida.
- **Marker:** item estable que registra la identidad y el estado de un evento para controlar duplicados.
- **Idempotencia:** repetir una operación produce el mismo resultado de negocio que ejecutarla una sola vez.
