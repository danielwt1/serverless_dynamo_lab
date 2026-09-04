# Inventario de servicios CDK: Blog Publicación

Este documento enumera exclusivamente los servicios que el stack `BlogPublicacionStack` creará al ejecutar un despliegue. Los nombres físicos los genera CloudFormation; no se fijan nombres manuales para evitar colisiones con el recorrido AWS CLI.

## 1. DynamoDB

| Recurso | Cantidad | Propósito |
| --- | ---: | --- |
| Tabla de posts | 1 | Guarda posts y relaciones de followers con `PK` y `SK`. Tiene GSI `draft_gsi` y Stream `NEW_AND_OLD_IMAGES`. |
| Tabla de progreso | 1 | Guarda `META`, checkpoints, leases y batches del fanout. |
| GSI | 1 | `draft_gsi` permite consultar borradores por autor sin `Scan`. |
| Stream | 1 | Entrega cambios de posts a `fanout-starter`. |

## 2. SQS

| Recurso | Cantidad | Propósito |
| --- | ---: | --- |
| Cola `fanout-jobs` | 1 | Transporta trabajos de recorrido de seguidores hacia el worker. |
| DLQ de fanout | 1 | Conserva mensajes que agotaron cinco recepciones. |
| Cola `notification-jobs` | 1 | Conserva lotes de recipients para el futuro consumidor final. |
| DLQ de notificaciones | 1 | Reserva los lotes que el futuro consumidor no pueda procesar. |

Las colas de trabajo usan visibilidad de 60 segundos, superior al timeout de 30 segundos del worker. Eso evita que el mismo mensaje vuelva a aparecer mientras una invocación normal sigue ejecutándose.

## 3. Lambda y fuentes de eventos

| Lambda | Entrada | Acciones principales | Salida |
| --- | --- | --- | --- |
| `lambda_create_post` | HTTP `POST /authors/{authorId}/posts` | Valida y guarda un post. | HTTP 201/4xx/5xx; el post publicado aparece en Stream. |
| `lambda_get_posts` | Dos rutas HTTP GET | Consulta publicados o drafts con cursor. | HTTP 200 con `items` y `nextCursor`. |
| `fanout-starter` | DynamoDB Stream | Crea `META` y envía el job inicial. | Mensaje en `fanout-jobs`. |
| `follower-fanout-worker` | SQS `fanout-jobs` | Busca followers, persiste checkpoint y batches. | Mensajes en `notification-jobs` o continuación. |

El stack crea dos event source mappings: Stream → starter y `fanout-jobs` → worker. El segundo declara `ReportBatchItemFailures`, para que un mensaje fallido no repita los demás mensajes correctos del mismo lote. No hay mapping para `notification-jobs`: falta implementar el writer final.

## 4. API Gateway, IAM y CloudWatch

| Servicio | Recursos | Propósito |
| --- | ---: | --- |
| API Gateway HTTP API | 1 API, 3 rutas, integraciones y stage `$default` | Expone crear post, listar publicados y listar borradores. |
| IAM | 4 roles y políticas asociadas | Un rol aislado por Lambda con permisos de logs y de los recursos que usa. |
| CloudWatch Logs | 5 grupos | Uno por Lambda y uno para access logs de API. Retención de 3 días. |

Los roles no tienen permisos de administrador ni acceso a ECR. ECR participa sólo durante el despliegue mediante los assets privados del bootstrap CDK; las Lambdas usan sus roles únicamente en ejecución.

## 5. Decisiones de laboratorio

- DynamoDB usa demanda bajo pedido y las tablas se destruyen con el stack.
- PITR y protección contra borrado están desactivados para facilitar la limpieza del laboratorio. En producción deberían revisarse porque agregan protección y costo.
- Las colas y los logs también usan `DESTROY`; sus datos se pierden con `cdk destroy`.
- La API es pública para practicar. Producción necesita autenticación, autorización, límites y revisión de exposición.
