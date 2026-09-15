import * as cdk from 'aws-cdk-lib';
import * as ecrAssets from 'aws-cdk-lib/aws-ecr-assets';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import * as path from 'node:path';
import type { UsersLabConfiguration, UsersOperation } from '../config/users-config';

/** Propiedades necesarias para una operación Lambda del dominio Usuarios. */
export interface UsersLambdaProps {
  readonly operation: UsersOperation;
  readonly table: cdk.aws_dynamodb.ITable;
  readonly imageBaseDirectory: string;
  readonly configuration: UsersLabConfiguration['lambda'] & UsersLabConfiguration['table'];
}

/**
 * Componente de una operación: imagen Docker, Lambda, grupo de logs y rol IAM.
 * Encapsularlos evita que otro componente pueda olvidar alguno de estos cuatro
 * recursos, pero expone la función para conectarla a HTTP API.
 */
export class UsersLambda extends Construct {
  public readonly lambdaFunction: lambda.DockerImageFunction;

  public constructor(scope: Construct, id: string, props: UsersLambdaProps) {
    super(scope, id);

    const { operation, table, imageBaseDirectory, configuration } = props;
    // Crear el grupo primero permite restringir el rol a un ARN de logs concreto.
    const logGroup = new logs.LogGroup(this, 'Logs', {
      // Cuántos días se conservan logs de esta Lambda.
      retention: configuration.logRetention,
      // El laboratorio borra sus logs junto con el stack.
      removalPolicy: configuration.logRemovalPolicy
    });
    const role = new iam.Role(this, 'Role', {
      // Solo el servicio Lambda puede asumir este rol en tiempo de ejecución.
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com')
    });
    role.addToPolicy(new iam.PolicyStatement({
      // Acciones mínimas para escribir en el grupo de logs ya creado.
      actions: ['logs:CreateLogStream', 'logs:PutLogEvents'],
      // `:*` cubre los streams dentro de este único Log Group, no otros grupos.
      resources: [`${logGroup.logGroupArn}:*`]
    }));

    const dynamoDbResource = operation.dynamoDb.target === 'activeUsersIndex'
      ? `${table.tableArn}/index/${configuration.activeUsersIndexName}`
      : table.tableArn;
    role.addToPolicy(new iam.PolicyStatement({
      // Cada operación declara sus propias acciones: crear, consultar o leer.
      actions: [...operation.dynamoDb.actions],
      // Create y login usan la tabla; el listado solo puede consultar el GSI1.
      resources: [dynamoDbResource]
    }));

    this.lambdaFunction = new lambda.DockerImageFunction(this, 'Function', {
      // Cada directorio es un contexto Docker independiente con su propio go.mod.
      code: lambda.DockerImageCode.fromImageAsset(path.join(imageBaseDirectory, operation.directory), {
        // La imagen debe coincidir con la arquitectura ARM_64 declarada abajo.
        platform: ecrAssets.Platform.LINUX_ARM64
      }),
      // Graviton/ARM64: requiere la imagen ARM64, normalmente reduce costo.
      architecture: lambda.Architecture.ARM_64,
      // Rol IAM aislado de esta operación; no se comparte entre Lambdas.
      role,
      // La Lambda recibe el nombre físico de la tabla sin escribirlo a mano.
      environment: { [operation.environmentVariableName]: table.tableName },
      // Memoria disponible para el proceso de la función.
      memorySize: configuration.memorySizeMiB,
      // Límite duro de ejecución de Lambda; supera al timeout de API en 1 s.
      timeout: cdk.Duration.seconds(configuration.timeoutSeconds),
      // Evita que Lambda cree un grupo implícito con nombre difícil de gobernar.
      logGroup,
      // Hace que los logs de aplicación tengan estructura JSON.
      loggingFormat: lambda.LoggingFormat.JSON,
      // Nivel mínimo de eventos que la aplicación y el runtime envían a logs.
      applicationLogLevelV2: lambda.ApplicationLogLevel.INFO,
      systemLogLevelV2: lambda.SystemLogLevel.INFO
    });
  }
}
