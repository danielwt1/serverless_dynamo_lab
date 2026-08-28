# Guía del laboratorio: usuarios

Esta es la ruta única de lectura. Primero explica los conceptos; después baja a las decisiones concretas, el código y el despliegue para que otra persona pueda reproducir el laboratorio.

1. [Conceptos del laboratorio](00-conceptos-del-lab.md)
2. [Modelo DynamoDB](01-modelo-dynamodb.md)
3. [Implementación de Lambdas](02-implementacion-lambdas.md)
4. [Contrato HTTP](03-contrato-api.md)
5. [Infraestructura AWS](04-infraestructura-aws.md)
6. [Despliegue y verificación](05-despliegue-y-verificacion.md)
7. [Guía manual de AWS CLI](comandos-aws-cli.md)
8. [Infraestructura como código con CDK y JavaScript](../04-cdk/README.md)

## Alcance actual

Están implementadas las Lambdas create-user, authenticate-user y list-active-users, junto con el contrato OpenAPI. La infraestructura puede recorrerse manualmente con AWS CLI para aprender cada recurso, o desplegarse como un stack independiente mediante CDK con JavaScript. No combines ambos recorridos sobre los mismos recursos. Eventos de `UserRegistered`, desactivación de usuarios y emisión de tokens no están implementados.
