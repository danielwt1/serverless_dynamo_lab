# 04. Órdenes e-commerce

## Contexto

Una tienda vende productos con varios ítems por pedido. Un cliente debe crear una orden de forma segura, consultar su detalle completo y operaciones debe localizar órdenes pendientes.

## Problema

Diseña un agregado de orden que devuelva metadata e ítems sin joins ni `Scan`. La creación debe tolerar reintentos HTTP y originar eventos de negocio que puedan tener varios consumidores independientes.

## Reglas

- Una orden creada con la misma idempotency key no duplica compra ni ítems.
- El detalle de una orden no puede requerir una consulta por cada ítem.
- Las órdenes pendientes no pueden obtenerse mediante `Scan`.
- Los consumidores de `OrderCreated` no deben obligar a modificar el productor.

## Criterios de aceptación

- `CreateOrder` persiste metadata e ítems de forma atómica cuando corresponda.
- `GetOrder` recupera el agregado con una consulta dirigida.
- Existe una vista paginada de pendientes.
- DynamoDB Streams, Lambda o EventBridge justifican y publican un contrato `OrderCreated` versionado.
- Añadir un consumidor nuevo no cambia el código de creación de orden.

## Alcance de servicios

DynamoDB, API Gateway, Lambda, IAM mínimo, DynamoDB Streams, EventBridge con bus personalizado y al menos tres consumidores con responsabilidades distintas.

## Nivel de ayuda

Semi-guiado: el dominio y criterios son fijos; modelo, momento de publicación del evento y responsabilidades de consumidores son decisiones tuyas.
