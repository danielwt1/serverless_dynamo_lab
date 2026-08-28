# CDK con TypeScript: laboratorio Usuarios

Esta carpeta expresa como infraestructura como código (IaC) el mismo servicio del ejercicio: una tabla DynamoDB con `GSI1`, tres Lambdas Go empaquetadas como imágenes Docker, roles IAM de mínimo privilegio, logs con tres días de retención y una HTTP API con sus tres rutas.

La guía [AWS CLI](../documentacion/comandos-aws-cli.md) sigue siendo el material para aprender qué crea cada servicio. CDK lo convierte en una definición declarativa, repetible y revisable. No ejecutes ambos mecanismos sobre los mismos recursos: los recursos manuales ya creados pertenecen a la guía CLI; este stack crea otra copia, con nombres físicos generados por CloudFormation.

## 1. Qué crea el stack y por qué

| Componente | Recurso CDK | Decisión del laboratorio | En producción |
| --- | --- | --- | --- |
| Persistencia | `dynamodb.Table` y `GSI1` | Claves `PK`/`SK`, demanda bajo pedido y la proyección necesaria para la lista de activos. La tabla se destruye con el laboratorio. | En producción usa `RETAIN`, protección contra borrado y PITR; generan costo y requieren una estrategia de ciclo de vida. |
| Imágenes | `DockerImageCode.fromImageAsset` | CDK construye las tres imágenes ARM64 y las carga al ECR privado creado por `cdk bootstrap`. | Usa digest o tags inmutables, escaneo, pipeline y separación por cuentas/ambientes. |
| Ejecución | `lambda.DockerImageFunction` | 256 MiB y 10 s, alineados con el despliegue manual. | Ajusta memoria/timeout con métricas de duración y costo. |
| IAM | Un rol por Lambda | Cada rol obtiene solo `PutItem`/`TransactWriteItems`, `GetItem` o `Query` sobre el ARN mínimo y escritura en su propio grupo de logs. | Mantén roles separados, revisa Access Analyzer y aplica permisos de despliegue aparte. |
| Observabilidad | `logs.LogGroup` | Logs de Lambda y de API Gateway con retención de 3 días; se destruyen con el stack. | Define la retención según cumplimiento, alarmas y una estrategia de auditoría. |
| HTTP | `HttpApi`, `HttpLambdaIntegration`, `HttpRoute`, `HttpStage` | HTTP API, proxy Lambda, formato de evento 2.0 y stage `$default`. CDK crea el permiso Lambda limitado al path de su ruta. | Añade autenticación, autorización, CORS restringido, límites de tráfico, WAF donde aplique, dominios y despliegues por ambiente. |

Las tres funciones no reciben acceso a ECR en sus roles. ECR interviene antes, durante `cdk deploy`: CDK publica la imagen y Lambda la obtiene mediante el servicio. El rol de ejecución solo necesita permisos para lo que hace el código en tiempo de ejecución.

## 2. Estructura y fuente de imágenes

```text
04-cdk/
├── bin/users-lab.ts          # Punto de entrada TypeScript de la aplicación CDK.
├── lib/config/               # Configuración y catálogo único de operaciones HTTP.
├── lib/constructs/           # Clases Construct de tabla, Lambda/IAM y HTTP API.
├── lib/users-lab-stack.ts    # Orquestador que conecta los constructos.
├── test/                     # Pruebas TypeScript de la plantilla sintetizada.
├── tsconfig.json             # Reglas estrictas de compilación TypeScript.
├── cdk.json                  # Indica a CDK cómo iniciar la aplicación.
└── package.json              # Dependencias y atajos npm.

../02-lambda/functions/
├── create-user/              # Contexto Docker de la primera imagen.
├── authenticate-user/        # Contexto Docker de la segunda imagen.
└── list-active-users/        # Contexto Docker de la tercera imagen.
```

`DockerImageCode.fromImageAsset` toma cada directorio Go como *contexto Docker*, utiliza su `Dockerfile`, construye para `linux/arm64` y publica el resultado como un asset. No necesitas ejecutar `docker login`, `docker build` ni `docker push` a mano para esta ruta CDK. Sí necesitas Docker funcionando dentro de WSL, porque CDK invoca Docker localmente.

El código fuente está en TypeScript y se compila con `tsc` a `dist/`. CDK ejecuta exclusivamente `dist/bin/users-lab.js`, definido en `cdk.json`; por eso cada comando de CDK compila antes. El compilador usa `strict` y `noUncheckedIndexedAccess`; no activa `exactOptionalPropertyTypes` porque las declaraciones de CDK 2.180 no son compatibles con esa opción. `.npmrc` contiene `ignore-scripts=true`: evita scripts automáticos de dependencias al instalar paquetes, pero no bloquea los comandos explícitos `npm run ...` que tú elijas ejecutar.

