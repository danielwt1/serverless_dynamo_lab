# 08. Inventario multi-almacén

## Contexto

Una empresa controla existencias y movimientos de SKU por almacén. Operaciones necesita el stock actual, el historial de cambios y alertas antes de quedarse sin inventario.

## Problema

Registra entradas, salidas y ajustes sin permitir stock negativo. Mantén una vista operacional de bajo stock y demuestra qué ocurre al recibir lotes o reintentos de movimientos.

## Reglas

- Un movimiento debe ser trazable e idempotente.
- Una salida no puede dejar stock negativo.
- El historial y el stock actual tienen responsabilidades diferentes.
- Un SKU de alto volumen no puede convertirse en una partición caliente sin que la propuesta lo detecte y mitigue.
- Bajo stock no puede requerir `Scan` global.

## Criterios de aceptación

- Describe el modelo de stock actual, historial y umbral mínimo.
- Define cómo medirías y qué diseño aplicarías si un SKU concentra demasiadas escrituras.
- Prueba concurrencia o reintento de movimientos.
- Procesa cambios desde Streams sin reprocesar con éxito ítems fallidos de un lote.
- Publica `StockBelowMinimum` sin generar alertas duplicadas innecesarias.

## Alcance de servicios

DynamoDB, Lambda, API Gateway, Streams, EventBridge, SQS/DLQ y alarmas de errores, edad de iterador y mensajes en DLQ.

## Nivel de ayuda

Reto de diseño: decide si la proyección de stock se actualiza síncrona o asíncronamente y defiende el trade-off.
