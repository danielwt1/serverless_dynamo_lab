# 12. Marketplace multi-vendedor

## Contexto

Un marketplace permite catálogo por vendedor, búsqueda por categoría/precio, carrito, checkout con varios vendedores, reserva temporal de inventario, pago simulado y fulfillment independiente.

## Problema

Integra lo aprendido en una plataforma serverless completa. Una compra puede dividirse por vendedor; debe resistir reintentos, faltantes de stock, fallos parciales y consumidores lentos sin perder trazabilidad.

## Reglas

- La creación de checkout debe usar una idempotency key.
- No se reserva inventario negativo; las reservas temporales expiran de forma segura.
- El fallo de pago compensa reservas; el fallo de notificación no compensa una orden confirmada.
- Cada vendedor recibe solo su vista y trabajos de fulfillment.
- Búsqueda de texto libre no se fuerza sobre DynamoDB.

## Criterios de aceptación

- Entrega access patterns, modelo de tabla e índices antes de código.
- Implementa una saga de checkout con compensación.
- Publica eventos versionados y distribuye fulfillment mediante colas independientes.
- Incluye DLQ, idempotencia, respuestas parciales de lote, alarmas, trazabilidad y plan de reproceso.
- Demuestra camino feliz, reintento, falta de stock, fallo recuperable y fallo permanente.

## Alcance de servicios

API Gateway, Lambda, DynamoDB, TTL, Streams cuando se justifique, EventBridge, SQS/DLQ, SNS cuando aporte valor, Step Functions, observabilidad y una integración de búsqueda o analítica elegida.

## Nivel de ayuda

Proyecto final: se evaluará la calidad de las decisiones y evidencias, no que uses todos los servicios. Cada servicio debe resolver una responsabilidad concreta.