Los archivos de `lib/constructs/` son clases que extienden `Construct`. Un `Construct` agrupa recursos que cumplen una única responsabilidad y expone solo lo necesario a quien lo compone: `UsersTable` expone su tabla, `UsersLambda` su función y `UsersHttpApi` su API. Así, `UsersLabStack` se limita a conectar componentes y sigue el modelo recomendado por CDK: modelar con constructos y desplegar con stacks.

El repositorio usado por los assets es privado y pertenece al bootstrap de CDK; no es el repositorio manual `users-service`. Esta separación evita que la automatización cambie una imagen o un tag que usas en el recorrido manual.

## 3. Prerrequisitos y datos a consultar

Ejecuta los comandos desde WSL, situado en `ejercicios/01-usuarios/04-cdk`. No guardes la salida con IDs de cuenta o endpoints en archivos versionados.

```code
node --version
npm --version
./node_modules/.bin/tsc --version
docker version
aws sts get-caller-identity
aws configure get region
```

- `node --version` y `npm --version` comprueban Node.js y su gestor de paquetes. Usa una versión LTS vigente de Node.js; CDK, TypeScript y las dependencias se instalan en el proyecto, no globalmente.
- `./node_modules/.bin/tsc --version` confirma el compilador local usado para transformar TypeScript a JavaScript. No instala ni ejecuta recursos AWS.
- `docker version` verifica que el daemon de Docker está disponible. Las Lambdas son imágenes de contenedor y el despliegue las compila localmente.
- `aws sts get-caller-identity` confirma la cuenta e identidad que recibirán los recursos. No concede permisos ni genera costo.
- `aws configure get region` muestra la región por defecto. DynamoDB, Lambda, API Gateway, CloudWatch y los assets de ECR se crearán allí. Si no devuelve una región, configura el perfil antes de continuar.

El operador que ejecuta CDK necesita permisos de despliegue para CloudFormation, IAM, Lambda, DynamoDB, API Gateway, CloudWatch, S3 y ECR. Son permisos distintos de los roles de ejecución de Lambda. En un equipo real se otorgan a un rol de CI/CD con límites de permisos, no a cada Lambda.

## 4. Instalar dependencias y preparar CDK

Instala las dependencias declaradas en `package.json`:

```code
npm install
```

`npm install` descarga `aws-cdk-lib`, `constructs` y la CLI local `aws-cdk`. Crea `node_modules` y `package-lock.json`; `node_modules` está ignorado, mientras que el lockfile se debe versionar para instalaciones reproducibles.

Después inicializa una vez el entorno AWS destino:

```code
npx cdk bootstrap aws://ACCOUNT_ID_REAL/REGION_REAL
```

- Reemplaza `ACCOUNT_ID_REAL` por el campo `Account` de `aws sts get-caller-identity`.
- Reemplaza `REGION_REAL` por `aws configure get region`.
- `npx cdk bootstrap` crea o actualiza el stack `CDKToolkit`, además del bucket S3 y el repositorio privado ECR que CDK usa para assets. No despliega todavía el servicio de usuarios.
- Se ejecuta por cuenta y región, no por aplicación. Si ya existe un bootstrap compatible administrado por tu equipo, consulta su versión antes de modificarlo.

Durante el primer bootstrap es normal ver `CREATE_IN_PROGRESS` para `CDKToolkit`, un bucket S3 y los roles `CloudFormationExecutionRole`, `FilePublishingRole`, `ImagePublishingRole` y `LookupRole`. CDK crea un *changeset* de CloudFormation y después crea esos recursos; espera el mensaje final de éxito antes de continuar. Si termina en `CREATE_FAILED` o `ROLLBACK`, consulta los eventos del stack antes de reintentar:

```code
aws cloudformation describe-stack-events \
  --stack-name CDKToolkit \
  --max-items 20 \
  --output table
```

La CLI también puede avisar que el bootstrap usará `AdministratorAccess` como política de ejecución predeterminada. Esa política se asocia al rol que CloudFormation asume durante despliegues posteriores; no es el rol de ejecución de las Lambdas. Es aceptable para aprender en una cuenta aislada, pero en producción se debe ejecutar el bootstrap con una política de ejecución de mínimo privilegio aprobada por el equipo mediante `--cloudformation-execution-policies`.

En una cuenta compartida, no borres `CDKToolkit` solo porque termines este lab: puede alojar assets de otros stacks. La limpieza completa se explica al final y requiere comprobar primero que ningún otro proyecto dependa de ese bootstrap.

## 5. Sintetizar antes de cambiar AWS

```code
npm run build
npm test
npm run synth
npm run diff
```

