# Guía manual de AWS CLI: laboratorio Usuarios

Esta es la bitácora de infraestructura del laboratorio. No hay scripts de despliegue: crea, verifica y entiende cada recurso con AWS CLI.

Los ejemplos son para **Bash/WSL**. Cada `\` debe ser el último carácter de su línea, sin espacios después. El JSON entre comillas simples puede ocupar varias líneas.

Primero autentícate en AWS CLI:

```code
aws login
```

Antes de crear recursos, valida identidad y región:

```code
aws sts get-caller-identity
aws configure list
```

`sts get-caller-identity` no hace login: solo muestra la cuenta y el usuario o rol activos. `aws configure list` muestra la región que AWS CLI resolverá y de dónde viene (perfil, variable de entorno o argumento). Si no aparece una región, define una con `--region <region>` en cada comando. Si manejas varios perfiles, usa `aws login --profile <perfil>` y añade el mismo `--profile <perfil>` a los comandos.

## Convención de etiquetas (*tags*)

Los *tags* de recursos son pares de metadatos, por ejemplo `environment=dev`. Sirven para reconocer quién es responsable de un recurso, a qué sistema pertenece y en qué entorno se ejecuta; **no otorgan permisos por sí mismos**. AWS CLI no usa una única sintaxis para todos los servicios: Lambda y CloudWatch Logs reciben un mapa (`environment=dev,...`), mientras IAM y ECR reciben una lista de objetos (`Key=environment,Value=dev ...`). Los comandos de cada sección usan el formato que exige su servicio.

```code
--tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

En producción podrías usar:

```code
--tags environment=prod,system=users,owner=team-backend,cost-center=engineering,managed-by=cdk,data-classification=internal
```

| Tag | Ejemplo producción | Por qué usarlo |
| --- | --- | --- |
| `environment` | `prod` | Evita confundir recursos de desarrollo y producción al operar o automatizar. |
| `system` | `users` | Agrupa Lambda, ECR y DynamoDB que forman el mismo servicio. |
| `owner` | `team-backend` | Define el equipo que mantiene el recurso y atiende incidencias. |
| `cost-center` | `engineering` | Permite agrupar costos de recursos facturables. Debe activarse como *cost allocation tag* en Billing para aparecer en Cost Explorer. |
| `managed-by` | `cdk` | Muestra qué herramienta es la fuente de verdad; en este lab es `manual-cli`. |
| `data-classification` | `internal` | Ayuda a aplicar controles distintos a datos públicos, internos o confidenciales. |

En una organización, esta convención permite inventario, reportes de costos, reglas de AWS Config y controles basados en atributos (ABAC). En ECR también existen **tags de imagen**, como `create-user-v1`: esos identifican una versión del artefacto que despliegas y no son los mismos que los tags del repositorio.

## 1. DynamoDB: tabla e índice

### Crear la tabla `users`

```code
aws dynamodb create-table \
  --table-name users \
  --attribute-definitions '[
    {"AttributeName":"PK","AttributeType":"S"},
    {"AttributeName":"SK","AttributeType":"S"},
    {"AttributeName":"GSI1PK","AttributeType":"S"},
    {"AttributeName":"GSI1SK","AttributeType":"S"}
  ]' \
  --key-schema '[
    {"AttributeName":"PK","KeyType":"HASH"},
    {"AttributeName":"SK","KeyType":"RANGE"}
  ]' \
  --global-secondary-indexes '[
    {
      "IndexName":"GSI1",
      "KeySchema":[
        {"AttributeName":"GSI1PK","KeyType":"HASH"},
        {"AttributeName":"GSI1SK","KeyType":"RANGE"}
      ],
      "Projection":{
        "ProjectionType":"INCLUDE",
        "NonKeyAttributes":["userId","name","lastName","email","state","created_at"]
      }
    }
  ]' \
  --billing-mode PAY_PER_REQUEST
```

Qué hace: crea la tabla y el GSI requeridos por los patrones de acceso del lab.

- `--table-name`: nombre físico de la tabla; debe coincidir con la variable de entorno de las Lambdas.
- `--attribute-definitions`: declara solo atributos que son claves de tabla o de índices, no el esquema completo de un ítem.
- `--key-schema`: `PK` es la partición (`HASH`) y `SK` el ordenamiento (`RANGE`).
- `--global-secondary-indexes`: crea `GSI1` con sus dos claves. La proyección `INCLUDE` copia solo atributos necesarios para listar sin hacer lecturas extra.
- `--billing-mode PAY_PER_REQUEST`: pago bajo demanda; apropiado para el lab y tráfico impredecible. Si eliges `PROVISIONED`, debes indicar capacidad para tabla e índice.

Otras opciones útiles de `create-table`: `--tags` para costos/gobierno, `--stream-specification` para DynamoDB Streams, `--sse-specification` para cifrado con KMS y `--table-class STANDARD_INFREQUENT_ACCESS` para datos con acceso poco frecuente. No son necesarias aún aquí.

### Esperar y comprobar

```code
aws dynamodb wait table-exists --table-name users

aws dynamodb describe-table \
  --table-name users \
  --query 'Table.TableStatus' \
  --output text
```

El *waiter* consulta hasta que la tabla exista y, si tiene éxito, normalmente no imprime nada. El segundo comando debe responder `ACTIVE`. `--query` extrae ese campo con JMESPath y `--output text` evita el JSON completo.

