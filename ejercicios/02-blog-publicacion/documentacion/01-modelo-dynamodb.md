# 1. Modelo DynamoDB y patrones de acceso

## Objetivo del modelo

La tabla `author_post` guarda posts y relaciones de seguimiento. El diseño empieza por las consultas, no por entidades relacionales.

| Patrón | Operación | Clave que lo permite |
|---|---|---|
| Crear post | `PutItem` | `PK=AUTHOR_ID#<author>` + `SK=STATUS#<status>#CREATED_AT#...#POST_ID#...` |
| Leer publicados | `Query` a tabla | PK del autor + prefijo `STATUS#PUBLISHED` |
| Leer drafts | `Query` a `draft_gsi` | `GSI1PK=STATUS#DRAFT#AUTHOR_ID#<author>` |
| Recorrer followers | `Query` a tabla | `PK=FOLLOWING#<author>`, prefijo `FOLLOWER#` |

No se usa `Scan`: cada lectura conoce su partición. El timestamp ISO-8601 UTC en el `SK` mantiene el orden cronológico lexicográfico.

## Items principales

```text
# Post DRAFT
PK     = AUTHOR_ID#author-1
SK     = STATUS#DRAFT#CREATED_AT#2026-08-24T15:00:00Z#POST_ID#post-1
GSI1PK = STATUS#DRAFT#AUTHOR_ID#author-1
GSI1SK = CREATED_AT#2026-08-24T15:00:00Z#POST_ID#post-1

# Post PUBLISHED
PK     = AUTHOR_ID#author-1
SK     = STATUS#PUBLISHED#CREATED_AT#2026-08-24T15:00:00Z#POST_ID#post-1

# Follow
PK = FOLLOWING#author-1
SK = FOLLOWER#user-45
```

## Decisiones importantes

### GSI disperso para drafts

Solo los drafts contienen `GSI1PK` y `GSI1SK`; los publicados no se copian al índice. La consulta privada no lee publicaciones y el índice contiene menos datos.

### Consistencia y paginación

Los publicados se consultan con lectura fuerte en la tabla base. Los drafts se leen por un GSI, que es eventualmente consistente. `LastEvaluatedKey` es un mapa, no un número: se serializa como cursor Base64 opaco y el cliente lo devuelve sin modificarlo.

### Creación e idempotencia

El create usa `PutItem` con `attribute_not_exists(PK) AND attribute_not_exists(SK)`. La condición y escritura son atómicas. Una repetición HTTP sin idempotency key aún puede crear otro post porque la Lambda genera un `postId` nuevo; si el producto lo requiere, se añade esa clave a nivel de API.

## Punto pendiente: publicar

Mover un draft a publicado cambia el `SK`, porque el estado está dentro de la clave. Un `UpdateItem` no puede cambiar una clave primaria: la publicación debe ser una transacción que cree el item publicado y elimine/condicione el draft, o se debe rediseñar la clave. Esta decisión afecta el Stream: el fanout actual espera un `MODIFY` con imágenes antigua y nueva. Antes de conectarlo hay que definir una representación de publicación que conserve esa transición observable.
