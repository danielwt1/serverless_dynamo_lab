# 02. Publicacion de un blog

## Contexto
Una plataforma permite que autores escriban borradores y publiquen articulos. Los lectores consultan el contenido publico ordenado por fecha.

## Problema
Un borrador nunca debe aparecer para lectores. Cuando un articulo se publica, varios procesos deben reaccionar sin hacer que el editor espere a cada notificacion.

## Reglas
- Solo el autor o un editor puede publicar.
- Publicar dos veces no debe producir dos publicaciones de negocio.
- El contenido publicado se ordena por fecha.
- Un lector no puede ver borradores.

## Preguntas que debes resolver
- Como consultarias todos los articulos de un autor?
- Como separarias borradores de contenido publico?
- Que cambio exacto debe producir el evento?
- Como harias idempotente la notificacion a suscriptores?

## Entregables
Patrones de acceso, modelo de posts, operaciones de borrador y publicacion, contrato de `PostPublished`, y pruebas de transicion de estado.

## Documentación del laboratorio

Sigue la [guía central](documentacion/README.md). Allí están el modelo
DynamoDB, la explicación de cada Lambda, el fanout, la infraestructura y el
despliegue por consola o CLI en un orden único de lectura.

Las carpetas de fase conservan el código y sus puntos de entrada; no duplican
las decisiones de diseño.

La alternativa IaC del ejercicio está en [04-cdk](04-cdk/README.md). El
[inventario de servicios](04-cdk/documentacion/01-inventario-servicios.md)
enumera qué creará su stack antes de desplegarlo.
