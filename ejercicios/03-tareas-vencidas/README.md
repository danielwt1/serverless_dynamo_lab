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
- Lista sus tareas pendientes sin filtrar una colección completa en memoria.
- Encuentra vencidas globales mediante una consulta dirigida.
- La ejecución programada publica `TaskOverdue` solo para el cambio que corresponda.

## Alcance de servicios

DynamoDB, API Gateway, Lambda, CloudWatch Logs y EventBridge Scheduler. Define tú qué Lambda publica el evento y qué consumidor inicial demuestra que el flujo funciona.

## Nivel de ayuda

Semi-guiado: documenta primero access patterns, claves, GSI y contrato de evento. No añadas índices ni servicios hasta justificar qué consulta o fallo resuelven.
