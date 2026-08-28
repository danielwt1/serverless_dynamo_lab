# Serverless Dynamo Lab

Laboratorios prácticos de Go, DynamoDB, Lambda y arquitectura serverless. Cada caso documenta el reto de negocio, la teoría aplicada, las decisiones arquitectónicas, su reflejo en código y un camino de despliegue por consola o CLI.

## Ruta de laboratorios

Los laboratorios 01 y 02 están implementados y guiados. Los laboratorios 03 a 12 son enunciados progresivos: primero resuelve su propuesta y después añade las decisiones, código e infraestructura. La cantidad de pistas disminuye de forma intencional.

| Etapa | Laboratorio | Foco principal | Nivel de guía |
| --- | --- | --- | --- |
| 01 | [Usuarios](ejercicios/01-usuarios/README.md) | Unicidad, GSI disperso y API HTTP. | Guiado e implementado. |
| 02 | [Publicación de blog](ejercicios/02-blog-publicacion/README.md) | Estados, Streams, SQS, fanout e idempotencia. | Guiado e implementado. |
| 03 | [Tareas vencidas](ejercicios/03-tareas-vencidas/README.md) | GSI disperso y EventBridge Scheduler. | Semi-guiado. |
| 04 | [Órdenes e-commerce](ejercicios/04-ordenes-ecommerce/README.md) | Agregado, transacción, Streams y EventBridge. | Semi-guiado. |
| 05 | [Hotel y reservas](ejercicios/05-hotel-reservas/README.md) | Concurrencia, disponibilidad derivada y eventos. | Reto con restricciones. |
| 06 | [Red social](ejercicios/06-red-social/README.md) | Relaciones bidireccionales, contadores e idempotencia. | Reto con restricciones. |
| 07 | [SaaS multi-tenant](ejercicios/07-saas-multitenant/README.md) | Aislamiento por tenant y auditoría por eventos. | Reto de diseño. |
| 08 | [Inventario](ejercicios/08-inventario/README.md) | Movimientos, stock, Streams y alertas. | Reto de diseño. |
| 09 | [Streaming y catálogo](ejercicios/09-streaming-catalogo/README.md) | Límites de DynamoDB y analítica complementaria. | Reto abierto. |
| 10 | [Reserva de vuelos](ejercicios/10-reserva-vuelos/README.md) | Saga, compensaciones y Step Functions. | Reto abierto. |
| 11 | [Facturación SaaS](ejercicios/11-facturacion-saas/README.md) | Eventos de uso, SQS, cierre mensual y analítica. | Reto abierto. |
| 12 | [Marketplace](ejercicios/12-marketplace/README.md) | Checkout, reservas, fulfillment y plataforma completa. | Proyecto final. |
| 13 | [Índices de soporte](ejercicios/13-indices-soporte/README.md) | Decidir y comprobar LSI frente a GSI. | Reto de diseño avanzado. |
| 14 | [Versionado documental](ejercicios/14-versionado-documentos/README.md) | Historial inmutable y control optimista de concurrencia. | Reto de diseño avanzado. |
| 15 | [Catálogo de lectura intensiva](ejercicios/15-catalogo-lectura-intensiva/README.md) | Caché, DAX y consistencia. | Decisión de arquitectura. |
| 16 | [Perfiles multi-región](ejercicios/16-perfiles-multiregion/README.md) | Global Tables, conflictos y recuperación regional. | Decisión de arquitectura. |
| 17 | [Recuperación y auditoría](ejercicios/17-recuperacion-auditoria/README.md) | PITR, backup, exportación y restauración verificable. | Operación de producción. |
| 18 | [Pagos y entrega de eventos](ejercicios/18-pagos-eventos/README.md) | Outbox, Streams, Pipes y entrega al menos una vez. | Reto de consistencia. |

## Cómo leer un laboratorio

1. Lee el `README.md`: contexto, problema, reglas y objetivo.
2. Continúa en `documentacion/00-conceptos-del-lab.md`: teoría aplicada al reto.
3. Revisa las decisiones concretas: modelo DynamoDB, arquitectura/código, contrato, infraestructura y despliegue.
4. Usa el código, OpenAPI, scripts y colección Postman para comprobar la implementación.

```text
ejercicios/<laboratorio>/
├── README.md                 reto y diseño general
├── documentacion/            teoría, decisiones y despliegue
├── 01-dynamodb/              activos del modelo, si aplica
├── 02-lambda/                código de las funciones
├── 03-api/                   contrato HTTP, si aplica
└── postman/                  pruebas manuales, si aplica
```

## Principio de diseño

Una consulta debe poder explicarse como una necesidad de negocio. A partir de ella se define el patrón de acceso, el modelo DynamoDB, la operación, la arquitectura y finalmente los recursos AWS; no al revés.
