# 0. Conceptos del laboratorio: publicación de un blog

Este documento presenta las ideas del laboratorio antes de entrar en las decisiones concretas y el código. Los documentos siguientes muestran las claves, Lambdas, recursos y despliegue que las implementan.

## Modelar desde patrones de acceso

El dominio tiene posts y seguidores, pero DynamoDB no se modela copiando entidades relacionales. Se modela desde las operaciones: crear un post, leer publicaciones de un autor, consultar sus borradores y recorrer seguidores cuando una publicación se hace pública.

Cada operación conoce su partición: `AUTHOR_ID#<author>` para posts y `FOLLOWING#<author>` para seguidores. Así las lecturas usan `Query`, no `Scan`. Las decisiones de claves e índices están en [01-modelo-dynamodb.md](01-modelo-dynamodb.md).

## Estado como parte del modelo

`DRAFT` y `PUBLISHED` no son solo etiquetas: determinan quién puede ver el post y qué consulta lo obtiene. El estado se incluyó en la sort key para leer publicados por prefijo y se creó un GSI disperso para borradores. Solo los borradores llevan las claves de ese índice, por lo que los publicados no ocupan ni aparecen en esa vista.

La consecuencia es importante: una clave primaria no puede actualizarse. Publicar un borrador cambia su sort key y necesita una transacción de crear/eliminar con condiciones, o un rediseño. La transición está documentada como pendiente para no presentar como implementado algo que aún no lo está.

## Paginación como parte del contrato

DynamoDB devuelve `LastEvaluatedKey`, una estructura interna. El API no debe exponerla tal cual: la Lambda la serializa como cursor Base64 opaco y el cliente solo la devuelve para pedir la siguiente página. Esto permite cambiar el almacenamiento sin romper consumidores y evita que manipulen claves internas.

## Arquitectura hexagonal por caso de uso

Cada Lambda sigue el recorrido:

```text
HTTP / DynamoDB Stream / SQS -> handler -> caso de uso -> puerto -> adaptador AWS
```

El handler se ocupa del evento de entrada; el caso de uso aplica la regla; el puerto expresa una dependencia mínima y el adaptador conversa con DynamoDB o SQS. No se crea una Lambda por tabla, sino por responsabilidad: escritura, lectura, inicio de fanout y procesamiento paginado tienen ritmos, permisos y fallos distintos.

La composición, responsabilidades y construcción de cada función se detallan en [02-implementacion-lambdas.md](02-implementacion-lambdas.md).

## Asincronía, fanout e idempotencia

Notificar a miles de seguidores dentro de la petición de publicar haría lenta y frágil la experiencia del editor. Un cambio de publicación inicia un proceso asíncrono: DynamoDB Stream detecta la transición, una Lambda arranca el fanout y SQS separa trabajos pequeños que un worker procesa por páginas.

SQS y Streams son *at least once*: un mensaje puede llegar más de una vez. La respuesta no es confiar en que no haya duplicados, sino hacer idempotente el destino. El progreso usa un checkpoint y batches con identificadores deterministas; el writer final reserva una notificación por usuario y post mediante una escritura condicional.

La arquitectura, los límites de batch, DLQ y escenarios de fallo están en [03-eventos-fanout-y-resiliencia.md](03-eventos-fanout-y-resiliencia.md).

## Lambda y recursos AWS

`main.go` construye configuración, clientes y dependencias una vez por entorno de ejecución. El handler recibe un `context.Context` por invocación y debe propagarlo a llamadas AWS. Las variables de entorno llevan nombres de tablas y URLs de colas; IAM permite solo las operaciones que necesita cada Lambda.

La infraestructura concreta y la guía para consola/CLI se encuentran en [04-infraestructura-aws.md](04-infraestructura-aws.md) y [05-despliegue-y-verificacion.md](05-despliegue-y-verificacion.md).

## Estrategia de pruebas

Se prueban casos de uso con fakes, handlers con eventos simulados y adaptadores contra DynamoDB Local o recursos aislados. También se prueban los fallos que definen la arquitectura: transición inválida, cursor inválido, evento repetido, fallo tras enviar una página, reanudación desde checkpoint y mensajes agotados en DLQ.