- `npm run build` ejecuta el compilador TypeScript con modo estricto y deja los archivos generados en `dist/`. No crea recursos AWS.
- `npm run synth` compila TypeScript y transforma el resultado en una plantilla CloudFormation dentro de `cdk.out`. No crea recursos AWS. Revísala para relacionar cada constructo con su recurso final.
- `npm test` compila TypeScript, sintetiza el stack en memoria y verifica que el laboratorio sea eliminable, existan tres operaciones completas y el timeout de la API deje margen a Lambda. No crea recursos AWS ni construye imágenes Docker.
- `npm run diff` compara la plantilla sintetizada con el stack desplegado. Antes del primer despliegue mostrará recursos nuevos; en cambios posteriores muestra altas, bajas y modificaciones.

`cdk.out` es generado y no se versiona. En producción, ejecuta `diff` en la revisión que realmente aprobarás y protege los cambios de IAM y de eliminación de datos. Este laboratorio usa `DESTROY` y no habilita PITR ni protección contra borrado para que `cdk destroy` lo limpie por completo; no copies esa decisión a producción.

### Feature flags de CDK

Al sintetizar, la CLI puede avisar que hay *feature flags* sin configurar, por ejemplo: `82 feature flags are not configured`. No es un error ni impide el despliegue. Un feature flag es una opción versionada de CDK que habilita un comportamiento nuevo que podría cambiar la plantilla CloudFormation respecto de un proyecto anterior. Mientras no se declare en `cdk.json`, CDK conserva el comportamiento compatible que corresponda.

Para estudiarlos sin modificar archivos ni recursos AWS, consulta la lista local:

```code
npx cdk flags --unstable=flags
```

No copies todos los flags automáticamente al laboratorio. Primero identifica qué comportamiento cambia, agrega solo el flag que quieras adoptar a `cdk.json`, ejecuta `npm test`, `npm run synth` y `npm run diff`, y revisa el cambio de plantilla. En un proyecto nuevo o de producción se adopta esta decisión de forma explícita y se versiona junto con el código; los flags no son secretos ni variables de runtime de Lambda.

## 6. Desplegar el stack

Cuando el `diff` sea el esperado, despliega:

```code
npm run deploy
```

Este atajo ejecuta `cdk deploy` y hará, en orden conceptual:

1. Construir las tres imágenes con sus Dockerfiles para `linux/arm64`.
2. Publicarlas como assets privados de CDK en ECR.
3. Crear DynamoDB, roles, políticas y grupos de logs.
4. Crear Lambdas con las variables de entorno que apuntan a la tabla física generada.
5. Crear HTTP API, integraciones, rutas, stage `$default` y permisos de invocación restringidos por método y ruta.

Al finalizar, CDK imprime `ApiEndpoint` y `UsersTableName`. Copia el endpoint solo en tu terminal o en la variable local `baseUrl` de Postman; no lo escribas en este repositorio. El nombre de tabla se resuelve automáticamente dentro de las Lambdas, por lo que no debes modificar `TABLE_NAME` ni `USERS_TABLE_NAME` a mano.

Si solicita confirmar cambios de seguridad, léelos antes de aceptar. CDK detecta especialmente cambios de IAM porque pueden ampliar permisos. En automatización de producción se usan revisiones, aprobaciones y `cdk deploy --require-approval ...` según la política del equipo.

## 7. Verificar el despliegue

Primero consulta CloudFormation, que es la fuente de estado del stack CDK:

```code
aws cloudformation describe-stacks \
  --stack-name UsersLabStack \
  --query 'Stacks[0].Outputs' \
  --output table

aws cloudformation describe-stack-resources \
  --stack-name UsersLabStack \
  --query 'StackResources[*].[LogicalResourceId,ResourceType,ResourceStatus]' \
  --output table
```

El primer comando devuelve las salidas, incluido el endpoint. El segundo muestra recursos físicos y su estado; es la forma segura de averiguar nombres generados sin adivinarlos.

Usa el endpoint que obtuviste como `API_ENDPOINT_REAL` y prueba la ruta que no escribe datos:

```code
curl -i 'API_ENDPOINT_REAL/users/active?limit=10'
```

Debe responder HTTP `200` y un JSON con `items`. Después importa `../postman/usuarios.postman_collection.json` en Postman y actualiza solo la variable `baseUrl`. Las rutas son `POST /users`, `POST /auth/login` y `GET /users/active`.

Para diagnosticar sin exponer identificadores en documentación, identifica primero los grupos de log desde CloudFormation y luego consulta el nombre físico que devuelva el comando:

```code
aws cloudformation describe-stack-resources \
  --stack-name UsersLabStack \
  --query 'StackResources[?ResourceType==`AWS::Logs::LogGroup`].[LogicalResourceId,PhysicalResourceId]' \
  --output table

aws logs tail LOG_GROUP_NAME_REAL \
  --since 10m
```

