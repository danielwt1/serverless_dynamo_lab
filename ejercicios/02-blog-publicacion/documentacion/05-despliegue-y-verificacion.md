# 5. Despliegue y verificación

La consola sirve para aprender y capturar evidencias; CLI sirve para repetir. Usa un camino completo y anota los nombres/ARN reales. Cuando el laboratorio se estabilice, el siguiente paso es dejar el camino CLI en una sola herramienta IaC (SAM, CDK, Terraform o CloudFormation).

## Orden obligatorio

1. Crear `author_post` con `PK`/`SK` (String), el GSI llamado exactamente `draft_gsi` (`GSI1PK`/`GSI1SK`, String, proyección `ALL`) y Stream `NEW_AND_OLD_IMAGES`.
2. Crear `FanoutProgress`.
3. Crear dos DLQ y después sus colas principales con redrive policy.
4. Compilar y crear Lambdas, roles y variables.
5. Crear HTTP API y conectar create/get.
6. Conectar Stream a `fanout-starter`.
7. Conectar `fanout-jobs` al worker con partial batch response.
8. Implementar y validar `DRAFT -> PUBLISHED` antes de activar el Stream real.

## Camino A: consola AWS

| Paso | Servicio | Captura sugerida |
|---|---|---|
| 1 | DynamoDB | claves, GSI y Stream |
| 2 | DynamoDB | tabla FanoutProgress |
| 3 | SQS | colas, DLQ y redrive policy |
| 4 | Lambda | runtime, arquitectura, IAM y variables |
| 5 | API Gateway | rutas e integración |
| 6-7 | Lambda | event source mapping y partial batch response |

Guarda imágenes en `documentacion/imagenes/`, por ejemplo `01-tabla-author-post.png`, `02-colas-dlq.png` y `03-trigger-stream.png`.

## Camino B: AWS CLI

Primero configura credenciales y confirma la cuenta:

```bash
aws sts get-caller-identity
```

### Sesión de AWS CLI

Los scripts y comandos de AWS no reciben access keys, secretos ni tokens. AWS CLI usa las credenciales temporales de la sesión que abriste.

- Con IAM Identity Center: ejecuta una vez `aws configure sso --profile lab-dev` y, cada vez que la sesión expire, `aws sso login --profile lab-dev`.
- Con una cuenta personal o acceso directo de consola: ejecuta `aws login --profile lab-dev` (AWS CLI v2.32 o posterior).

Antes de crear o actualizar recursos, confirma cuenta y rol:

```bash
aws sts get-caller-identity --profile lab-dev
```

Si exportas `AWS_PROFILE=lab-dev` en la terminal, los comandos y scripts de `scripts/aws/` usarán ese perfil sin recibir datos de autenticación.

Con la sesión confirmada, el camino automatizado desde `ejercicios/02-blog-publicacion/` es:

```bash
./scripts/aws/01-create-resources.sh
./scripts/aws/02-build-and-push-images.sh
./scripts/aws/03-deploy-lambdas-and-api.sh
./scripts/aws/04-seed-followers.sh # opcional
```

Los scripts son repetibles y piden confirmación. El despliegue crea tablas, colas, ECR, roles, Lambdas y API, pero no conecta todavía el Stream al fanout: la transición `DRAFT -> PUBLISHED` sigue pendiente de implementación.

Después de crear recursos, verifica:

```bash
aws dynamodb describe-table --table-name author_post
aws sqs get-queue-attributes --queue-url <fanout-jobs-url> --attribute-names All
aws lambda get-function --function-name follower-fanout-worker
aws lambda list-event-source-mappings --function-name follower-fanout-worker
```

## Imágenes Lambda en ECR

Cada Lambda tiene `Dockerfile` multi-stage y `.dockerignore`. La construcción
deja solo el runtime oficial Lambda `provided.al2023` y el binario estático
`bootstrap`: no se suben compilador, fuente, tests ni eventos de ejemplo.

Usa un repositorio ECR por función y tags inmutables, por ejemplo
`lambda-create-post:git-<sha>`. Desde `02-lambda/`:

```bash
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REPOSITORY=lambda-create-post
IMAGE_TAG=git-$(git rev-parse --short HEAD)
ECR_URI=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${REPOSITORY}

aws ecr create-repository --repository-name "$REPOSITORY" \
  --image-tag-mutability IMMUTABLE \
  --image-scanning-configuration scanOnPush=true
aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t "${ECR_URI}:${IMAGE_TAG}" functions/lambda_create_post
docker push "${ECR_URI}:${IMAGE_TAG}"
```

Repite cambiando `REPOSITORY` y directorio para `lambda-get-posts`,
`fanout-starter` y `follower-fanout-worker`. `--platform linux/arm64` debe
coincidir con la arquitectura Lambda. `--provenance=false` publica una imagen
única compatible con Lambda en lugar de una manifestación con attestation.

Para actualizar una función ya creada:

```bash
aws lambda update-function-code --function-name lambda-create-post \
  --image-uri "${ECR_URI}:${IMAGE_TAG}"
aws lambda wait function-updated --function-name lambda-create-post
```

Lambda y ECR deben estar en la misma región. Revisa el escaneo ECR antes de
promover una imagen y evita desplegar tags mutables como `latest`.

## Prueba funcional mínima

1. Crear DRAFT y comprobarlo mediante la consulta de drafts.
2. Insertar follows con `PK=FOLLOWING#author-1`.
3. Publicar mediante la transición real implementada.
4. Revisar `META` en `FanoutProgress`.
5. Comprobar batches de máximo 50 en `notification-jobs`.
6. Confirmar `FANOUT_COMPLETED`, logs CloudWatch y DLQ vacía.

## Datos de prueba y Postman

Para preparar followers para el fanout ejecuta, desde el ejercicio:

```bash
FOLLOWER_COUNT=250 AUTHOR_ID=author-1 ./scripts/local/seed-followers.sh
```

El script usa `http://localhost:8000` por defecto y credenciales ficticias: es exclusivamente para DynamoDB Local.
El script crea relaciones `FOLLOWING#author-1 / FOLLOWER#user-...`, que son las
que consulta el worker. La colección `postman/blog-publicacion.postman_collection.json`
prueba create draft y ambos GET; reemplaza `baseUrl` por la URL de HTTP API.

## Checklist

- [ ] Ninguna consulta GET usa Scan.
- [ ] GSI contiene solo drafts.
- [ ] Stream tiene imágenes nueva y anterior.
- [ ] Colas tienen DLQ y redrive policy.
- [ ] Worker tiene partial batch response y timeout menor al visibility timeout.
- [ ] IAM nombra recursos concretos.
- [ ] Retry retoma desde checkpoint y el writer es idempotente.
