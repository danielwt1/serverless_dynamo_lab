# 14. Versionado documental

## Contexto

Una herramienta colaborativa permite editar políticas internas. Legal debe consultar cualquier revisión anterior; las personas usuarias deben ver siempre la versión actual y dos editores no pueden sobrescribir silenciosamente cambios entre sí.

## Problema

Modela documentos con historial inmutable y una representación actual eficiente. El API debe detectar conflictos de edición y permitir leer una versión específica sin recorrer el historial completo.

## Reglas

- Cada versión publicada conserva su contenido y autoría.
- Una actualización basada en una versión antigua debe informar conflicto, no reemplazar cambios recientes.
- Listar el historial y obtener una versión puntual no usa `Scan`.
- El documento actual debe ser rápido de leer sin reconstruir todas las revisiones.

## Criterios de aceptación

- Define access patterns para actual, historial, versión puntual y conflicto.
- Prueba dos actualizaciones concurrentes contra la misma revisión.
- Explica inmutabilidad, versionado y política de retención.
- Distingue evento de cambio de documento de evento de una versión ya confirmada.

## Alcance de servicios

DynamoDB, Lambda, API Gateway y EventBridge solo si hay un consumidor real de revisiones confirmadas.

## Nivel de ayuda

Reto de diseño avanzado: el esquema de versión, la condición de escritura y la forma de representar “actual” son parte de tu propuesta.
