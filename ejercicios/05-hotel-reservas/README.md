# 05. Hotel y reservas

## Contexto

Una cadena hotelera recibe solicitudes concurrentes para reservar habitaciones. Huéspedes y personal operativo necesitan vistas distintas de las reservas y de la disponibilidad.

## Problema

Evita doble reserva para el mismo recurso y periodo sin convertir la tabla en un calendario imposible de consultar. Cuando una reserva cambia, las notificaciones, auditoría y disponibilidad deben reaccionar sin acoplarse entre sí.

## Reglas

- Dos solicitudes concurrentes no pueden confirmar la misma habitación y fecha.
- Cancelar debe liberar la disponibilidad de forma coherente.
- Consultar reservas de una habitación o de un huésped no usa `Scan`.
- La disponibilidad por día es un dato derivado que debes mantener y explicar.

## Criterios de aceptación

- Demuestra una condición de escritura que protege la reserva.
- Explica las consultas por habitación, huésped y fecha.
- Define eventos `BookingCreated` y `BookingCancelled` con idempotencia.
- Separa fallos de notificación de la confirmación de una reserva.

## Alcance de servicios

DynamoDB, Lambda, API Gateway, Streams, EventBridge, TTL cuando aplique y observabilidad de conflictos de concurrencia.

## Nivel de ayuda

Reto con restricciones: no se entrega modelo de claves ni arquitectura. Entrega propuesta de diseño antes de consultar una referencia técnica futura.
