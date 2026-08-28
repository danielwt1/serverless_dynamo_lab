# Serverless Dynamo Lab

Laboratorios prácticos de Go, DynamoDB, Lambda y arquitectura serverless. Cada caso documenta el reto de negocio, la teoría aplicada, las decisiones arquitectónicas, su reflejo en código y un camino de despliegue por consola o CLI.

## Laboratorios disponibles

- [Usuarios](ejercicios/01-usuarios/README.md): identidad, unicidad de email, GSI disperso y API HTTP.
- [Publicación de blog](ejercicios/02-blog-publicacion/README.md): estados, paginación, Streams, SQS, fanout e idempotencia.

## Cómo leer un laboratorio

1. Lee el `README.md`: contexto, problema, reglas y objetivo.
2. Continúa en `documentacion/00-conceptos-del-lab.md`: teoría aplicada al reto.
3. Revisa las decisiones concretas: modelo DynamoDB, arquitectura/código, contrato, infraestructura y despliegue.
4. Usa el código, OpenAPI, scripts y colección Postman para comprobar la implementación.

```text
ejercicios/<laboratorio>/
├── README.md                 reto y diseño general
├── documentacion/            teoría, decisiones y despliegue
├── 01-dynamodb/              activos del modelo, si aplica
├── 02-lambda/                código de las funciones
├── 03-api/                   contrato HTTP, si aplica
└── postman/                  pruebas manuales, si aplica
```

## Principio de diseño

Una consulta debe poder explicarse como una necesidad de negocio. A partir de ella se define el patrón de acceso, el modelo DynamoDB, la operación, la arquitectura y finalmente los recursos AWS; no al revés.
