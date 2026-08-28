# 0. Conceptos del laboratorio: usuarios

Este documento explica las ideas que se practican en el laboratorio. Los documentos siguientes muestran las decisiones concretas, los recursos AWS y el código que las implementa.

## Modelar DynamoDB desde las preguntas del producto

En una base relacional suele empezarse por una entidad `User` y sus columnas. En DynamoDB el punto de partida son las lecturas y escrituras que debe soportar el producto. Aquí son: autenticar por correo, obtener un perfil por identificador y listar usuarios activos recientes.

Cada pregunta debe poder responderse con una clave de partición conocida. Por eso el laboratorio no hace `Scan`: la autenticación conoce `EMAIL#<correo>`, el perfil conoce `USER#<id>` y el listado usa una partición fija en un índice. El modelo específico está en [01-modelo-dynamodb.md](01-modelo-dynamodb.md).

## Unicidad bajo concurrencia

Comprobar primero si existe un correo y escribir después no garantiza unicidad: dos solicitudes pueden observar que el correo no existe antes de que cualquiera escriba. La garantía debe estar en DynamoDB.

El laboratorio reserva `EMAIL#<correo-normalizado> / UNIQUE` con una condición de no existencia y crea el perfil en la misma transacción. Solo una solicitud puede reservar esa clave; si falla, no se escribe el perfil. Es el patrón de **reserva de unicidad**.

## Índice secundario disperso

Un GSI es una vista adicional de la tabla. Es disperso cuando solo aparecen en él los ítems que incluyen sus atributos de índice. Aquí solo un usuario activo lleva `GSI1PK=ACTIVE_USERS`; al desactivarlo se retiran esos atributos y deja de aparecer en el listado sin borrar su historial.

Los GSIs tienen consistencia eventual. Por eso sirven para la vista operativa, mientras el login usa una lectura fuerte de la reserva de email en la tabla base.

## Arquitectura hexagonal ligera

La dependencia va hacia el centro:

```text
API Gateway -> handler -> caso de uso -> puerto de salida -> adaptador DynamoDB
```

- El **handler** entiende HTTP y JSON: deserializa, valida la forma de la petición y traduce resultados a códigos HTTP.
- El **caso de uso** expresa la operación de negocio sin depender del SDK de AWS.
- Los **puertos** son interfaces pequeñas: definen lo que el caso de uso necesita de afuera.
- El **adaptador** implementa el puerto usando DynamoDB y traduce errores de infraestructura a errores de dominio.

El resultado es que el caso de uso se puede probar con un fake y que cambiar el transporte o la persistencia no obliga a reescribir la regla de negocio. La estructura real de las Lambdas se describe en [02-implementacion-lambdas.md](02-implementacion-lambdas.md).

## Lambda, configuración y contexto

`main.go` es el punto de composición: carga `aws.Config`, lee variables de entorno, crea el cliente DynamoDB, conecta adaptador, caso de uso y handler, y finalmente registra el handler en Lambda. Ese trabajo se ejecuta cuando AWS crea un entorno de ejecución; puede reutilizarse en invocaciones cálidas.

Cada invocación recibe su propio `context.Context`. Ese contexto debe viajar desde el handler hasta DynamoDB para respetar deadline y cancelación. El cliente AWS se construye fuera del handler para reutilizar conexiones entre invocaciones del mismo entorno.

## Errores entre capas

Una condición fallida de la transacción es un detalle de DynamoDB. El adaptador la reconoce y devuelve `domain.ErrUserAlreadyExists`; el handler conoce ese error de dominio y lo transforma en `409 Conflict`. Así HTTP no depende de tipos del SDK y el dominio no depende de HTTP.

## Pruebas

Las pruebas unitarias cubren el caso de uso con fakes y el handler con eventos HTTP simulados. Las pruebas del adaptador validan la traducción de DynamoDB y conviene complementarlas con integración contra DynamoDB Local o una tabla aislada. Los escenarios importantes son creación exitosa, correo repetido, JSON inválido, paginación y credenciales inválidas.
