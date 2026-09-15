# 13. Índices de soporte

## Contexto

Una plataforma de soporte B2B registra tickets por tenant. Un administrador consulta los tickets de su tenant ordenados por prioridad; el equipo central de incidentes debe encontrar escalaciones activas de todos los tenants por región.

## Problema

Demuestra que elegir un índice no depende solo de “necesito ordenar por otro atributo”. Debes decidir qué consultas comparten partición, cuáles necesitan una partición alternativa y qué límites hacen que una opción deje de ser adecuada.

## Reglas

- La consulta del administrador debe mantener aislamiento por tenant.
- El equipo central no puede hacer `Scan` global para ver escalaciones.
- Los tickets pueden crecer mucho en un tenant grande.
- La prioridad y el estado pueden cambiar durante la vida de un ticket.

## Criterios de aceptación

- Escribe los access patterns antes del esquema.
- Diseña y justifica una consulta que requiera un orden alternativo dentro de una misma partición.
- Diseña y justifica una consulta con clave de partición alternativa.
- Compara consistencia, límites, costo de escritura y restricciones de ambos tipos de índice.
- Incluye una prueba que evidencie por qué un `Scan` sería inaceptable.

## Alcance de servicios

DynamoDB y una Lambda/API mínima para demostrar las consultas. No añadas eventos si no solucionan un requisito del caso.

## Nivel de ayuda

Reto de diseño avanzado: debes decidir qué requisito conduce a LSI, qué requisito conduce a GSI y cuándo ninguna de las dos opciones es suficiente.
