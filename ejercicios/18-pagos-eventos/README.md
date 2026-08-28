# 18. Pagos y entrega de eventos

## Contexto

Un servicio confirma pagos y debe avisar a facturación, fulfillment y notificaciones. Si el pago queda persistido pero falla la publicación del evento, los sistemas posteriores no pueden quedar sin enterarse.

## Problema

Diseña entrega confiable al menos una vez entre el estado de pago y sus consumidores. Compara una publicación directa, una estrategia outbox y el uso de DynamoDB Streams/Pipes; maneja duplicados sin convertir la entrega en “exactamente una vez” ficticia.

## Reglas

- Confirmar pago y registrar intención de evento no pueden divergir silenciosamente.
- Un consumidor puede recibir el mismo evento más de una vez.
- Un consumidor lento o fallido no puede bloquear a los demás.
- El sistema debe poder reprocesar eventos y explicar el orden que sí o no garantiza.

## Criterios de aceptación

- Define estados de pago, identificador de evento y clave de idempotencia por consumidor.
- Demuestra el fallo entre persistir y publicar con una prueba controlada.
- Justifica el puente elegido entre tabla y EventBridge/SQS.
- Implementa DLQ, respuesta parcial de lote cuando aplique y procedimiento de reproceso.
- Documenta contrato, versión, correlación y trazabilidad de cada evento.

## Alcance de servicios

DynamoDB, Lambda, Streams o EventBridge Pipes, EventBridge, SQS/DLQ y CloudWatch. SNS solo si una responsabilidad concreta requiere su fan-out de suscriptores.

## Nivel de ayuda

Reto de consistencia: se evalúa el manejo de fallos y duplicados, no la cantidad de servicios usados.
