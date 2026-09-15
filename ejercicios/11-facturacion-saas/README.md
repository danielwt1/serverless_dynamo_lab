# 11. Facturación SaaS por uso

## Contexto

Una plataforma SaaS recibe eventos de consumo de muchos tenants y debe facturar mensualmente sin perder ni cobrar dos veces el mismo uso.

## Problema

Absorbe picos de eventos, agrega consumo operacional, conserva evidencia auditable y ejecuta un cierre mensual que calcula, cobra de forma simulada y actualiza la suscripción.

## Reglas

- Cada `usageEventId` solo puede afectar una vez la facturación.
- Un pico no debe saturar directamente la tabla ni perder eventos.
- El cierre mensual debe ser reintentable y observable por tenant y periodo.
- La evidencia histórica para análisis no tiene por qué vivir toda en DynamoDB.

## Criterios de aceptación

- Justifica datos crudos, agregados y retención.
- Usa una cola para amortiguar consumo y maneja fallos parciales por lote.
- Programa el cierre mensual y coordina sus pasos con estado persistente.
- Define qué se exporta a S3/Athena y cómo se reconcilia con lo operacional.

## Alcance de servicios

API o EventBridge de entrada, SQS/DLQ, Lambda, DynamoDB, EventBridge Scheduler, Step Functions, S3/Athena y alarmas.

## Nivel de ayuda

Reto abierto: debes presentar primero diagrama de flujo, modelo de idempotencia y plan de reproceso antes de implementar.
