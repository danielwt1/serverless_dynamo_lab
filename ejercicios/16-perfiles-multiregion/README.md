# 16. Perfiles multi-región

## Contexto

Un producto global permite a sus clientes actualizar preferencias de perfil desde dos regiones. El negocio necesita menor latencia regional y continuar atendiendo si una región queda indisponible.

## Problema

Diseña qué datos merecen replicación multi-región, qué escrituras se aceptan en cada región y cómo se detecta o resuelve un conflicto cuando dos actualizaciones compiten.

## Reglas

- No todos los datos tienen por qué replicarse globalmente.
- Una preferencia no puede terminar en un estado imposible después de conflictos.
- El equipo debe poder explicar qué ocurre durante pérdida temporal de una región.
- La solución debe considerar costo de réplicas, escritura y operación.

## Criterios de aceptación

- Separa datos globales, regionales y derivados.
- Define una política explícita de conflicto y qué experiencia verá el cliente.
- Prueba o simula actualizaciones concurrentes desde dos regiones.
- Documenta objetivo de recuperación, procedimiento de degradación y observabilidad de réplica.

## Alcance de servicios

DynamoDB Global Tables, Lambda/API regional, Route 53 o estrategia de enrutamiento justificada y alarmas de replicación. Puedes modelar la segunda región sin desplegarla si el costo no es aceptable, pero el diseño debe ser ejecutable.

## Nivel de ayuda

Decisión de arquitectura: Global Tables no es una respuesta automática; debes justificarla frente a una sola región con recuperación.