Referencia: [AWS CLI `dynamodb create-table`](https://docs.aws.amazon.com/cli/latest/reference/dynamodb/create-table.html).

## 2. ECR: repositorio de imágenes Lambda

### Crear un repositorio

El repositorio real de este lab es `users-service`:

```code
aws ecr create-repository \
  --repository-name users-service \
  --image-tag-mutability IMMUTABLE \
  --encryption-configuration encryptionType=AES256 \
  --tags Key=lab,Value=usuarios Key=managed-by,Value=manual-cli
```

Qué hace: crea un repositorio privado de ECR donde publicarás la imagen que Lambda ejecuta.

- `--repository-name`: nombre. Este lab usa un único repositorio, `users-service`. En ese caso los tags deben identificar función y versión, por ejemplo `create-user-v1`. Una alternativa futura es un repositorio por función para separar versiones y permisos. Solo admite minúsculas, números, `.`, `_`, `-` y `/`.
- `--image-tag-mutability IMMUTABLE`: un tag ya publicado no se puede reemplazar. Usa tags versionados, como `git-a1b2c3d`; no `latest` para despliegues reproducibles.
- `--encryption-configuration encryptionType=AES256`: cifrado en reposo administrado por AWS. Para control de claves puedes elegir `encryptionType=KMS,kmsKey=<arn-o-alias>`; esa clave debe existir en la misma región.
- `--tags`: metadatos del repositorio. Sigue la [convención de etiquetas](#convención-de-etiquetas-tags); ECR sí es facturable, por lo que `cost-center` es útil para reportes de costo.

Opciones disponibles que valen la pena conocer:

- `--registry-id <cuenta>`: crea en otro registro; al omitirlo se usa la cuenta autenticada.
- `--image-tag-mutability MUTABLE`: deja reemplazar tags; puede ser útil en desarrollo efímero, pero no en artefactos que despliegas.
- `--image-tag-mutability-exclusion-filters`: permite excepciones por patrón, por ejemplo un `latest` mutable. Úsalo solo con una razón clara.
- `--image-scanning-configuration scanOnPush=true`: existe, pero AWS recomienda mover el escaneo a configuración a nivel de *registry* para producción.

Referencia: [AWS CLI `ecr create-repository`](https://docs.aws.amazon.com/cli/latest/reference/ecr/create-repository.html).

### Retención: política de ciclo de vida

Sin una política, ECR acumula imágenes y costo. Este lab conserva las 20 más recientes y expira las anteriores. La política está lista en `documentacion/ecr-lifecycle-policy.json`:

```json
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Conservar solo las 20 imagenes mas recientes",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": 20
      },
      "action": {
        "type": "expire"
      }
    }
  ]
}
```

- `rulePriority: 1`: prioridad más alta; los números menores se procesan primero.
- `tagStatus: any`: considera imágenes con y sin tag.
- `imageCountMoreThan` + `20`: conserva las 20 más nuevas y expira las antiguas según su fecha de publicación.
- `expire`: elimina las imágenes seleccionadas; revisa siempre antes su efecto.

Previsualiza y luego aplica la política:

`--lifecycle-policy-text` espera el **contenido JSON**, no una ruta normal. El prefijo `file://` le indica a AWS CLI que lea ese contenido desde un archivo. Como los comandos se ejecutan desde `ejercicios/01-usuarios`, la ruta relativa correcta es `file://documentacion/ecr-lifecycle-policy.json`.

No uses una ruta de Windows como `C:\\Users\\...` en WSL/Bash: sin `file://` AWS CLI intenta interpretarla como JSON y falla. Si necesitas una ruta absoluta en WSL, sería `file:///mnt/c/Users/USUARIO/.../ecr-lifecycle-policy.json`.

```code
aws ecr start-lifecycle-policy-preview \
  --repository-name users-service \
  --lifecycle-policy-text file://documentacion/ecr-lifecycle-policy.json

aws ecr get-lifecycle-policy-preview \
  --repository-name users-service

aws ecr put-lifecycle-policy \
  --repository-name users-service \
  --lifecycle-policy-text file://documentacion/ecr-lifecycle-policy.json
```

`put-lifecycle-policy` crea o reemplaza la política; es seguro repetirlo si el JSON es el que quieres. ECR puede tardar hasta 24 horas en expirar una imagen que cumple la regla. En producción separa tags temporales (`dev-*`) de releases (`release-*` o `git-*`) y evita eliminar una imagen que sigue desplegada por una versión de Lambda.

### Editar, previsualizar y publicar: ciclo completo

Hay tres estados distintos que conviene no confundir:

1. **Archivo local:** `documentacion/ecr-lifecycle-policy.json`. Puedes modificarlo todas las veces que quieras; todavía no cambia nada en AWS.
2. **Vista previa:** `start-lifecycle-policy-preview`. AWS evalúa el JSON propuesto y muestra qué imágenes *expiraría*. Tampoco guarda una política en el repositorio.
3. **Política publicada:** `put-lifecycle-policy`. Este sí guarda la política en `users-service` y ECR la aplicará a futuro.

ECR permite **una sola política de ciclo de vida por repositorio**, pero esa política contiene una o varias reglas:

```text
Repositorio users-service
└── Una política de ciclo de vida
    ├── Regla con prioridad 1
    ├── Regla con prioridad 2
    └── Regla con prioridad 3
```

Por eso `put-lifecycle-policy` reemplaza el documento de política completo: no crea una segunda política ni agrega automáticamente una regla. AWS debe recibir todas las reglas juntas para validarlas y resolver sus prioridades.

Antes de publicarla, el flujo es siempre editar archivo → preview → revisar → `put`. Por ejemplo, para conservar solo 10 imágenes, cambia en el archivo:

```json
"countNumber": 10
```

y vuelve a ejecutar el preview. En esta regla puedes cambiar principalmente:

- `countNumber`: cuántas imágenes recientes conservar.
- `tagStatus`: `any`, `tagged` o `untagged`.
- `countType`: `imageCountMoreThan` para retener por cantidad, o `sinceImagePushed` para expirar por antigüedad. Este último requiere además `countUnit: "days"` y un `countNumber` de días.
- Si usas `tagStatus: "tagged"`, puedes filtrar tags con `tagPrefixList` o `tagPatternList`; sirven para dar reglas distintas a `dev-*` y `release-*`.

Si el JSON contiene una única regla y cambias solo `countNumber` de `20` a `10`, al ejecutar `put` seguirá existiendo una sola regla: ahora conservará 10 imágenes. No se crea otra regla porque el arreglo `rules` continúa teniendo un solo objeto.

Para añadir una segunda regla, conserva la primera y añade un segundo objeto dentro de `rules`. Cada `rulePriority` debe ser único: `1, 2, 3` es válido; `1, 1, 2` no lo es. La prioridad `1` es la más alta; si una imagen coincide con varias reglas, la de menor número prevalece. Coloca reglas específicas primero y las generales después. Una regla con `tagStatus: "any"` es muy amplia y debe tener la prioridad numéricamente más alta.

La acción de una lifecycle policy de ECR es `expire`; por eso el preview es importante: no existe una acción de “archivar” que puedas deshacer desde la misma regla.

Si ya está publicada, primero puedes leer lo que tiene AWS:

```code
aws ecr get-lifecycle-policy \
  --repository-name users-service \
  --query 'lifecyclePolicyText' \
  --output text
```

Luego modifica tu archivo local, previsualiza el cambio y ejecuta otra vez `put-lifecycle-policy`. No hace falta borrarla antes: `put` **reemplaza por completo** la política anterior por el JSON nuevo. Si quieres eliminar la política de retención, no modificarla, usa:

En consecuencia, si ya tenías tres reglas y envías un JSON que contiene solo una, las otras dos desaparecen de la política publicada. Antes de modificarla, usa `get-lifecycle-policy` y asegúrate de que tu archivo local incluya todas las reglas que deseas conservar.

```code
aws ecr delete-lifecycle-policy \
  --repository-name users-service
```

Eso detiene futuras expiraciones programadas; no recupera imágenes que ECR ya haya eliminado.

Referencias: [AWS CLI `ecr put-lifecycle-policy`](https://docs.aws.amazon.com/cli/latest/reference/ecr/put-lifecycle-policy.html) y [funcionamiento de lifecycle policies](https://docs.aws.amazon.com/AmazonECR/latest/userguide/LifecyclePolicies.html).

## 3. IAM: políticas, roles y asociaciones

Una Lambda necesita un **rol de ejecución** para llamar servicios AWS. El modelo del lab tiene tres capas: primero definimos permisos en archivos JSON, luego creamos las políticas y los roles, y al final asociamos cada política al rol que la necesita.

Hay cuatro documentos JSON, aunque solo tres son políticas de permisos:

| Archivo | Tipo | Para qué sirve |
| --- | --- | --- |
| `policy_lambda_asume_rol.json` | Política de confianza (*trust policy*) | Permite que el servicio `lambda.amazonaws.com` asuma un rol. No concede acceso a DynamoDB. |
| `policy_create_user_dynamo.json` | Política de permisos | Permite `dynamodb:TransactWriteItems` y los `dynamodb:PutItem` que la transacción ejecuta sobre `users`. |
| `policy_authenticate_user_dynamo.json` | Política de permisos | Permite `dynamodb:GetItem` sobre `users`. |
| `policy_list_active_users_dynamo.json` | Política de permisos | Permite `dynamodb:Query` solo sobre `users/index/GSI1`. |

No compartas un rol entre las tres funciones: si `authenticate-user` recibe el permiso `Query`, podría consultar el índice aun cuando su código no lo hace. Un rol por función aplica mínimo privilegio y limita el impacto de un error o una credencial comprometida.

Los tres JSON de DynamoDB son **plantillas públicas**: contienen `<AWS_REGION>` y `<AWS_ACCOUNT_ID>`, nunca valores de una cuenta personal. Primero consulta y anota los valores de tu sesión activa:

```code
aws configure get region
aws sts get-caller-identity --query Account --output text
```

Con esos valores, crea tres copias temporales en `/tmp`. En cada comando sustituye `REGION_REAL` y `ACCOUNT_ID_REAL` por lo que acabas de consultar; no uses esos textos literalmente ni guardes los valores en Git.

```code
sed -e 's/<AWS_REGION>/REGION_REAL/g' -e 's/<AWS_ACCOUNT_ID>/ACCOUNT_ID_REAL/g' ejercicios/01-usuarios/documentacion/policy/policy_authenticate_user_dynamo.json > /tmp/policy_authenticate_user_dynamo.json
sed -e 's/<AWS_REGION>/REGION_REAL/g' -e 's/<AWS_ACCOUNT_ID>/ACCOUNT_ID_REAL/g' ejercicios/01-usuarios/documentacion/policy/policy_create_user_dynamo.json > /tmp/policy_create_user_dynamo.json
sed -e 's/<AWS_REGION>/REGION_REAL/g' -e 's/<AWS_ACCOUNT_ID>/ACCOUNT_ID_REAL/g' ejercicios/01-usuarios/documentacion/policy/policy_list_active_users_dynamo.json > /tmp/policy_list_active_users_dynamo.json
```

### 3.1 Crear las políticas administradas de DynamoDB

Una política administrada por el cliente es un recurso IAM reutilizable con su propio ARN. En este laboratorio creamos una por patrón de acceso; después cada una se asocia solamente a su rol. Ejecuta una vez cada comando:

```code
aws iam create-policy \
  --policy-name lambda-users-read-getitem \
  --policy-document file:///tmp/policy_authenticate_user_dynamo.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam create-policy \
  --policy-name lambda-users-create-transaction \
  --policy-document file:///tmp/policy_create_user_dynamo.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam create-policy \
  --policy-name lambda-users-query \
  --policy-document file:///tmp/policy_list_active_users_dynamo.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli
```

Los ARN resultantes usan la cuenta activa, por ejemplo `arn:aws:iam::<AWS_ACCOUNT_ID>:policy/lambda-users-query`. No copies un ARN de otra cuenta: una política administrada de cliente pertenece a una cuenta concreta.

Si ejecutas `create-policy` con un nombre que ya existe, IAM responde `EntityAlreadyExists`; no es necesario crearla otra vez. Para cambiar una política administrada existente se crea una versión nueva con `aws iam create-policy-version --set-as-default`; no se modifica la versión actual directamente. Comprueba siempre cuál versión quedó predeterminada antes de asociarla a un rol.

Verifica las tres antes de crear roles:

```code
aws iam list-policies \
  --scope Local \
  --query 'Policies[?starts_with(PolicyName, `lambda-users-`)].[PolicyName,Arn,DefaultVersionId,AttachmentCount]' \
  --output table
```

En producción, los nombres suelen incorporar sistema y entorno, por ejemplo `users-prod-create-transaction`. Aplica la [convención de etiquetas](#convención-de-etiquetas-tags) a las políticas y recursos facturables. IAM también permite *paths*, descripciones y límites de permisos (*permissions boundaries*) al crear políticas. Evita `Action: "dynamodb:*"` y `Resource: "*"`: son cómodos al inicio, pero eliminan el control de mínimo privilegio.

Las tres políticas ya creadas pueden recibir tags después, sin recrearlas:

```code
aws iam tag-policy \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-read-getitem \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam tag-policy \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-create-transaction \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam tag-policy \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-query \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli
```

### 3.2 Crear los roles de ejecución

El rol contiene la trust policy. Los tres roles reutilizan `policy_lambda_asume_rol.json`, porque todos deben ser asumidos por Lambda:

```code
aws iam create-role \
  --role-name lambda-create-user-role \
  --assume-role-policy-document file://ejercicios/01-usuarios/documentacion/policy/policy_lambda_asume_rol.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam create-role \
  --role-name lambda-authenticate-user-role \
  --assume-role-policy-document file://ejercicios/01-usuarios/documentacion/policy/policy_lambda_asume_rol.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli

aws iam create-role \
  --role-name lambda-list-active-users-role \
  --assume-role-policy-document file://ejercicios/01-usuarios/documentacion/policy/policy_lambda_asume_rol.json \
  --tags Key=environment,Value=dev Key=system,Value=users Key=owner,Value=backend Key=managed-by,Value=manual-cli
```

Cada Lambda usa su propio rol. Si las funciones están en una VPC, el rol también requiere los permisos de red apropiados; no los agregues todavía porque este laboratorio no configura VPC.

### 3.3 Asociar las políticas a cada rol y verificar

Toda Lambda necesita logs; por eso los tres roles reciben la política administrada por AWS `AWSLambdaBasicExecutionRole`. Luego cada uno recibe únicamente la política de DynamoDB correspondiente:

```code
aws iam attach-role-policy \
  --role-name lambda-create-user-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam attach-role-policy \
  --role-name lambda-create-user-role \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-create-transaction

aws iam attach-role-policy \
  --role-name lambda-authenticate-user-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam attach-role-policy \
  --role-name lambda-authenticate-user-role \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-read-getitem

aws iam attach-role-policy \
  --role-name lambda-list-active-users-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam attach-role-policy \
  --role-name lambda-list-active-users-role \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-query
```

`attach-role-policy` no imprime salida cuando tiene éxito. Verifica exactamente los dos permisos de cada rol; cada consulta debe listar `AWSLambdaBasicExecutionRole` y una única política `lambda-users-*`:

```code
aws iam list-attached-role-policies --role-name lambda-create-user-role
aws iam list-attached-role-policies --role-name lambda-authenticate-user-role
aws iam list-attached-role-policies --role-name lambda-list-active-users-role

aws iam get-role \
  --role-name lambda-create-user-role \
  --query 'Role.[RoleName,Arn]' \
  --output table
```

Conserva el ARN de `lambda-create-user-role`; será el valor de `--role` al crear la primera Lambda.

### Checklist para terminar `create-user`

El rol es solo un requisito. La primera Lambda estará terminada cuando cumplas esta secuencia:

1. Políticas `lambda-users-*` creadas y rol `lambda-create-user-role` asociado a `AWSLambdaBasicExecutionRole` y `lambda-users-create-transaction`.
2. Imagen `create-user-<versión>` construida y publicada en ECR `users-service`.
3. Lambda `create-user` creada desde esa imagen, con `--package-type Image`, arquitectura `arm64` y el ARN del rol.
4. Variable de entorno `USERS_TABLE_NAME=users` configurada. Esta función no usa `TABLE_NAME`.
5. Invocación de prueba exitosa y logs revisados en `/aws/lambda/create-user`.

Para saber qué falta sin adivinar, consulta cada recurso. Si `get-function` responde `ResourceNotFoundException`, todavía no existe la Lambda; si devuelve JSON, revisa imagen, rol y variables con el mismo comando:

```code
aws lambda get-function \
  --function-name create-user \
  --query 'Configuration.[FunctionName,State,Role,Architectures,Environment.Variables]' \
  --output json

aws ecr describe-images \
  --repository-name users-service \
  --query 'imageDetails[*].[imageTags,imagePushedAt]' \
  --output table
```

Para producción, aplica la misma convención de tags a Lambda, separa permisos de despliegue de permisos de ejecución, configura alertas de CloudWatch y usa *permission boundaries* o SCP si tu organización los utiliza.

## 4. Construir y publicar las imágenes en ECR

En este paso construimos y publicamos las tres imágenes, pero todavía no creamos ni invocamos ninguna Lambda. El repositorio privado es `users-service`; los tres artefactos vivirán allí con tags distintos.

Todo se ejecuta en **Bash/WSL**, desde la raíz del repositorio. No usamos `export` ni sustituciones automáticas: primero consultas los datos con CLI, los anotas y después reemplazas manualmente los marcadores de los comandos.

### 4.1 Datos que debes consultar y anotar

Ejecuta estos comandos de solo lectura, uno por uno:

```code
aws sts get-caller-identity --query Account --output text
```

```code
aws configure get region
```

Anota los dos resultados con estos nombres; los usaremos más abajo. Para este primer despliegue usa también la versión simple `v1`:

| Marcador | De dónde sale | Ejemplo ficticio | Qué representa |
| --- | --- | --- | --- |
| `ACCOUNT_ID_REAL` | primer comando | `123456789012` | Cuenta AWS que posee ECR, Lambda e IAM. |
| `REGION_REAL` | segundo comando | `us-east-1` | Región del repositorio ECR y de las futuras Lambdas. Debe ser la misma para ambos. |
| `VERSION` | la eliges tú | `v1` | Versión simple de la imagen que vas a publicar. |

Si mañana publicas contenido nuevo, usa `v2`, luego `v3`, y así sucesivamente. ECR tiene tags inmutables, por lo que no permite publicar contenido distinto con el mismo tag.

#### Estrategia de tags: laboratorio frente a producción

Para este laboratorio elegimos una estrategia deliberadamente simple: `create-user-v1`, `authenticate-user-v1` y `list-active-users-v1`. En la siguiente publicación cambias solo la versión a `v2`. Es fácil de leer, funciona con mutabilidad `IMMUTABLE` y permite aprender el flujo sin introducir Git en los comandos de Docker.

En producción, normalmente conviene que el tag identifique el **código exacto** que generó la imagen. La práctica más común es incluir el SHA corto del commit de Git, por ejemplo `create-user-git-a1b2c3d`, donde `a1b2c3d` representa los primeros caracteres del identificador del commit. Git genera ese identificador a partir del contenido e historial del repositorio: si cambia el código, el SHA cambia. No es un secreto y no se inventa; se consulta en el repositorio o lo entrega el sistema de CI/CD que ejecutó el build.

Si estuvieras haciendo una publicación manual de producción, el SHA corto se consulta desde la raíz de un repositorio Git con este único comando:

```code
git rev-parse --short HEAD
```

- `git`: cliente de control de versiones; consulta el repositorio local, no AWS.
- `rev-parse`: resuelve una referencia de Git a su identificador.
- `HEAD`: el commit que tienes actualmente seleccionado en tu rama local.
- `--short`: muestra una forma abreviada legible del SHA, por ejemplo `a1b2c3d`, en vez del identificador completo.

Con una salida ficticia `a1b2c3d`, el tag de producción para `create-user` sería `create-user-git-a1b2c3d`. No ejecutes ese comando ni sustituyas los tags de este laboratorio por el SHA: aquí seguimos con `create-user-v1`. En CI/CD el pipeline debe obtener el SHA del commit que lo activó, no necesariamente el `HEAD` de la computadora de alguien; así se garantiza que la imagen corresponde al código revisado y fusionado.

| Estrategia | Ejemplo | Cuándo usarla | Ventaja principal | Límite que debes conocer |
| --- | --- | --- | --- | --- |
| Versión manual | `create-user-v1` | Laboratorio, prueba controlada o una primera entrega pequeña. | Muy legible y fácil de introducir a mano. | No indica por sí sola qué commit contiene; depende de que documentes la relación. |
| Release semántico | `create-user-v1.4.0` | Producto con releases planificados y notas de versión. | Comunica compatibilidad y versión de negocio. | Varias correcciones de compilación del mismo release deben seguir siendo trazables. |
| SHA corto de Git | `create-user-git-a1b2c3d` | Producción y CI/CD. | Une de forma directa imagen, código revisado, pipeline y despliegue. Facilita auditoría y rollback. | Es menos amigable para una persona; hay que consultar Git o el pipeline para interpretarlo. |
| Release + SHA | `create-user-v1.4.0-git-a1b2c3d` | Releases de producción donde importan tanto negocio como auditoría. | Combina una etiqueta humana con una referencia exacta al código. | El tag es más largo; define una convención única para todo el equipo. |

La recomendación habitual en producción es publicar cada build con un tag inmutable basado en SHA —o `release + SHA`— y desplegar Lambda usando el **digest** que devuelve ECR (`sha256:...`) cuando el proceso ya esté automatizado. El digest identifica bytes exactos de una imagen; un tag es un nombre que apunta a ella. Con ECR inmutable, ambos ayudan a evitar reemplazos accidentales, pero el digest da la referencia más precisa para una promoción entre ambientes.

La relación entre las piezas queda así:

```text
commit Git a1b2c3d
        ↓ construye CI una vez
tag de ECR create-user-git-a1b2c3d
        ↓ ECR calcula y devuelve
digest sha256:imagen-exacta
        ↓ se promueve sin recompilar
Lambda dev → Lambda staging → Lambda prod
```

El nombre de función (`create-user`) evita mezclar artefactos de las tres Lambdas dentro del mismo repositorio. `git` deja claro el tipo de referencia, y `a1b2c3d` localiza el cambio exacto. El digest es inmutable por naturaleza: si dos ambientes usan el mismo digest, usan los mismos bytes de imagen incluso si alguien añadiera otros tags al repositorio.

Un flujo de producción típico es: el desarrollador fusiona un commit; CI ejecuta pruebas; CI construye la imagen; la publica con el SHA de ese commit; se guarda el digest resultante; y el despliegue de `dev`, `staging` y `prod` promociona ese mismo digest. Así no se recompila código distinto para cada ambiente y puedes responder con evidencia a “¿qué código está ejecutando producción?”. Para volver atrás, despliegas el digest o tag inmutable de una imagen anterior que ya fue verificada; no se reconstruye a ciegas ni se usa `latest`.

No uses un tag mutable como `latest` para decidir una versión de Lambda en producción: su significado puede cambiar entre una consulta y otra, dificulta reproducir incidentes y puede hacer que dos despliegues aparentemente iguales usen artefactos distintos. Puedes mantener `latest` como comodidad de desarrollo, pero nunca como única referencia de una promoción. La mutabilidad `IMMUTABLE` que ya elegiste para ECR protege los tags de release y de SHA frente a sobrescrituras.

Cuando lleguemos a automatizar CI/CD, obtendremos el SHA desde Git y el pipeline reemplazará el marcador de forma automática. Por ahora no necesitas ejecutar ni memorizar ese mecanismo: conserva los tags `v1`, `v2`, `v3` y anota qué cambio introdujo cada versión.

No copies los valores ficticios. Reemplaza solamente `ACCOUNT_ID_REAL` y `REGION_REAL` por los valores de tu terminal.

### 4.2 Pruebas: antes de construir

Comprueba que Go esté instalado y que su versión cumple la que exige el `go.mod` de cada función:

```code
go version
```

Consulta también la versión que cada módulo declara y la imagen de Go que Docker intentará descargar:

```code
rg '^go ' ejercicios/01-usuarios/02-lambda/functions/*/go.mod
rg '^FROM .*golang:' ejercicios/01-usuarios/02-lambda/functions/*/Dockerfile
```

La versión mostrada por `go version` debe ser igual o superior a la declarada en los tres `go.mod`. La etiqueta `golang:<versión>` de los Dockerfiles debe existir en Docker Hub. Si cualquiera de las dos comprobaciones no coincide, detente antes de publicar: primero hay que alinear el proyecto a una versión de Go disponible.

Luego ejecuta las pruebas. `go -C <carpeta>` le indica a Go que ejecute desde ese módulo, sin cambiar tu terminal de carpeta:

```code
go -C ejercicios/01-usuarios/02-lambda/functions/create-user test ./...
go -C ejercicios/01-usuarios/02-lambda/functions/authenticate-user test ./...
go -C ejercicios/01-usuarios/02-lambda/functions/list-active-users test ./...
```

No continúes si una prueba falla. El Dockerfile puede compilar Go dentro de Docker, pero una compilación exitosa no reemplaza las pruebas unitarias.

### 4.3 Docker: comprobar que puede construir

```code
docker version
docker buildx version
```

`docker version` debe mostrar **Client** y **Server**: el cliente es el comando instalado en WSL y el servidor es el demonio que realmente construye imágenes. `docker buildx version` comprueba que está disponible Buildx, necesario para fijar la arquitectura objetivo. Si aparece `permission denied ... /var/run/docker.sock`, tu usuario no puede usar el demonio Docker; resuelve el servicio o los permisos antes de continuar.

### 4.4 AWS y ECR: comprobar destino y autenticar Docker

Primero renueva la sesión de AWS CLI y verifica la identidad que recibirá los permisos de ECR:

```code
aws login
```

```code
aws sts get-caller-identity
```

Después comprueba que existe el repositorio correcto en la región configurada:

```code
aws ecr describe-repositories --repository-names users-service
```

La respuesta debe contener `repositoryName: users-service`, un `repositoryUri` que comienza por tu cuenta y región, y `imageTagMutability: IMMUTABLE`. Si falla, detente: normalmente significa región equivocada, sesión equivocada o que el repositorio no existe.

Ahora autentica Docker contra **el registro**, no contra el repositorio. Reemplaza los dos marcadores con los valores que anotaste en 4.1:

```code
aws ecr get-login-password --region REGION_REAL | docker login --username AWS --password-stdin ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com
```

| Parte | Qué hace |
| --- | --- |
| `aws ecr get-login-password` | Pide a ECR un token temporal de autenticación. Tu identidad necesita `ecr:GetAuthorizationToken`. |
| `--region REGION_REAL` | Solicita el token para la región donde vive el registro. |
| `|` | Pasa el token directamente a Docker; no lo imprime ni lo deja en el historial. |
| `docker login` | Guarda temporalmente la autenticación de ese registro en la configuración local de Docker. |
| `--username AWS` | Literal obligatorio para el login de un registro privado ECR. |
| `--password-stdin` | Indica a Docker que lea el token desde la tubería. |
| `ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com` | URI del registro privado. No incluye `/users-service`, porque el login es al registro completo. |

El resultado esperado es `Login Succeeded`. Esto no vuelve público ECR ni sube imágenes: solo permite que el siguiente `docker push` pueda autenticarse.

### 4.5 Qué imagen corresponde a cada Lambda

| Lambda futura | Carpeta de contexto Docker | Tag que se publicará | Rol IAM que usará mañana |
| --- | --- | --- | --- |
| `create-user` | `ejercicios/01-usuarios/02-lambda/functions/create-user` | `create-user-v1` | `lambda-create-user-role` |
| `authenticate-user` | `ejercicios/01-usuarios/02-lambda/functions/authenticate-user` | `authenticate-user-v1` | `lambda-authenticate-user-role` |
| `list-active-users` | `ejercicios/01-usuarios/02-lambda/functions/list-active-users` | `list-active-users-v1` | `lambda-list-active-users-role` |

La URI completa de una imagen sigue siempre este patrón:

```text
ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:NOMBRE_DE_TAG
```

El texto antes de `:` identifica el repositorio; el texto después de `:` es el tag de versión. El tag no es un secreto ni un permiso: es el nombre de una versión concreta de la imagen.

### 4.6 Construir, comprobar localmente y publicar `create-user`

Este es el primer build. La última ruta es el **contexto de build**: Docker puede leer únicamente archivos dentro de esa carpeta y allí debe encontrar el `Dockerfile`, `go.mod`, código y `.dockerignore`.

```code
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v1 \
  ejercicios/01-usuarios/02-lambda/functions/create-user
```

Qué hace cada parte:

| Parámetro | Función | Por qué está aquí |
| --- | --- | --- |
| `docker buildx build` | Construye la imagen usando Buildx/BuildKit. | Permite construir para una arquitectura objetivo aunque tu máquina sea otra. |
| `--platform linux/arm64` | Produce una imagen Linux de arquitectura ARM64. | Mañana la Lambda se creará con arquitectura `arm64`; imagen y función deben coincidir. Lambda no admite una imagen multi-arquitectura como paquete de función. |
| `--provenance=false` | Desactiva la atestación de procedencia añadida por Buildx. | Lambda exige una imagen compatible de una sola arquitectura; AWS recomienda esta opción al construir imágenes Lambda con Buildx. |
| `--load` | Carga el resultado de plataforma única en el almacén local de Docker. | Sin esto, la imagen puede terminar solo en el caché de Buildx y `docker push` no la encontrará. |
| `-t <URI>:<tag>` | Asigna nombre y tag locales iguales al destino de ECR. | Así `docker push` sabe exactamente a qué repositorio y versión debe subir. |
| ruta final | Define el contexto y el Dockerfile por defecto. | Cada Lambda es un módulo Go y una imagen independientes. |

El Dockerfile es multi-etapa: primero descarga dependencias y compila el binario Go `bootstrap`; luego copia solo ese binario a la imagen final `provided.al2023` de Lambda. La imagen final no debería incluir el compilador Go, tests ni código fuente.

Como `provided.al2023` es un runtime personalizado, el Dockerfile debe terminar indicando el binario que el *entrypoint* de Lambda recibirá como handler:

```code
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /out/bootstrap ${LAMBDA_RUNTIME_DIR}/bootstrap
CMD ["bootstrap"]
```

`COPY` deja el ejecutable en el directorio del runtime; `CMD ["bootstrap"]` lo entrega como primer argumento al *entrypoint* de la imagen base. Si omites `CMD`, la imagen puede construirse y subirse correctamente, pero Lambda fallará antes de ejecutar Go con el mensaje `entrypoint requires the handler name to be the first argument`. Verifica esta condición antes de construir las tres funciones.

Antes de publicar, verifica que Docker tiene la imagen local y que su arquitectura es `arm64`:

```code
docker image inspect ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v1 --format '{{.Id}} {{.Architecture}}'
```

Debe mostrar un ID con prefijo `sha256:` y `arm64`. Luego publica la imagen:

```code
docker push ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v1
```

`docker push` compara las capas locales con ECR, sube únicamente las que ECR no tiene y publica el manifiesto de la imagen con ese tag. No crea una Lambda ni ejecuta código.

Confirma desde AWS CLI que la publicación terminó:

```code
aws ecr describe-images \
  --repository-name users-service \
  --image-ids imageTag=create-user-v1 \
  --query 'imageDetails[0].[imageTags,imageDigest,imagePushedAt,imageSizeInBytes]' \
  --output table
```

Debes ver el tag, un digest `sha256:...`, la fecha de publicación y el tamaño. Conserva ese tag: mañana formará parte de `--image-uri` de la Lambda `create-user`.

### 4.7 Publicar `authenticate-user`

Repite exactamente el mismo ciclo: build local → inspección local → push → verificación en ECR. Solo cambian carpeta y tag:

```code
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:authenticate-user-v1 \
  ejercicios/01-usuarios/02-lambda/functions/authenticate-user
```

```code
docker image inspect ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:authenticate-user-v1 --format '{{.Id}} {{.Architecture}}'
```

```code
docker push ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:authenticate-user-v1
```

```code
aws ecr describe-images \
  --repository-name users-service \
  --image-ids imageTag=authenticate-user-v1 \
  --query 'imageDetails[0].[imageTags,imageDigest,imagePushedAt,imageSizeInBytes]' \
  --output table
```

### 4.8 Publicar `list-active-users`

```code
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:list-active-users-v1 \
  ejercicios/01-usuarios/02-lambda/functions/list-active-users
```

```code
docker image inspect ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:list-active-users-v1 --format '{{.Id}} {{.Architecture}}'
```

```code
docker push ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:list-active-users-v1
```

```code
aws ecr describe-images \
  --repository-name users-service \
  --image-ids imageTag=list-active-users-v1 \
  --query 'imageDetails[0].[imageTags,imageDigest,imagePushedAt,imageSizeInBytes]' \
  --output table
```

### 4.9 Verificación final, errores frecuentes y costos

Al terminar deben existir tres tags. Lista las imágenes del repositorio:

```code
aws ecr describe-images \
  --repository-name users-service \
  --query 'imageDetails[*].[imageTags,imageDigest,imagePushedAt,imageSizeInBytes]' \
  --output table
```

| Síntoma | Causa habitual | Qué revisar |
| --- | --- | --- |
| `no such host` o endpoint incorrecto | Región errónea. | El valor de `REGION_REAL` y `aws configure get region`. |
| `no basic auth credentials` | No se hizo login de ECR o expiró. | Repite el comando de login de 4.4. |
| `denied` al hacer push | La identidad no tiene permisos de publicación. | Permisos ECR del usuario/rol que ejecuta AWS CLI, no del rol de ejecución de la Lambda. |
| `tag invalid` o `ImageTagAlreadyExistsException` | Tag mal escrito o ya publicado. | Usa el formato indicado y un commit/tag de versión nuevo. |
| `docker push` no encuentra imagen | El build falló o faltó `--load`. | Repite build y `docker image inspect`. |
| La Lambda falla después por arquitectura | Imagen y Lambda usan arquitecturas distintas. | Mantén `linux/arm64` ahora y `--architectures arm64` al crear la Lambda. |

Publicar una imagen no ejecuta una Lambda, por lo que no genera solicitudes ni duración de cómputo Lambda. Lambda cobra por solicitudes y duración cuando se invoca; la excepción es habilitar *Provisioned Concurrency*, que mantiene capacidad lista y sí cobra aunque no lleguen solicitudes. ECR sí puede cobrar almacenamiento por las imágenes que permanezcan en el repositorio, incluso sin Lambdas creadas; la política de ciclo de vida limita ese crecimiento. La transferencia entre ECR y Lambda en la misma región no tiene cargo de transferencia de datos. Consulta [precios de Lambda](https://aws.amazon.com/lambda/pricing/) y [precios de ECR](https://aws.amazon.com/ecr/pricing/) para los importes vigentes.

Referencias técnicas: [imágenes de contenedor para Lambda](https://docs.aws.amazon.com/lambda/latest/dg/images-create.html), [arquitecturas de Lambda](https://docs.aws.amazon.com/lambda/latest/dg/foundation-arch.html), [ciclo de imagen en ECR](https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html) y [referencia de Docker Buildx](https://docs.docker.com/reference/cli/docker/buildx/build/).

## 5. Crear las Lambdas desde imágenes ECR

El orden correcto ahora es crear **una Lambda, verificar que quedó activa y probarla de forma controlada**. Recién después repetiremos el ciclo para las otras dos. API Gateway viene después: una imagen en ECR no expone una URL, y una Lambda recién creada tampoco queda pública ni invocable por HTTP.

```text
Imágenes verificadas en ECR
        ↓
Lambda create-user → esperar estado Active → verificar configuración → prueba controlada
        ↓
Lambda authenticate-user → mismo ciclo
        ↓
Lambda list-active-users → mismo ciclo
        ↓
API Gateway HTTP → rutas → permisos explícitos de invocación → pruebas HTTP
```

### 5.1 Antes de crear `create-user`

Esta función usará la imagen `create-user-v1`, el rol `lambda-create-user-role` y la tabla DynamoDB `users`. Verifica primero que esos tres elementos existen y que el rol tiene las políticas esperadas:

```code
aws ecr describe-images \
  --repository-name users-service \
  --image-ids imageTag=create-user-v1 \
  --query 'imageDetails[0].[imageTags,imageDigest,imagePushedAt]' \
  --output table

aws iam get-role \
  --role-name lambda-create-user-role \
  --query 'Role.[RoleName,Arn]' \
  --output table

aws iam list-attached-role-policies \
  --role-name lambda-create-user-role \
  --query 'AttachedPolicies[*].[PolicyName,PolicyArn]' \
  --output table

aws dynamodb describe-table \
  --table-name users \
  --query 'Table.[TableName,TableStatus,TableArn]' \
  --output table
```

Debes comprobar lo siguiente antes de seguir:

- ECR muestra `create-user-v1` y un digest `sha256:...`.
- IAM muestra el ARN del rol, que tendrá el formato `arn:aws:iam::ACCOUNT_ID_REAL:role/lambda-create-user-role`.
- El rol tiene `AWSLambdaBasicExecutionRole` para escribir logs y `lambda-users-create-transaction` para la transacción de creación en DynamoDB.
- DynamoDB muestra `users` con estado `ACTIVE`.

El valor que todavía debes consultar es la región, si no la anotaste en el paso de ECR:

```code
aws configure get region
```

En los comandos siguientes reemplaza `ACCOUNT_ID_REAL` por el resultado de `aws sts get-caller-identity --query Account --output text` y `REGION_REAL` por el resultado de este último comando. No reemplaces `users`, `create-user-v1` ni los nombres de rol: son los nombres definidos para este laboratorio.

### 5.2 Crear la Lambda `users-create-user`

```code
aws lambda create-function \
  --function-name users-create-user \
  --package-type Image \
  --code ImageUri=ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v1 \
  --role arn:aws:iam::ACCOUNT_ID_REAL:role/lambda-create-user-role \
  --architectures arm64 \
  --timeout 10 \
  --memory-size 256 \
  --environment 'Variables={USERS_TABLE_NAME=users}' \
  --logging-config LogFormat=Text \
  --publish \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

`create-function` crea el recurso Lambda; no invoca el código. La primera descarga y optimización de la imagen puede tardar un momento, por eso la respuesta inicial puede mostrar estado `Pending`.

| Parámetro | Valor de este laboratorio | Qué hace y por qué |
| --- | --- | --- |
| `--function-name` | `users-create-user` | Nombre de la función dentro de la región y cuenta. El prefijo `users-` evita ambigüedad cuando existan más ejercicios. |
| `--package-type` | `Image` | Indica que el paquete es una imagen ECR, no un archivo `.zip`. Por eso no se pasan `--runtime` ni `--handler`: el Dockerfile ya incorpora el runtime y `bootstrap`. |
| `--code ImageUri=...` | URI completa de `create-user-v1` | Le dice a Lambda cuál imagen privada descargar. El prefijo contiene cuenta, registro ECR y región; `users-service` es el repositorio; lo posterior a `:` es el tag. ECR y Lambda deben estar en la misma región. |
| `--role` | ARN de `lambda-create-user-role` | Es el rol que asume el código al ejecutarse. No es el usuario ni el rol con el que tú ejecutas AWS CLI. Sus permisos mínimos ya fueron definidos en IAM. |
| `--architectures` | `arm64` | Debe coincidir con `--platform linux/arm64` usado al construir la imagen. ARM64 suele dar buena relación costo/rendimiento, pero no mezcles una imagen ARM64 con Lambda `x86_64`. |
| `--timeout` | `10` segundos | Límite máximo de una invocación. Es suficiente para este CRUD inicial; un timeout no cancela por sí solo una transacción ya aceptada por un servicio externo, por lo que la idempotencia importa en producción. |
| `--memory-size` | `256` MB | Memoria disponible; en Lambda también condiciona CPU disponible. Es un punto inicial modesto que debe medirse con métricas reales antes de ajustar. |
| `--environment` | `USERS_TABLE_NAME=users` | Entrega al código el nombre de la tabla. `create-user` exige exactamente esta variable; no usa `TABLE_NAME`. No guardes contraseñas o tokens aquí en producción: usa Secrets Manager o Parameter Store con KMS y permisos mínimos. |
| `--logging-config` | `LogFormat=Text` | Mantiene logs de texto, adecuados para empezar y compatibles con los `log` actuales de Go. En producción suele convenir JSON estructurado, retención definida y alertas sobre errores. |
| `--publish` | presente | Crea también la primera versión publicada de la función además de `$LATEST`. Una versión publicada es inmutable y permitirá usar alias como `dev` o `prod` cuando automaticemos despliegues. |
| `--tags` | etiquetas de laboratorio | Metadatos del recurso Lambda para inventario, costo y gobierno. No conceden permisos y no son tags de la imagen ECR. |

Lambda necesita recuperar la imagen privada, pero **no lo hace usando el rol de ejecución**. Ese rol solo existe después, cuando corre el handler y accede a DynamoDB y CloudWatch. Para una Lambda y un repositorio ECR en la misma cuenta, el servicio Lambda queda autorizado mediante la política basada en recursos del repositorio; al crear la función AWS puede gestionar esa política mínima si la identidad operadora tiene permiso para consultarla y actualizarla. No agregues permisos ECR amplios al rol `lambda-create-user-role` por costumbre.

### 5.3 Esperar, comprobar y entender el resultado

No intentes invocarla hasta que esté activa:

```code
aws lambda wait function-active-v2 \
  --function-name users-create-user
```

El *waiter* consulta Lambda periódicamente y termina sin salida cuando el estado pasa a `Active`; si termina con error, no asumas que la función quedó bien. Consulta el detalle:

```code
aws lambda get-function-configuration \
  --function-name users-create-user \
  --query '[FunctionName,State,LastUpdateStatus,PackageType,Architectures,MemorySize,Timeout,Role,Environment.Variables,Version]' \
  --output table
```

La salida debe mostrar `users-create-user`, `Active`, `Image`, `arm64`, `256`, `10`, el ARN del rol correcto, `USERS_TABLE_NAME` con valor `users` y versión `1`. La primera versión publicada es `1`; `$LATEST` sigue existiendo como copia editable de trabajo, pero no debes usarla como referencia de producción.

Configura ahora la retención del grupo de CloudWatch Logs. La retención **no** se define con `create-function`: CloudWatch Logs la administra por separado. Para no depender de que la primera invocación cree el grupo automáticamente, créalo tú antes de invocar la función:

```code
aws logs create-log-group \
  --log-group-name /aws/lambda/users-create-user \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

El nombre `/aws/lambda/NOMBRE_DE_LAMBDA` es la convención que Lambda usa para su grupo por defecto; en este caso `NOMBRE_DE_LAMBDA` es `users-create-user`. Los tags de Lambda no se copian de forma automática al grupo de logs, por eso se declaran otra vez. Si el comando responde `ResourceAlreadyExistsException`, el grupo ya fue creado por una invocación anterior: no es un problema y debes continuar con el siguiente comando.

```code
aws logs put-retention-policy \
  --log-group-name /aws/lambda/users-create-user \
  --retention-in-days 3
```

`--retention-in-days 3` conserva cada evento durante tres días y luego CloudWatch Logs lo elimina. Es adecuado para estos labs: deja tiempo para investigar un error sin mantener datos de prueba ni acumular costo indefinidamente. Repite ambos pasos para cada Lambda, sustituyendo únicamente el nombre de la función. Si el grupo ya existe, solo necesitas ejecutar `put-retention-policy`.

En producción no hay un número universal. Se elige la retención con requisitos de auditoría, privacidad, capacidad de diagnóstico y costo: por ejemplo, 30 días para desarrollo, 90 días para operación habitual y más tiempo solo cuando cumplimiento lo exija. Es preferible crear el grupo y su retención desde IaC antes del despliegue para impedir que una función nueva quede accidentalmente con retención infinita.

Comprueba el valor configurado:

```code
aws logs describe-log-groups \
  --log-group-name-prefix /aws/lambda/users-create-user \
  --query 'logGroups[*].[logGroupName,retentionInDays]' \
  --output table
```

Debe aparecer `/aws/lambda/users-create-user` y `3` en la columna de retención. Si aparece vacío, todavía no hay política de retención y CloudWatch Logs conservaría los eventos indefinidamente.

### 5.4 Prueba directa opcional: tiene efecto en DynamoDB

La función está preparada para recibir un evento de API Gateway HTTP API versión 2. Ya existe un archivo de evento de ejemplo en el proyecto. **Invocarlo crea un usuario en DynamoDB**, así que hazlo una sola vez y no lo repitas con el mismo correo: la operación debe responder conflicto por correo duplicado.

Este comando asume que estás ubicado en `ejercicios/01-usuarios`, tal como los pasos anteriores. En WSL usa rutas Linux: la carpeta de Windows `C:\\Users\\USUARIO\\...` corresponde a `/mnt/c/Users/USUARIO/...`, no a `/c/Users/...`. La ruta relativa evita tener que escribir la ruta absoluta.

```code
aws lambda invoke \
  --function-name users-create-user \
  --cli-binary-format raw-in-base64-out \
  --payload fileb://02-lambda/functions/create-user/internal/handler/events/request.json \
  create-user-response.json
```

- `--function-name`: destino de la invocación directa. Esto no da acceso público a la función.
- `--cli-binary-format raw-in-base64-out`: hace que AWS CLI v2 envíe el JSON del archivo como bytes sin pedir que lo codifiques en Base64.
- `--payload fileb://02-lambda/...`: archivo de evento HTTP API v2, relativo al directorio actual. El prefijo `fileb://` lo trata como contenido binario; el handler recibirá el cuerpo y ruta de ejemplo. Si prefieres una ruta absoluta en WSL, usa `fileb:///mnt/c/Users/USUARIO/GolandProjects/serverless_dynamo_lab/ejercicios/01-usuarios/02-lambda/...`; nunca `fileb:///c/...`.
- `create-user-response.json`: archivo local de salida que AWS CLI creará o reemplazará en tu directorio actual con la respuesta del handler. No es un archivo en AWS.

Después, consulta la respuesta y el log más reciente:

```code
cat create-user-response.json

aws logs tail /aws/lambda/users-create-user \
  --since 10m
```

Una respuesta correcta contiene un `statusCode` `201`. Si recibes `409`, el correo del evento ya fue creado: no es un fallo de infraestructura, sino la protección de unicidad esperada. Si recibes `500`, consulta los logs antes de cambiar permisos o recrear recursos.

### Resultado final esperado: Lambda `users-create-user`

Da por terminada esta Lambda solo después de comprobar ambos escenarios con el mismo evento de ejemplo:

| Prueba | Resultado en `create-user-response.json` | Qué demuestra |
| --- | --- | --- |
| Primera invocación | `statusCode` `201` | La imagen inicia, la variable de entorno se lee, el rol puede ejecutar la transacción y DynamoDB creó el usuario. |
| Segunda invocación con el mismo email | `statusCode` `409` y mensaje `user already exists` | La condición transaccional reservó el correo y evita que se creen dos usuarios con la misma dirección. |

El campo `StatusCode: 200` que imprime AWS CLI pertenece al servicio Lambda: confirma que la invocación llegó y produjo una respuesta. El `statusCode` dentro de `create-user-response.json` es la respuesta HTTP de tu handler, que API Gateway devolverá al cliente cuando conectemos la ruta. No confundas ambos valores.

El log de éxito debe incluir `dynamodb TransactWriteItems completed: table=users result=created`; al repetir el correo debe indicar el resultado de duplicado. Conserva esta prueba como evidencia manual del contrato de la Lambda antes de continuar con `authenticate-user`.

### 5.5 Errores frecuentes al crear la primera Lambda

| Síntoma | Causa probable | Acción concreta |
| --- | --- | --- |
| `ResourceConflictException` | Ya existe `users-create-user`. | No repitas `create-function`; consulta su configuración con `get-function-configuration`. Para actualizar una imagen en el futuro se usa `update-function-code`. |
| `InvalidParameterValueException` relacionado con imagen | URI/tag inexistente, región distinta o arquitectura incompatible. | Revisa `describe-images`, el URI y que build/Lambda sean `arm64`. |
| Error de permisos ECR al crear | La identidad de AWS CLI no puede leer/gestionar la política del repositorio. | Revisa los permisos del usuario o rol de despliegue; no amplíes a ciegas el rol de ejecución de Lambda. |
| `AccessDenied` al invocar DynamoDB | Rol incorrecto o política `lambda-users-create-transaction` no adjunta. | Ejecuta `list-attached-role-policies` y revisa los logs. |
| `AccessDeniedException` para `dynamodb:PutItem` durante `TransactWriteItems` | La política permite la operación de transacción, pero no las escrituras `Put` contenidas en ella. | Añade `dynamodb:PutItem` a `lambda-users-create-transaction`, publica una nueva versión predeterminada de la política y vuelve a invocar; no reconstruyas la imagen. |
| Error `USERS_TABLE_NAME is required` | Se usó `TABLE_NAME` o falta la variable. | Comprueba `Environment.Variables` con `get-function-configuration`. |
| La invocación devuelve `FunctionError` | El handler falló aunque Lambda exista. | Lee `create-user-response.json` y `aws logs tail`; no recrees la función sin diagnosticar. |
| `Runtime.ExitError` y logs con `entrypoint requires the handler name to be the first argument` | La imagen basada en `provided.al2023` no tiene `CMD ["bootstrap"]`, por lo que el runtime falla durante inicialización. El payload aún no se procesa. | Corrige el Dockerfile, publica un tag nuevo y actualiza el código de la Lambda; sigue el procedimiento inmediato siguiente. |

### 5.6 Recuperación: la imagen inicia sin handler

Este caso ocurrió al invocar `users-create-user`: el error está antes de `lambda.Start(...)`. Por tanto no cambies el body, DynamoDB ni las políticas IAM todavía. Primero abre `02-lambda/functions/create-user/Dockerfile` y agrega al final `CMD ["bootstrap"]`, debajo de `COPY --from=build ...`.

ECR se creó con tags inmutables. No intentes volver a publicar `create-user-v1`: usa una versión nueva. Desde `ejercicios/01-usuarios`, reconstruye, publica y despliega `create-user-v2`:

```code
docker buildx build --platform linux/arm64 --provenance=false --load \
  -t ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v2 \
  02-lambda/functions/create-user

docker push ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v2

aws lambda update-function-code \
  --function-name users-create-user \
  --image-uri ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:create-user-v2 \
  --publish

aws lambda wait function-updated-v2 \
  --function-name users-create-user
```

`update-function-code` no crea una segunda Lambda: cambia la imagen de la función existente. `--publish` genera una nueva versión inmutable; el *waiter* evita invocarla mientras Lambda continúa optimizando la imagen. Luego repite **exactamente** la invocación de la sección 5.4. Si ahora aparece un error sobre el body, variables o DynamoDB, ese será el siguiente nivel de diagnóstico: antes no podía existir porque el proceso Go no arrancaba.

### 5.7 Recuperación: transacción sin permiso `PutItem`

Una llamada `TransactWriteItems` no elimina la necesidad de autorizar las operaciones que contiene. `create-user` usa dos `Put` dentro de la transacción: una reserva el correo para garantizar unicidad y otra crea el perfil. Por eso `policy_create_user_dynamo.json` incluye tanto `dynamodb:TransactWriteItems` como `dynamodb:PutItem`, limitados a la tabla `users`.

Si ya creaste la política administrada, editar el archivo local no cambia IAM. Desde `ejercicios/01-usuarios`, reemplaza los marcadores del archivo y crea una versión nueva predeterminada. Consulta primero tu cuenta y región; copia los dos valores reales en el comando `sed`:

```code
aws sts get-caller-identity --query Account --output text

aws configure get region
```

```code
sed -e 's/<AWS_REGION>/REGION_REAL/g' -e 's/<AWS_ACCOUNT_ID>/ACCOUNT_ID_REAL/g' documentacion/policy/policy_create_user_dynamo.json > /tmp/policy_create_user_dynamo.json
```

```code
aws iam create-policy-version \
  --policy-arn arn:aws:iam::ACCOUNT_ID_REAL:policy/lambda-users-create-transaction \
  --policy-document file:///tmp/policy_create_user_dynamo.json \
  --set-as-default
```

`--set-as-default` hace que el rol adjunto use la versión nueva sin volver a asociar la política ni actualizar la Lambda. Después repite la invocación de la sección 5.4. Si IAM informa que ya existen cinco versiones de la política, lista las versiones, confirma cuál no es la predeterminada y elimina únicamente una versión antigua antes de crear la nueva; nunca elimines la versión marcada como predeterminada.

### 5.8 Crear y validar `users-authenticate-user`

Esta Lambda lee el registro de unicidad por correo con `GetItem`. Usa la imagen `authenticate-user-v2`, el rol `lambda-authenticate-user-role` y la variable de entorno `TABLE_NAME=users`. No reutilices `USERS_TABLE_NAME`: ese nombre solo pertenece a `create-user`.

```code
aws lambda create-function \
  --function-name users-authenticate-user \
  --package-type Image \
  --code ImageUri=ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:authenticate-user-v2 \
  --role arn:aws:iam::ACCOUNT_ID_REAL:role/lambda-authenticate-user-role \
  --architectures arm64 \
  --timeout 10 \
  --memory-size 256 \
  --environment 'Variables={TABLE_NAME=users}' \
  --logging-config LogFormat=Text \
  --publish \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

`--code` identifica la imagen privada ya publicada; `--role` limita el handler a la política `lambda-users-read-getitem`; `--environment` entrega el nombre físico de tabla sin fijarlo en código. El resto conserva las decisiones de la primera Lambda: imagen ARM64, diez segundos, 256 MB y versión publicada. Crear la función no ejecuta un login ni expone una URL pública.

```code
aws lambda wait function-active-v2 \
  --function-name users-authenticate-user

aws logs create-log-group \
  --log-group-name /aws/lambda/users-authenticate-user \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli

aws logs put-retention-policy \
  --log-group-name /aws/lambda/users-authenticate-user \
  --retention-in-days 3
```

Si `create-log-group` informa que el grupo ya existe, continúa con la retención. Comprueba la función y la retención antes de invocar:

```code
aws lambda get-function-configuration \
  --function-name users-authenticate-user \
  --query '[FunctionName,State,LastUpdateStatus,Architectures,MemorySize,Timeout,Role,Environment.Variables,Version]' \
  --output table

aws logs describe-log-groups \
  --log-group-name-prefix /aws/lambda/users-authenticate-user \
  --query 'logGroups[*].[logGroupName,retentionInDays]' \
  --output table
```

Desde `ejercicios/01-usuarios`, realiza la prueba directa. No crea ni modifica usuarios:

```code
aws lambda invoke \
  --function-name users-authenticate-user \
  --cli-binary-format raw-in-base64-out \
  --payload fileb://02-lambda/functions/authenticate-user/internal/handler/events/request.json \
  authenticate-user-response.json

cat authenticate-user-response.json

aws logs tail /aws/lambda/users-authenticate-user \
  --since 10m
```

Para un registro existente y credenciales que coincidan, el body debe contener `statusCode` `200` y el identificador de usuario. Con credenciales inválidas, la implementación actual responde `404`; el contrato OpenAPI plantea `401`, por lo que esa diferencia es una mejora pendiente del handler, no un error de Lambda, IAM o DynamoDB. Un log `dynamodb GetItem completed: table=users result=found` demuestra la lectura; `result=not_found` demuestra que la consulta se ejecutó pero no encontró el correo.

**Hallazgo de la validación actual:** si el usuario fue creado por la versión inicial de `create-user`, el adaptador de creación guarda el valor literal `PASSWORD` en DynamoDB en vez de la contraseña recibida en el request. En consecuencia, el evento de ejemplo de autenticación puede mostrar `result=found` en logs y aun así responder `404` por credenciales distintas. Esto confirma que la infraestructura de lectura funciona y revela un defecto de aplicación que debe corregirse antes de declarar el login funcional. La corrección no consiste en ampliar IAM: debe persistirse la contraseña recibida —en un sistema real, un hash seguro, nunca texto plano— y crear un usuario de prueba nuevo con la imagen corregida.

### 5.9 Crear y validar `users-list-active-users`

Esta Lambda consulta el GSI `GSI1` mediante el rol `lambda-list-active-users-role`. Por eso `Query` debe autorizarse sobre el ARN del índice, no solo sobre la tabla. Usa la imagen `list-active-users-v2` y también requiere `TABLE_NAME=users`.

```code
aws lambda create-function \
  --function-name users-list-active-users \
  --package-type Image \
  --code ImageUri=ACCOUNT_ID_REAL.dkr.ecr.REGION_REAL.amazonaws.com/users-service:list-active-users-v2 \
  --role arn:aws:iam::ACCOUNT_ID_REAL:role/lambda-list-active-users-role \
  --architectures arm64 \
  --timeout 10 \
  --memory-size 256 \
  --environment 'Variables={TABLE_NAME=users}' \
  --logging-config LogFormat=Text \
  --publish \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

```code
aws lambda wait function-active-v2 \
  --function-name users-list-active-users

aws logs create-log-group \
  --log-group-name /aws/lambda/users-list-active-users \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli

aws logs put-retention-policy \
  --log-group-name /aws/lambda/users-list-active-users \
  --retention-in-days 3
```

```code
aws lambda get-function-configuration \
  --function-name users-list-active-users \
  --query '[FunctionName,State,LastUpdateStatus,Architectures,MemorySize,Timeout,Role,Environment.Variables,Version]' \
  --output table

aws lambda invoke \
  --function-name users-list-active-users \
  --cli-binary-format raw-in-base64-out \
  --payload fileb://02-lambda/functions/list-active-users/internal/handler/events/request.json \
  list-active-users-response.json

cat list-active-users-response.json

aws logs tail /aws/lambda/users-list-active-users \
  --since 10m
```

La respuesta de negocio debe tener `statusCode` `200` y un cuerpo con `items`; como `create-user` ya creó un usuario `ACTIVE`, la lista debe contener al menos uno. El handler acepta `limit` entre 1 y 100 y devuelve un `nextCursor` opaco si existe otra página. El log esperado es `dynamodb Query completed: table=users index=GSI1`; si aparece `AccessDenied` para `Query`, revisa que la política se refiera a `table/users/index/GSI1`.

Resultado final de las dos funciones: `authenticate-user` confirma el patrón de lectura directa por clave (`GetItem`) y `list-active-users` confirma el patrón de lectura por índice ordenado (`Query` sobre GSI). Ambas deben mostrar su grupo de logs con retención de 3 días antes de empezar la fase de API Gateway.

## 6. Exponer las Lambdas mediante API Gateway HTTP

Una HTTP API es la capa pública del laboratorio: recibe HTTP, genera un evento *payload v2.0* y lo entrega a la Lambda integrada. Las Lambdas no se vuelven públicas por crear esta API; cada una recibe un permiso explícito y restringido a su ruta. Todos los comandos de esta sección usan `ACCOUNT_ID_REAL` y `REGION_REAL` como marcadores. No los sustituyas en el repositorio: consúltalos con AWS CLI y úsalos solo al ejecutar.

### 6.1 Crear la API y entender el stage `$default`

```code
aws apigatewayv2 create-api \
  --name users-http-api \
  --protocol-type HTTP \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli
```

`--protocol-type HTTP` selecciona HTTP API, más simple y económica que REST API para este caso. La respuesta contiene `ApiId` y `ApiEndpoint`; anótalos como `API_ID_REAL` y `API_ENDPOINT_REAL`. La API no trae un stage utilizable hasta que lo crees: lo haremos en 6.4 como `$default` con *auto deploy*, de modo que cada ruta nueva quede publicada sin ejecutar `create-deployment`. Es cómodo para el lab; en producción suele usarse un stage nombrado, despliegues controlados y dominios personalizados.

### 6.2 Crear las integraciones Lambda

Una integración especifica qué Lambda recibe una ruta. `AWS_PROXY` entrega a Go el evento HTTP API v2 prácticamente sin transformaciones. Crea una por función; copia el `IntegrationId` de cada respuesta como `INTEGRATION_ID_*`.

```code
aws apigatewayv2 create-integration \
  --api-id API_ID_REAL \
  --integration-type AWS_PROXY \
  --integration-uri arn:aws:apigateway:REGION_REAL:lambda:path/2015-03-31/functions/arn:aws:lambda:REGION_REAL:ACCOUNT_ID_REAL:function:users-create-user/invocations \
  --payload-format-version 2.0 \
  --timeout-in-millis 10000

aws apigatewayv2 create-integration \
  --api-id API_ID_REAL \
  --integration-type AWS_PROXY \
  --integration-uri arn:aws:apigateway:REGION_REAL:lambda:path/2015-03-31/functions/arn:aws:lambda:REGION_REAL:ACCOUNT_ID_REAL:function:users-authenticate-user/invocations \
  --payload-format-version 2.0 \
  --timeout-in-millis 10000

aws apigatewayv2 create-integration \
  --api-id API_ID_REAL \
  --integration-type AWS_PROXY \
  --integration-uri arn:aws:apigateway:REGION_REAL:lambda:path/2015-03-31/functions/arn:aws:lambda:REGION_REAL:ACCOUNT_ID_REAL:function:users-list-active-users/invocations \
  --payload-format-version 2.0 \
  --timeout-in-millis 10000
```

La URI no es la URL pública de Lambda: es el formato interno que API Gateway exige para invocarla. `--payload-format-version 2.0` debe coincidir con `events.APIGatewayV2HTTPRequest` de los handlers. Los 10 000 ms se alinean con el timeout de Lambda; API Gateway puede agotar su propio timeout antes que la función, por lo que en producción ambos valores se diseñan juntos.

### 6.3 Crear rutas y permisos mínimos de invocación

Primero vincula método, path e integración. El `--target` siempre tiene formato `integrations/INTEGRATION_ID_REAL`.

```code
aws apigatewayv2 create-route --api-id API_ID_REAL --route-key 'POST /users' --target integrations/INTEGRATION_ID_CREATE_REAL
aws apigatewayv2 create-route --api-id API_ID_REAL --route-key 'POST /auth/login' --target integrations/INTEGRATION_ID_AUTH_REAL
aws apigatewayv2 create-route --api-id API_ID_REAL --route-key 'GET /users/active' --target integrations/INTEGRATION_ID_LIST_REAL
```

Después permite que **solo esta API, método y ruta** invoquen cada función:

```code
aws lambda add-permission \
  --function-name users-create-user \
  --statement-id allow-apigateway-create-user \
  --action lambda:InvokeFunction \
  --principal apigateway.amazonaws.com \
  --source-arn arn:aws:execute-api:REGION_REAL:ACCOUNT_ID_REAL:API_ID_REAL/*/POST/users

aws lambda add-permission \
  --function-name users-authenticate-user \
  --statement-id allow-apigateway-authenticate-user \
  --action lambda:InvokeFunction \
  --principal apigateway.amazonaws.com \
  --source-arn arn:aws:execute-api:REGION_REAL:ACCOUNT_ID_REAL:API_ID_REAL/*/POST/auth/login

aws lambda add-permission \
  --function-name users-list-active-users \
  --statement-id allow-apigateway-list-active-users \
  --action lambda:InvokeFunction \
  --principal apigateway.amazonaws.com \
  --source-arn arn:aws:execute-api:REGION_REAL:ACCOUNT_ID_REAL:API_ID_REAL/*/GET/users/active
```

`add-permission` modifica la política basada en recursos de Lambda, distinta del rol de ejecución IAM. El rol permite que el código acceda a DynamoDB; este permiso permite que API Gateway active la función. `--source-arn` evita conceder invocación a cualquier API Gateway de la cuenta. Si repites un comando con el mismo `--statement-id`, Lambda responde conflicto: consulta la política antes de crear otro identificador.

### 6.4 Logs de acceso y comprobación

Los logs de Lambda ya tienen tres días de retención. Para tener también visibilidad de HTTP, crea un grupo de acceso de API Gateway y configura el stage `$default`; las comillas simples evitan que Bash interprete el carácter `$`.

```code
aws logs create-log-group \
  --log-group-name /aws/apigateway/users-http-api \
  --tags environment=dev,system=users,owner=backend,managed-by=manual-cli

aws logs put-retention-policy \
  --log-group-name /aws/apigateway/users-http-api \
  --retention-in-days 3

aws apigatewayv2 create-stage \
  --api-id API_ID_REAL \
  --stage-name '$default' \
  --auto-deploy \
  --access-log-settings '{"DestinationArn":"arn:aws:logs:REGION_REAL:ACCOUNT_ID_REAL:log-group:/aws/apigateway/users-http-api","Format":"{\\"requestId\\":\\"$context.requestId\\",\\"status\\":\\"$context.status\\",\\"routeKey\\":\\"$context.routeKey\\",\\"integrationError\\":\\"$context.integrationErrorMessage\\"}"}'
```

En producción restringe CORS si lo habilitas, usa autenticación/autorización antes de exponer rutas sensibles, y define alarmas sobre 5xx y latencia. Para este lab no habilitamos CORS ni autenticación: la URL ejecuta una API pública deliberadamente y temporal.

```code
aws apigatewayv2 get-routes --api-id API_ID_REAL --query 'Items[*].[RouteKey,Target]' --output table
aws apigatewayv2 get-integrations --api-id API_ID_REAL --query 'Items[*].[IntegrationId,IntegrationType,PayloadFormatVersion,IntegrationUri]' --output table
aws lambda get-policy --function-name users-create-user --output json
```

Debes ver las tres rutas, tres integraciones `AWS_PROXY` con payload `2.0` y un statement de API Gateway en la política de cada Lambda.

### 6.5 Prueba HTTP y colección Postman

Antes de probar o importar Postman, consulta la URL exacta de la HTTP API v2. `API_ID_REAL` es el campo `ApiId` que devolvió `aws apigatewayv2 create-api`; si no lo anotaste, obténlo con `aws apigatewayv2 get-apis --query 'Items[*].[Name,ApiId]' --output table` y localiza `users-http-api`.

```code
aws apigatewayv2 get-api \
  --api-id API_ID_REAL \
  --query 'ApiEndpoint' \
  --output text
```

La salida tiene una forma como `https://IDENTIFICADOR.execute-api.REGION_REAL.amazonaws.com`. Guárdala como `API_ENDPOINT_REAL` para los comandos de esta sección y como valor de la variable `baseUrl` en Postman. No es necesario agregar `/$default`: ese stage se sirve directamente desde la URL base. Si en producción usas un stage con nombre, por ejemplo `dev`, la URL sí incluiría `/dev` o se resolvería mediante un dominio personalizado.

Para comprobar una ruta que no modifica datos, lista activos:

```code
curl -i 'API_ENDPOINT_REAL/users/active?limit=10'
```

Para las rutas POST usa un body JSON normal, no el archivo de evento Lambda: API Gateway construye ese evento por ti. Repetir un correo existente debe devolver `409`; autenticación puede devolver `404` por la regla simplificada descrita en 5.8.

```code
curl -i -X POST API_ENDPOINT_REAL/users \
  -H 'Content-Type: application/json' \
  --data '{"email":"EMAIL_EXISTENTE_REAL","password":"PASSWORD_DE_PRUEBA","name":"Nombre","lastName":"Apellido","dateOfBirth":"2000-01-02"}'

curl -i -X POST API_ENDPOINT_REAL/auth/login \
  -H 'Content-Type: application/json' \
  --data '{"email":"EMAIL_EXISTENTE_REAL","password":"PASSWORD_DE_PRUEBA"}'
```

La colección importable está en `postman/usuarios.postman_collection.json`. Al importarla, abre la colección, entra en **Variables**, ubica `baseUrl` y pega exactamente el valor obtenido con `get-api`; no añadas una barra final. No contiene cuentas, ARNs ni endpoint real. Sus requests cubren crear usuario, repetirlo, autenticar y listar activos.

La variable de ejemplo `demoPassword=PASSWORD` existe para reproducir el comportamiento simplificado actual del ejercicio: `create-user` guarda ese valor fijo y `authenticate-user` lo compara. Por eso el request de Postman puede funcionar aunque el contrato OpenAPI declare una contraseña de al menos 12 caracteres. Es una deuda didáctica deliberada, no un patrón de seguridad: antes de producción hay que almacenar un hash de la contraseña, validar el mínimo real y alinear handler, colección y OpenAPI.

### Resultado final del laboratorio

El laboratorio queda completo cuando `GET /users/active` responde `200`, los POST llegan a sus handlers (resultado de negocio `201`/`409` y `200`/`404` según datos), los logs de API Gateway y Lambda tienen retención de tres días, y la colección Postman reproduce las mismas rutas. Al terminar los dos días, elimina API Gateway, las Lambdas, la tabla y las imágenes si no continuarás el lab, para evitar costos residuales.

Fuentes: [AWS CLI `create-function`](https://docs.aws.amazon.com/cli/latest/reference/lambda/create-function.html), [crear Lambda desde una imagen de contenedor](https://docs.aws.amazon.com/lambda/latest/dg/images-create.html) y [AWS CLI `apigatewayv2 create-api`](https://docs.aws.amazon.com/cli/latest/reference/apigatewayv2/create-api.html).

## 7. Inventario y limpieza del laboratorio manual

La limpieza es destructiva: elimina datos, imágenes y configuraciones. Hazla solo al terminar el lab o cuando hayas confirmado que no necesitas conservar nada. Primero inventaría y anota los identificadores reales; después elimina en el orden de dependencias. No borres recursos de otro ejercicio solo porque tengan un nombre parecido.

### 7.1 Inventario antes de borrar

Ejecuta estas consultas de solo lectura. Sustituye `API_ID_REAL` por el `ApiId` anotado al crear la HTTP API. Los nombres de los tres roles, Lambdas, tabla y repositorio sí son los que creó este recorrido manual.

```code
aws apigatewayv2 get-apis \
  --query 'Items[*].[Name,ApiId,ApiEndpoint]' \
  --output table

aws apigatewayv2 get-routes \
  --api-id API_ID_REAL \
  --query 'Items[*].[RouteKey,Target]' \
  --output table

aws lambda list-functions \
  --query 'Functions[?starts_with(FunctionName, `users-`)].[FunctionName,PackageType,LastModified]' \
  --output table

aws dynamodb list-tables \
  --query 'TableNames' \
  --output table

aws ecr describe-repositories \
  --repository-names users-service \
  --query 'repositories[0].[repositoryName,repositoryUri,imageTagMutability]' \
  --output table

aws logs describe-log-groups \
  --log-group-name-prefix /aws/ \
  --query 'logGroups[?retentionInDays==`3`].[logGroupName,retentionInDays]' \
  --output table

aws iam list-roles \
  --query 'Roles[?starts_with(RoleName, `lambda-`)].[RoleName,Arn]' \
  --output table

aws iam list-policies \
  --scope Local \
  --query 'Policies[?starts_with(PolicyName, `lambda-users-`)].[PolicyName,Arn,AttachmentCount]' \
  --output table
```

Revisa especialmente `AttachmentCount`: una política administrada por el cliente no se puede borrar mientras siga asociada a un rol. La búsqueda por prefijos es una ayuda, no una autorización para borrar en bloque; comprueba cada fila contra el inventario esperado.

### 7.2 Orden de eliminación y motivo

| Orden | Recursos manuales | Motivo |
| --- | --- | --- |
| 1 | HTTP API `users-http-api` | Elimina rutas, integraciones y stage que apuntan a Lambda. |
| 2 | Grupo de logs de API Gateway | API Gateway no lo elimina automáticamente. |
| 3 | Las tres Lambdas | Quita funciones y sus políticas basadas en recursos de invocación. |
| 4 | Grupos de logs de Lambda | Lambda no garantiza borrar los logs al eliminar una función. |
| 5 | Roles IAM | Ya no deben estar en uso por Lambda. |
| 6 | Políticas IAM administradas por el cliente | Primero se desasocian de los roles; luego se eliminan. |
| 7 | Imágenes y repositorio ECR | La Lambda ya no depende de las imágenes. |
| 8 | Tabla DynamoDB `users` | Es el último recurso con datos y posible costo de almacenamiento. |

### 7.3 Eliminar API y logs de API Gateway

El primer comando borra el API, incluidas sus rutas, integraciones y stage. No borra el grupo de logs que creaste por separado.

```code
aws apigatewayv2 delete-api \
  --api-id API_ID_REAL

aws logs delete-log-group \
  --log-group-name /aws/apigateway/users-http-api
```

`delete-api` no requiere borrar rutas una por una. Si recibes `NotFoundException`, consulta el inventario: puede que la API ya se hubiera eliminado. En producción no elimines una API pública sin retirar antes DNS, consumidores y alarmas de forma planificada.

### 7.4 Eliminar Lambdas y sus logs

Las políticas de invocación de API Gateway forman parte de cada Lambda; `delete-function` las elimina junto con la función. Las tres funciones usan imágenes, pero eliminar la función no borra esas imágenes de ECR.

```code
aws lambda delete-function --function-name users-create-user
aws lambda delete-function --function-name users-authenticate-user
aws lambda delete-function --function-name users-list-active-users

aws logs delete-log-group --log-group-name /aws/lambda/users-create-user
aws logs delete-log-group --log-group-name /aws/lambda/users-authenticate-user
aws logs delete-log-group --log-group-name /aws/lambda/users-list-active-users
```

Verifica que ya no existan antes de tocar roles IAM:

```code
aws lambda get-function --function-name users-create-user
```

El resultado esperado es un error `ResourceNotFoundException`. Repite la misma consulta, cambiando el nombre, para las otras dos Lambdas si quieres una verificación individual.

### 7.5 Desasociar y borrar IAM

Primero quita la política AWS administrada de logs y la política DynamoDB propia de cada rol. Después borra el rol. Las políticas DynamoDB se eliminan al final, cuando `AttachmentCount` sea cero.

```code
aws iam detach-role-policy \
  --role-name lambda-create-user-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam detach-role-policy \
  --role-name lambda-create-user-role \
  --policy-arn POLICY_ARN_CREATE_REAL

aws iam detach-role-policy \
  --role-name lambda-authenticate-user-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam detach-role-policy \
  --role-name lambda-authenticate-user-role \
  --policy-arn POLICY_ARN_AUTHENTICATE_REAL

aws iam detach-role-policy \
  --role-name lambda-list-active-users-role \
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

aws iam detach-role-policy \
  --role-name lambda-list-active-users-role \
  --policy-arn POLICY_ARN_LIST_REAL

aws iam delete-role --role-name lambda-create-user-role
aws iam delete-role --role-name lambda-authenticate-user-role
aws iam delete-role --role-name lambda-list-active-users-role
```

`POLICY_ARN_*_REAL` se obtiene de `aws iam list-policies --scope Local` en el inventario; no copies un ARN de otra cuenta. `AWSLambdaBasicExecutionRole` es una política administrada por AWS: la desasocias, pero **no** la eliminas porque pertenece a AWS y puede ser utilizada por otros roles.

Ahora borra solo las tres políticas administradas por el cliente del ejercicio:

```code
aws iam delete-policy --policy-arn POLICY_ARN_CREATE_REAL
aws iam delete-policy --policy-arn POLICY_ARN_AUTHENTICATE_REAL
aws iam delete-policy --policy-arn POLICY_ARN_LIST_REAL
```

Si `delete-policy` indica que existen varias versiones, primero consulta cuál es la predeterminada y elimina únicamente las versiones que no lo sean. Nunca intentes borrar la versión predeterminada.

```code
aws iam list-policy-versions --policy-arn POLICY_ARN_CREATE_REAL
aws iam delete-policy-version --policy-arn POLICY_ARN_CREATE_REAL --version-id VERSION_ID_NO_PREDETERMINADA_REAL
```

Repite esas dos líneas por cada política que tenga versiones no predeterminadas, y vuelve a ejecutar `delete-policy`. Esto evita acumular versiones y respeta el límite de cinco versiones por política administrada.

### 7.6 Vaciar ECR y borrar el repositorio

Un repositorio ECR debe estar vacío para poder eliminarse. Primero lista sus imágenes para revisar lo que desaparecerá:

```code
aws ecr list-images \
  --repository-name users-service \
  --query 'imageIds[*]' \
  --output json
```

Para cada objeto `imageTag` o `imageDigest` que devuelva la consulta, ejecuta un borrado explícito. Ejemplo con un tag de laboratorio:

```code
aws ecr batch-delete-image \
  --repository-name users-service \
  --image-ids imageTag=CREATE_USER_TAG_REAL
```

Repite el comando para las imágenes restantes; usar una a una hace visible qué vas a destruir. Cuando `list-images` devuelva una lista vacía, elimina el repositorio:

```code
aws ecr delete-repository \
  --repository-name users-service
```

No uses `--force` en el lab como sustituto de revisar el inventario. En producción una política de ciclo de vida suele retirar imágenes antiguas, pero antes de borrar un repositorio se confirma que ningún despliegue, rollback o función lo necesita.

### 7.7 Borrar la tabla y comprobar la cuenta

Este es el paso que elimina todos los usuarios de prueba y el índice `GSI1` con ellos:

```code
aws dynamodb delete-table \
  --table-name users

aws dynamodb wait table-not-exists \
  --table-name users
```

`delete-table` inicia el borrado; `wait table-not-exists` espera hasta que DynamoDB confirme que terminó. Para verificar el inventario final de este recorrido manual:

```code
aws dynamodb describe-table --table-name users
aws ecr describe-repositories --repository-names users-service
aws iam get-role --role-name lambda-create-user-role
aws lambda get-function --function-name users-create-user
```

Cada consulta debe responder `ResourceNotFoundException` o el error equivalente de recurso inexistente. Cambia el nombre para comprobar las otras dos funciones y roles. Si también desplegaste la alternativa CDK, destrúyela desde [su guía](../04-cdk/README.md#9-inventario-y-limpieza-del-laboratorio-cdk): sus recursos y el bootstrap de CDK son independientes de esta limpieza manual.

## Cuando migres de CLI a CDK

Los comandos manuales sirven para aprender el recurso. CDK los expresará de forma declarativa y repetible; no deben convertirse en dos fuentes de verdad para producción.

| AWS CLI manual | Equivalente conceptual en CDK |
| --- | --- |
| `aws dynamodb create-table` | `dynamodb.Table` con claves, facturación y GSI |
| `aws ecr create-repository` | `ecr.Repository` con mutabilidad, cifrado y tags |
| `aws ecr put-lifecycle-policy` | `lifecycleRules` del repositorio |
| `aws lambda create-function` | `lambda.DockerImageFunction` y su rol IAM |

Al adoptar CDK, pásalo a ser la única fuente de verdad de los recursos. Conserva esta guía como explicación de cada decisión y para depurar lo que CDK crea.
