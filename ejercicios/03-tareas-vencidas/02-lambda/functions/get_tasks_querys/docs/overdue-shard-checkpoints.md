# Documento reemplazado

Esta guía pertenecía a un diseño anterior con leases y publicación dentro del worker. La implementación vigente usa concurrencia reservada, un proceso abierto, checkpoints, marker y outbox en dos fases.

Consulta estas fuentes actuales:

- [Guía detallada de `lambda_find_expired_task`](../../lambda_find_expired_task/README.md).
- [Diseño de proceso y recuperación](../../lambda_find_expired_task/docs/overdue-processing-design.md).
- [Modelo DynamoDB global](../../../../documentacion/01-modelo-dynamodb.md).
- [Reintentos e idempotencia](../../../../05-resiliencia/reintentos-e-idempotencia.md).
