# AGENTS.md

## Propósito del repositorio

Este repositorio es un laboratorio práctico para aprender arquitectura serverless en AWS de forma incremental. Cada ejercicio debe poder entenderse, ejecutarse y verificarse sin ocultar decisiones importantes detrás de automatizaciones prematuras.

El ejercicio `ejercicios/01-usuarios` construye un servicio de usuarios con DynamoDB, imágenes Docker en ECR, funciones Lambda, API Gateway e IAM. La guía operativa principal es `ejercicios/01-usuarios/documentacion/comandos-aws-cli.md`.

## Forma de trabajar

- Antes de cambiar recursos AWS, consulta el estado actual con AWS CLI cuando sea posible; después realiza el cambio y finalmente verifica su resultado.
- Avanza por capas: DynamoDB e IAM → imagen en ECR → una Lambda y su prueba → las demás Lambdas → API Gateway e integración → permisos de invocación → pruebas HTTP.
- Trabaja una Lambda a la vez. No des por creada, activa o integrada una función solo porque su imagen exista en ECR.
- No ejecutes despliegues, `docker push`, invocaciones que escriban datos, eliminaciones ni cambios de IAM por iniciativa propia. Primero documenta el comando, explica el efecto y deja que la persona usuaria lo ejecute, salvo que lo solicite expresamente.
- Conserva los recursos y cambios existentes. No uses comandos destructivos como `delete-*`, `rm -rf`, `git reset --hard` o equivalentes sin una instrucción explícita y un objetivo exacto.

## Convenciones de la documentación AWS CLI

- Escribe las explicaciones en español claro, con enfoque didáctico y orden reproducible: propósito, prerrequisitos, consulta, cambio, verificación, errores frecuentes y consideraciones de producción.
- Cada comando debe ir en un bloque Markdown marcado como `code`, nunca como `bash`.
- Para un comando aislado se puede usar un solo bloque `code`; cuando haya varios comandos relacionados, agrúpalos en un mismo bloque `code` y explica el orden.
- No uses `export`, sustituciones de comando como `$()` ni IDs de cuenta, ARN, regiones o credenciales reales en documentación. Usa marcadores explícitos: `ACCOUNT_ID_REAL`, `REGION_REAL`, `ROLE_ARN_REAL`, `FUNCTION_NAME_REAL` o equivalentes.
- Junto a cada marcador, indica de dónde se obtiene con un comando de consulta y qué valor debe reemplazar la persona usuaria. Los ejemplos de cuenta deben ser ficticios.
- Explica todos los parámetros que no sean obvios: qué hacen, por qué se eligieron para el laboratorio, qué cambiaría en producción y qué efecto o costo pueden tener.
- Distingue tags de recursos AWS (`--tags`) de tags de imágenes ECR (`create-user-v1`). Para el laboratorio usa versiones simples e inmutables (`v1`, `v2`); documenta SHA de Git y digest de ECR como estrategia de trazabilidad de producción.
- Mantén los comandos orientados a Bash/WSL. La persona usuaria ejecuta Docker desde WSL.
- No copies respuestas reales de AWS CLI que contengan identificadores de una cuenta. Resume la salida esperada usando nombres o valores ficticios.

## Seguridad y producción

- Aplica mínimo privilegio: una Lambda por rol de ejecución y una política DynamoDB limitada a la acción y recurso que realmente necesita.
- Separa permisos del operador o pipeline de despliegue de los permisos del rol que ejecuta Lambda.
- ECR es privado en este laboratorio. No confundas acceso privado al repositorio con exposición pública de una Lambda o de API Gateway.
- No incluyas secretos, contraseñas reales, tokens, datos personales ni valores de `.env` en documentación, fixtures o comandos. Las variables de entorno de Lambda no son un sustituto de Secrets Manager o Parameter Store para secretos de producción.
- Indica efectos de costo y de seguridad antes de recursos persistentes o configuraciones como Provisioned Concurrency, logs con retención amplia, NAT Gateway, KMS administrado por el cliente o API pública.

## Código, pruebas y cambios locales

- Cada Lambda Go es un módulo independiente en `ejercicios/01-usuarios/02-lambda/functions/<funcion>`; ejecuta pruebas con `go -C <carpeta> test ./...` cuando Go esté instalado y su versión sea compatible con el `go.mod` y Dockerfile.
- Revisa el contrato fuente antes de cambiar handlers o rutas: `ejercicios/01-usuarios/03-api/contracts/openapi.yaml`.
- Mantén coherentes código, políticas IAM, eventos de prueba, Dockerfile, contrato OpenAPI y guía CLI cuando un cambio los afecte.
- Usa `apply_patch` para editar archivos locales y valida JSON/YAML/documentación modificada con verificaciones no destructivas apropiadas.

## Fuente de verdad y automatización futura

- La guía CLI existe para aprender y depurar. Cuando un ejercicio migre a CDK u otra infraestructura como código, esa herramienta debe convertirse en la única fuente de verdad para producción.
- No presentes un comando manual y CDK como dos mecanismos que deban operar simultáneamente sobre el mismo recurso de producción.
