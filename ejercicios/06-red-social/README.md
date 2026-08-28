# 06. Red social

## Contexto

Una red profesional permite seguir y dejar de seguir a otras personas. La aplicación muestra ambos sentidos de la relación y contadores de seguidores.

## Problema

Modela una relación muchos-a-muchos que responda “a quién sigo”, “quién me sigue” y “¿sigo a esta persona?” sin `Scan`. Los contadores agregados deben tolerar reintentos de eventos.

## Reglas

- No se puede seguir dos veces a la misma persona.
- Dejar de seguir una relación inexistente no puede corromper contadores.
- Las relaciones en ambos sentidos y la verificación puntual son consultas dirigidas.
- Reprocesar `FollowCreated` o `FollowDeleted` no altera el total dos veces.

## Criterios de aceptación

- Documenta todos los access patterns y su costo esperado.
- Implementa creación y eliminación condicional de la relación.
- Mantiene o justifica una proyección de contadores.
- Publica eventos versionados y prueba su idempotencia.

## Alcance de servicios

DynamoDB, Lambda, API Gateway, Streams, EventBridge y una estrategia explícita para reintentos.

## Nivel de ayuda

Reto con restricciones: el patrón de datos, la proyección y la frontera entre síncrono/asíncrono son parte de la solución.