La primera consulta no altera nada. En el segundo comando, reemplaza `LOG_GROUP_NAME_REAL` por el `PhysicalResourceId` de la fila que quieras revisar. Los cuatro grupos tienen retención de tres días.

## 8. Errores frecuentes

| Síntoma | Causa habitual | Cómo comprobar y corregir |
| --- | --- | --- |
| `docker: command not found` o no conecta al daemon | Docker no está instalado o iniciado en WSL. | Ejecuta `docker version`; instala/inicia Docker antes de repetir `npm run deploy`. |
| `This stack uses assets, so the toolkit stack must be deployed` | Falta bootstrap en cuenta/región destino. | Consulta cuenta y región, luego ejecuta el comando de la sección 4 con esos valores. |
| `AccessDenied` durante deploy | La identidad operadora no puede crear algún recurso o pasar un rol. | Consulta `aws sts get-caller-identity`; pide permisos de despliegue al responsable. No agregues esos permisos al rol de Lambda. |
| `cdk diff` muestra recursos manuales como inexistentes | CLI y CDK gestionan recursos distintos. | Es correcto: no intentes importar ni renombrar recursos de la guía manual durante este lab. |
| `ResourceInUseException` por logs | Una ejecución previa dejó un grupo con el mismo nombre físico. | Consulta el stack y el grupo. Elimina solo el recurso identificado si ya no pertenece a otro stack. |
| Error de Lambda al iniciar | El Dockerfile no deja `CMD ["bootstrap"]`. | Comprueba los tres Dockerfiles; para runtime `provided.al2023`, `bootstrap` debe ser el primer argumento. |

## 9. Inventario y limpieza del laboratorio CDK

Antes de borrar, verifica qué stack vas a afectar. Este paso es de solo lectura y evita confundir `UsersLabStack` con recursos manuales del mismo ejercicio:

```code
aws cloudformation describe-stacks \
  --stack-name UsersLabStack \
  --query 'Stacks[0].[StackName,StackStatus]' \
  --output table

aws cloudformation describe-stack-resources \
  --stack-name UsersLabStack \
  --query 'StackResources[*].[LogicalResourceId,ResourceType,PhysicalResourceId]' \
  --output table
```

El inventario esperado es una tabla DynamoDB y GSI, tres funciones, tres roles, políticas en línea, cuatro grupos de logs, una HTTP API, tres integraciones, tres rutas, un stage y permisos de Lambda. Las imágenes de Docker quedan en el ECR del bootstrap (`CDKToolkit`), no como un repositorio `users-service` creado por este stack.

Si esa lista corresponde exactamente al stack de práctica, destrúyelo:

```code
npm run destroy
```

CDK pedirá confirmación antes de enviar la eliminación a CloudFormation. Léela: este laboratorio elimina API e integraciones, Lambdas y permisos, logs, roles y tabla en el orden de sus dependencias. La tabla y los cuatro grupos de logs tienen `RemovalPolicy.DESTROY` solo porque este laboratorio es efímero; sus datos se perderán. En producción usa retención, copias de seguridad y un plan de borrado aprobado.

Cuando el comando termine, comprueba primero que CloudFormation ya no puede encontrar el stack:

```code
aws cloudformation describe-stacks \
  --stack-name UsersLabStack
```

La respuesta esperada es `ValidationError` indicando que el stack no existe. Es el comprobante principal: CloudFormation solo elimina el stack cuando finalizó la eliminación de sus recursos administrados.

Como comprobación adicional, consulta los recursos de la región con los tags que este stack aplica. La salida esperada es una lista vacía (`ResourceTagMappingList: []`):

```code
aws resourcegroupstaggingapi get-resources \
  --tag-filters Key=system,Values=users Key=managed-by,Values=cdk \
  --query 'ResourceTagMappingList[*].ResourceARN' \
  --output table
```

Este segundo comando es solo de lectura. Si muestra algún ARN, no lo borres automáticamente: primero confirma si pertenece a una ejecución anterior del laboratorio o a otro recurso que use los mismos tags.

La destrucción del stack no borra los assets Docker que CDK pudo publicar en el ECR del bootstrap, ni el propio bootstrap. Es intencional: esos assets y `CDKToolkit` pueden ser compartidos por otros stacks. Para revisar el bootstrap sin cambiarlo, consulta:

```code
aws cloudformation describe-stack-resources \
  --stack-name CDKToolkit \
  --query 'StackResources[*].[LogicalResourceId,ResourceType,PhysicalResourceId]' \
  --output table
```

No destruyas `CDKToolkit` ni vacíes su ECR salvo que hayas comprobado que ningún otro stack CDK de la cuenta/región los usa y que no necesitas conservar assets. Es infraestructura compartida de CDK, no un recurso propio de Usuarios. Para dejar una cuenta completamente vacía, consulta también el inventario de la sección de limpieza del manual CLI.
