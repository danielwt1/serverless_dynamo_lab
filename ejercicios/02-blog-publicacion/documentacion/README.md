# Guía del laboratorio: publicación de un blog

Esta carpeta es la ruta de lectura del ejercicio. Cada documento tiene un propósito único.

## Orden recomendado

1. [00-conceptos-del-lab.md](00-conceptos-del-lab.md): teoría aplicada y los patrones que el caso enseña.
2. [01-modelo-dynamodb.md](01-modelo-dynamodb.md): consultas, claves y decisiones de modelo.
3. [02-implementacion-lambdas.md](02-implementacion-lambdas.md): código y patrones por Lambda.
4. [03-eventos-fanout-y-resiliencia.md](03-eventos-fanout-y-resiliencia.md): eventos, cursores, reintentos e idempotencia.
5. [04-infraestructura-aws.md](04-infraestructura-aws.md): recursos AWS, configuración e IAM.
6. [05-despliegue-y-verificacion.md](05-despliegue-y-verificacion.md): consola, CLI, pruebas y evidencias.

## Estado del laboratorio

Están implementados create, consultas GET, `fanout-starter` y `follower-fanout-worker`.
Falta implementar la transición `DRAFT -> PUBLISHED`. Debe resolverse antes de activar el
Stream en un entorno real, porque el estado actual forma parte de la clave de ordenamiento.

Los README de las fases técnicas solo actúan como puntos de entrada hacia estas guías.
