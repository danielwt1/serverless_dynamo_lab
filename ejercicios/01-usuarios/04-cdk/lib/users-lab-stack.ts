import * as cdk from 'aws-cdk-lib';
import * as path from 'node:path';
import { USERS_LAB_CONFIGURATION, USERS_OPERATIONS } from './config/users-config';
import { UsersHttpApi, type OperationWithFunction } from './constructs/users-http-api';
import { UsersLambda } from './constructs/users-lambda';
import { UsersTable } from './constructs/users-table';

/**
 * Orquestador del laboratorio: no define detalles de servicios individuales.
 * Su responsabilidad es conectar persistencia, cómputo y transporte HTTP.
 */
export class UsersLabStack extends cdk.Stack {
  public constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    for (const [tagKey, tagValue] of Object.entries(USERS_LAB_CONFIGURATION.tags)) {
      cdk.Tags.of(this).add(tagKey, tagValue);
    }

    const usersTable = new UsersTable(this, 'UsersTable', {
      configuration: USERS_LAB_CONFIGURATION.table
    });
    // `__dirname` apunta a dist/lib al ejecutar CDK compilado; subir tres niveles
    // llega a 01-usuarios, donde están los contextos Docker de las Lambdas Go.
    const imageBaseDirectory = path.resolve(__dirname, '../../../02-lambda');
    const lambdaConfiguration = { ...USERS_LAB_CONFIGURATION.lambda, ...USERS_LAB_CONFIGURATION.table };
    const operationsWithFunctions: OperationWithFunction[] = [];

    // Por cada operación declarada, crea su conjunto aislado: logs, rol IAM,
    // política de logs, política DynamoDB y Lambda. Después conserva la pareja
    // operación + Lambda para que UsersHttpApi cree la ruta correspondiente.
    for (const operation of USERS_OPERATIONS) {
      const usersLambda = new UsersLambda(this, operation.id, {
        operation,
        table: usersTable.table,
        imageBaseDirectory,
        configuration: lambdaConfiguration
      });

      operationsWithFunctions.push({
        operation,
        fn: usersLambda.lambdaFunction
      });
    }
    const usersHttpApi = new UsersHttpApi(this, 'UsersHttpApi', {
      operationsWithFunctions,
      configuration: USERS_LAB_CONFIGURATION.api
    });

    // Outputs no son secretos: permiten probar el stack sin adivinar nombres físicos.
    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: usersHttpApi.api.apiEndpoint,
      description: 'URL base de la HTTP API; úsala como baseUrl de Postman.'
    });
    new cdk.CfnOutput(this, 'UsersTableName', {
      value: usersTable.table.tableName,
      description: 'Nombre físico de la tabla DynamoDB creada por este stack.'
    });
  }
}
