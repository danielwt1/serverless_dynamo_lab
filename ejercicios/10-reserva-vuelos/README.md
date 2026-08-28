# 10. Reserva de vuelos

## Contexto

Una aerolínea permite buscar vuelos, bloquear un asiento, autorizar un pago simulado y emitir un ticket. Cualquier paso puede fallar después de que otro ya haya tenido efecto.

## Problema

Construye una reserva como workflow con estado y compensaciones. Debe ser posible identificar el paso fallido, reintentar el recuperable y liberar el asiento si no se completa el pago o la emisión.

## Reglas

- Ver asientos de un vuelo y fecha no puede usar `Scan`.
- Un asiento solo puede bloquearse una vez para una solicitud válida.
- Un fallo de pago o emisión libera el asiento mediante compensación.
- La notificación de confirmación no revierte una reserva ya confirmada si falla.

## Criterios de aceptación

- Modela estados de reserva y del asiento de forma explícita.
- Implementa una State Machine con retry, `Catch` y camino compensador.
- Publica `FlightBookingConfirmed` solo después de una confirmación consistente.
- Prueba camino feliz, fallo de pago y fallo posterior al bloqueo.

## Alcance de servicios

API Gateway, Lambda, DynamoDB, Step Functions, EventBridge, SQS/DLQ para trabajo posterior y alarmas del workflow.

## Nivel de ayuda

Reto abierto: solo se entrega el resultado de negocio y los fallos que debes resolver. El diagrama de estados es parte del entregable.
