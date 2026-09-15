# 07. SaaS multi-tenant

## Contexto

Una plataforma B2B maneja tenants, proyectos, tareas, comentarios y personas asignadas. Cada tenant exige aislamiento lógico y trazabilidad de las acciones relevantes.

## Problema

Diseña un sistema que permita trabajar dentro de un tenant, consultar tareas asignadas y producir auditoría sin mezclar datos de organizaciones distintas ni bloquear la operación por cargas de auditoría.

## Reglas

- Toda consulta de negocio debe conservar el contexto de `tenantId`.
- Un usuario solo ve recursos del tenant del que forma parte.
- Dos ediciones concurrentes de una tarea no pueden perder silenciosamente el cambio de otra persona.
- La auditoría no debe ralentizar la escritura principal.
- Un pico de eventos de auditoría debe tolerar reintentos y procesarse de forma idempotente.

## Criterios de aceptación

- Define consultas por proyecto, tarea y persona asignada dentro de tenant.
- Demuestra el conflicto y la resolución de una actualización concurrente.
- Justifica cualquier GSI y cómo evita fuga de datos entre tenants.
- Publica eventos que incluyan identidad, tenant, correlación y versión.
- Separa auditoría, métricas y notificaciones mediante consumidores desacoplados.

## Alcance de servicios

DynamoDB single-table o alternativa justificada, Lambda, API Gateway, EventBridge, SQS con DLQ para proyecciones pesadas y CloudWatch.

## Nivel de ayuda

Reto de diseño: recibe dominio y restricciones, no entidades ni esquema propuesto.
