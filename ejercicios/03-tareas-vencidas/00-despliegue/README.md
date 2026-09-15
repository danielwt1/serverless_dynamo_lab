# Despliegue

Requisitos: Docker, Node.js, credenciales AWS y CDK bootstrap en la cuenta/región.

```code
cd 04-cdk
npm install
npm test
npm run synth
npm run deploy
```

Usa el output `ApiEndpoint` para llamar la API. La regla diaria ejecuta el batch a las `05:00 UTC`; `ScheduleDLQUrl` identifica la cola de fallos definitivos.

Antes de desplegar, estudia la [arquitectura global](../documentacion/00-arquitectura-y-flujo-global.md) y la [infraestructura creada por CDK](../04-cdk/README.md). `npm run synth` genera la plantilla CloudFormation sin modificar AWS; `npm run deploy` crea o actualiza recursos en la cuenta configurada.

Para eliminar todos los recursos del laboratorio:

```code
npm run destroy
```

`destroy` elimina recursos configurados con `RemovalPolicy.DESTROY`, incluidas las tablas del laboratorio. Revisa los datos antes de ejecutarlo.
