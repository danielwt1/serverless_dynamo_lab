# 5. Despliegue y verificación

## Camino A: consola AWS

1. Crear tabla `users` con `PK` (String) como partition key, `SK` (String) como sort key y pago por demanda.
2. Crear el GSI llamado exactamente `GSI1`, con `GSI1PK` (String), `GSI1SK` (String) y proyección `INCLUDE` de `userId`, `name`, `lastName`, `email`, `state`, `created_at`.
3. Crear tres Lambdas desde sus Dockerfiles o artefactos: `USERS_TABLE_NAME=users` en create-user y `TABLE_NAME=users` en las otras dos.
4. Crear roles IAM mínimos y revisar logs CloudWatch.
5. Crear HTTP API y conectar rutas a sus Lambdas.
6. Probar create, login y listado activo.

Puedes guardar capturas en `documentacion/imagenes/`: `01-tabla-gsi.png`, `02-roles-lambda.png`, `03-api-rutas.png` y `04-cloudwatch-prueba.png`.

## Camino B: AWS CLI manual

Para AWS, primero verifica cuenta y región:

```bash
aws sts get-caller-identity
aws dynamodb describe-table --table-name users
```

Si la tabla todavía no existe y quieres crearla manualmente por CLI, usa esta versión en **una sola línea**. Así no dependes de `\` ni de continuaciones de Bash:

```bash
aws dynamodb create-table --table-name users --attribute-definitions '[{"AttributeName":"PK","AttributeType":"S"},{"AttributeName":"SK","AttributeType":"S"},{"AttributeName":"GSI1PK","AttributeType":"S"},{"AttributeName":"GSI1SK","AttributeType":"S"}]' --key-schema '[{"AttributeName":"PK","KeyType":"HASH"},{"AttributeName":"SK","KeyType":"RANGE"}]' --global-secondary-indexes '[{"IndexName":"GSI1","KeySchema":[{"AttributeName":"GSI1PK","KeyType":"HASH"},{"AttributeName":"GSI1SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"INCLUDE","NonKeyAttributes":["userId","name","lastName","email","state","created_at"]}}]' --billing-mode PAY_PER_REQUEST
```

Después espera a que DynamoDB termine de crear el índice antes de insertar datos:

```bash
aws dynamodb wait table-exists --table-name users
```

### Sesión de AWS CLI

No guardes access keys en archivos ni se las pases a scripts. Inicia sesión una vez y AWS CLI entrega credenciales temporales a todos los comandos posteriores.

- Si tu organización usa IAM Identity Center: configura el perfil una sola vez con `aws configure sso --profile lab-dev`; en cada sesión usa `aws sso login --profile lab-dev`.
- Si accedes directamente con una cuenta de consola personal o IAM/federación: usa `aws login --profile lab-dev` (AWS CLI v2.32 o posterior).

Después valida siempre el destino antes de desplegar:

```bash
aws sts get-caller-identity --profile lab-dev
```

Puedes definir `AWS_PROFILE=lab-dev` en tu terminal para no repetir `--profile`. Los comandos manuales no reciben ni almacenan claves, secretos ni tokens.

La [guía manual de AWS CLI](comandos-aws-cli.md) documenta los comandos, parámetros y verificaciones. Ejecútalos paso a paso para entender qué recurso crea cada uno.

Al convertirlo a infraestructura reproducible, reúne creación de tabla/GSI, roles, Lambdas y API en un único stack SAM, CDK, Terraform o CloudFormation; no mantengas comandos manuales dispersos como fuente de verdad.

## Imágenes Lambda en ECR

Cada función tiene un `Dockerfile` multi-stage y un `.dockerignore`. Compila un
binario Go estático para `linux/arm64` y deja en la imagen final únicamente el
runtime oficial Lambda `provided.al2023` y `bootstrap`. La imagen resultante no
lleva compilador, código fuente, tests ni eventos de ejemplo.

El lab usa el repositorio ECR compartido `users-service`; usa tags inmutables
que incluyan función y versión, por ejemplo `create-user-git-<sha>`. Desde `02-lambda/`:

```bash
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REPOSITORY=users-service
IMAGE_TAG=create-user-git-$(git rev-parse --short HEAD)
ECR_URI=${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${REPOSITORY}

aws ecr create-repository --repository-name "$REPOSITORY" \
  --image-tag-mutability IMMUTABLE \
  --image-scanning-configuration scanOnPush=true
aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t "${ECR_URI}:${IMAGE_TAG}" functions/create-user
docker push "${ECR_URI}:${IMAGE_TAG}"
```

`--platform linux/arm64` debe coincidir con la arquitectura configurada en la
Lambda. `--provenance=false` evita publicar una manifestación/attestation que
Lambda no puede ejecutar como una imagen única. Para las otras funciones cambia
`REPOSITORY` y el directorio de build.

Al crear una Lambda nueva usa `--package-type Image`; en una ya creada actualiza
solo la imagen:

```bash
aws lambda update-function-code --function-name create-user \
  --image-uri "${ECR_URI}:${IMAGE_TAG}"
aws lambda wait function-updated --function-name create-user
```

Después revisa el resultado del escaneo ECR y prueba la función. La Lambda y su
ECR deben estar en la misma región; no reutilices tags mutables como `latest`
para un despliegue reproducible.

## Prueba funcional

1. Crear `ana@example.com`; debe responder 201.
2. Repetir el mismo correo, incluso con mayúsculas/espacios; debe responder 409.
3. Autenticar con credenciales correctas y fallar con una incorrecta.
4. Crear más de 10 activos y comprobar `nextCursor`.
5. Usar el cursor en la siguiente petición y verificar que no repite la página.
6. Consultar logs y confirmar que no hay password ni detalles internos expuestos.

## Datos de prueba y Postman

La colección importable está en `postman/usuarios.postman_collection.json`; cambia
`baseUrl` por la URL de tu HTTP API. Sus credenciales demo son
`ana@example.com` / `PASSWORD`; son solo para el esqueleto actual y nunca para producción.

## Checklist

- [ ] GSI proyecta atributos de listado.
- [ ] La reserva de email y perfil se escriben en una transacción.
- [ ] Login tiene respuesta uniforme para credenciales inválidas antes de producción.
- [ ] Password se almacena como hash, nunca literal.
- [ ] IAM no usa permisos amplios.
- [ ] OpenAPI y handlers responden los mismos códigos HTTP.
