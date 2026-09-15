# Laboratorios

Cada carpeta es un laboratorio. Los dos primeros están implementados y documentados de punta a punta; desde el tercero la carpeta inicia con el problema, reglas y criterios de aceptación. La solución se construye progresivamente y no debe adelantarse en el enunciado.

| Laboratorio | Qué se aprende |
|---|---|
| [01 · Usuarios](01-usuarios/README.md) | Access patterns, unicidad atómica, GSI disperso, arquitectura hexagonal y API HTTP. |
| [02 · Publicación de blog](02-blog-publicacion/README.md) | Estados, paginación, arquitectura por caso de uso, Streams, SQS, fanout e idempotencia. |
| [03 · Tareas vencidas](03-tareas-vencidas/README.md) | GSI disperso, CRUD y tareas programadas con EventBridge Scheduler. |
| [04 · Órdenes e-commerce](04-ordenes-ecommerce/README.md) | Agregado de orden, transacción, Streams y eventos desacoplados. |
| [05 · Hotel y reservas](05-hotel-reservas/README.md) | Concurrencia, reservas por fecha y disponibilidad derivada. |
| [06 · Red social](06-red-social/README.md) | Relaciones muchos-a-muchos, proyecciones y contadores. |
| [07 · SaaS multi-tenant](07-saas-multitenant/README.md) | Aislamiento, asignaciones y auditoría basada en eventos. |
| [08 · Inventario](08-inventario/README.md) | Movimientos, consistencia de stock y alertas de bajo inventario. |
| [09 · Streaming y catálogo](09-streaming-catalogo/README.md) | Fronteras de DynamoDB, búsqueda y analítica. |
| [10 · Reserva de vuelos](10-reserva-vuelos/README.md) | Sagas y compensaciones con Step Functions. |
| [11 · Facturación SaaS](11-facturacion-saas/README.md) | Procesamiento de uso, cierre periódico e idempotencia. |
| [12 · Marketplace](12-marketplace/README.md) | Capstone de checkout y fulfillment multi-vendedor. |
| [13 · Índices de soporte](13-indices-soporte/README.md) | Decidir entre LSI y GSI a partir de consultas y límites. |
| [14 · Versionado documental](14-versionado-documentos/README.md) | Versiones inmutables y control optimista de concurrencia. |
| [15 · Catálogo de lectura intensiva](15-catalogo-lectura-intensiva/README.md) | Caché/DAX, frescura y lecturas muy frecuentes. |
| [16 · Perfiles multi-región](16-perfiles-multiregion/README.md) | Global Tables, conflictos y degradación regional. |
| [17 · Recuperación y auditoría](17-recuperacion-auditoria/README.md) | PITR, restauración y exportación analítica. |
| [18 · Pagos y entrega de eventos](18-pagos-eventos/README.md) | Outbox, CDC y publicación confiable de eventos. |

Dentro de cada laboratorio, empieza por el README y sigue el índice de `documentacion/`.
