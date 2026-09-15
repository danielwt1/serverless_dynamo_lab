# 2. Implementación de Lambdas

## Patrón común: arquitectura hexagonal ligera

```text
evento HTTP / Stream / SQS -> handler -> caso de uso -> puerto -> adaptador AWS SDK
```

El dominio no depende de AWS. El handler traduce entrada/salida; el caso de uso aplica reglas; el puerto expresa la dependencia y el adaptador usa DynamoDB o SQS. Así las reglas se prueban con fakes sin AWS.

## `lambda_create_post`

**Entrada:** `POST /authors/{authorId}/posts`, con `description` y `status`.

Valida campos y estados, genera UUID y timestamp UTC, modela las claves y hace `PutItem` condicional. Para `DRAFT` añade claves del GSI; para `PUBLISHED` no. Su código se organiza en `domain`, `application`, `ports`, `infrastructure` y `handler`.

## `lambda_get_posts`

- `GET /authors/{authorId}/posts`: publicados desde la tabla base.
- `GET /authors/{authorId}/drafts`: drafts desde `draft_gsi`.

Cada ruta hace `Query`, nunca `Scan`, y devuelve un cursor opaco. Publicados usan lectura fuerte; el GSI de drafts usa la consistencia eventual propia de DynamoDB.

## `fanout-starter`

**Trigger:** DynamoDB Stream con `NEW_AND_OLD_IMAGES`.

Filtra únicamente `MODIFY` de `DRAFT` a `PUBLISHED`, crea el `META` de progreso con el `eventID` del Stream y envía el primer job. Si el Stream reintenta después de SQS puede reenviar el job: se acepta el duplicado antes que perder el fanout.

## `follower-fanout-worker`

**Trigger:** `fanout-jobs` SQS con partial batch response.

Consulta followers de a 100, produce recipients en grupos de 50 y procesa como máximo 1.000 por invocación. Guarda batches y checkpoint con una transacción; si queda trabajo envía una continuación. La guía 3 explica sus contratos y fallos.

## Construcción de cada Lambda Go

Desde la carpeta de cada función, sin mezclar módulos:

```bash
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bootstrap .
zip function.zip bootstrap
```

En AWS selecciona `provided.al2023` y `arm64`. Para `x86_64`, usa `GOARCH=amd64`.
