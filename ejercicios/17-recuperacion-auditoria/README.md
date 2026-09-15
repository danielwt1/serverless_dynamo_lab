# 17. Recuperación y auditoría de datos

## Contexto

Un error de aplicación elimina o modifica masivamente datos de órdenes. Finanzas también necesita conservar evidencia histórica para consultas ad hoc sin cargar la tabla operacional.

## Problema

Diseña y ejecuta un plan de recuperación verificable. Debes distinguir respaldo, restauración, exportación analítica y auditoría de cambios; ningún concepto sustituye automáticamente al otro.

## Reglas

- Restaurar no puede destruir a ciegas la tabla productiva actual.
- El tiempo y punto de recuperación deben estar definidos antes del incidente.
- Las consultas analíticas no pueden competir sin límite con las transacciones operacionales.
- La evidencia debe permitir comparar estado restaurado y estado esperado.

## Criterios de aceptación

- Define RPO, RTO y procedimiento de incidente para una tabla concreta.
- Configura o documenta PITR, backup y restauración hacia un destino seguro.
- Exporta una porción de datos a S3 y plantea una consulta analítica con Athena.
- Ejecuta un simulacro con datos no productivos y registra evidencias/resultados.

## Alcance de servicios

DynamoDB PITR/backups, S3, Athena, IAM mínimo, CloudWatch y documentación operacional. No requiere API pública.

## Nivel de ayuda

Operación de producción: el entregable central es el runbook probado, no una Lambda adicional.
