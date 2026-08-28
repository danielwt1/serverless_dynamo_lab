# 09. Streaming y catálogo

## Contexto

Una plataforma de contenido guarda catálogo, historial de reproducción, ratings y métricas de consumo. Producto pide búsquedas complejas y rankings globales.

## Problema

Diseña lo que DynamoDB resuelve bien y reconoce de forma explícita lo que debe salir a servicios de búsqueda o analítica. El éxito del ejercicio no es forzar todo en una tabla.

## Reglas

- El historial por persona debe ser paginado y ordenado.
- El catálogo por una dimensión bien definida no usa `Scan`.
- Las actualizaciones de rating toleran concurrencia.
- “Más visto global” y texto libre deben tener una solución honesta, no una consulta artificial de DynamoDB.

## Criterios de aceptación

- Separa modelo operacional de datos analíticos.
- Explica qué eventos salen del dominio y quién los consume.
- Justifica si usas agregados, S3/Athena, OpenSearch u otra alternativa.
- Incluye costo, latencia y consistencia en la decisión.

## Alcance de servicios

DynamoDB, Lambda, EventBridge y una integración de analítica o búsqueda elegida y justificada. Incluye observabilidad de la canalización de eventos.

## Nivel de ayuda

Reto abierto: no hay una arquitectura canónica. Debes documentar alternativas descartadas y por qué.
