# Permisos IAM por Lambda

- `lambda_create_task`: `dynamodb:TransactWriteItems` sobre `task_data`.
- `lambda_complete_task`: `dynamodb:GetItem` y `dynamodb:TransactWriteItems` sobre `task_data`.
- `get_tasks_querys`: `dynamodb:Query` sobre `task_data`.
- `lambda_find_expired_task`: `dynamodb:Query` sobre el GSI de tareas; `GetItem`, `Query`, `UpdateItem` y `TransactWriteItems` sobre la tabla de procesos; `sns:Publish` sobre el tópico.
- `lambda_task_overdue_consumer`: `dynamodb:GetItem` y `dynamodb:UpdateItem` sobre la tabla de procesos.

El stack CDK asigna estos permisos por función y restringe cada recurso por ARN.
