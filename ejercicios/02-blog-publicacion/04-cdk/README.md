# CDK: laboratorio Blog Publicación

Esta carpeta expresa como infraestructura como código el ejercicio Blog. Crea una copia independiente de los recursos manuales del laboratorio; no mezcles ambos mecanismos sobre los mismos recursos AWS.

La topología contiene las cuatro Lambdas Go actuales, tablas DynamoDB, Stream, SQS con DLQ, roles IAM mínimos, grupos de logs y una HTTP API. CDK construye las imágenes Docker como assets privados del bootstrap, por lo que no crea los repositorios ECR manuales `blog-*`.

```text
04-cdk/
├── bin/                 punto de entrada de CDK
├── lib/config/          decisiones y catálogo de rutas
├── lib/constructs/      componentes de tablas, colas, Lambdas y API
├── lib/blog-publicacion-stack.ts
├── test/                pruebas de la plantilla CloudFormation
└── documentacion/       inventario de servicios creados
```

Consulta el [inventario de servicios](documentacion/01-inventario-servicios.md) antes de desplegar.

## Verificar sin cambiar AWS

Desde esta carpeta, después de instalar dependencias, ejecuta:

```code
npm install
npm run build
npm test
npm run synth
```

`build`, `test` y `synth` no crean recursos AWS. `synth` puede construir localmente los assets Docker para representar las cuatro imágenes en la plantilla. Para desplegar en una cuenta aislada, primero revisa `npm run diff`; `npm run deploy` y `npm run destroy` sí cambian recursos AWS y se deben ejecutar explícitamente.

La cola `notification-jobs` se crea con su DLQ, pero no tiene event source mapping todavía: el consumidor final de correo/push aún no existe en el código del ejercicio.
