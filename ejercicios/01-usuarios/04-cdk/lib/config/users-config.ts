import * as cdk from 'aws-cdk-lib';
import * as logs from 'aws-cdk-lib/aws-logs';

export type DynamoDbTarget = 'table' | 'activeUsersIndex';

/** Datos mínimos para crear una Lambda, su permiso DynamoDB y su ruta HTTP. */
export interface UsersOperation {
  id: string;
  directory: string;
  environmentVariableName: string;
  dynamoDb: {
    actions: string[];
    target: DynamoDbTarget;
  };
  route: { method: 'GET' | 'POST'; path: string };
}

export interface UsersLabConfiguration {
  tags: Record<string, string>;
  table: {
    activeUsersIndexName: string;
    removalPolicy: cdk.RemovalPolicy;
    deletionProtection: boolean;
    pointInTimeRecoveryEnabled: boolean;
  };
  lambda: {
    memorySizeMiB: number;
    timeoutSeconds: number;
    logRetention: logs.RetentionDays;
    logRemovalPolicy: cdk.RemovalPolicy;
  };
  api: {
    name: string;
    logRetention: logs.RetentionDays;
    logRemovalPolicy: cdk.RemovalPolicy;
    integrationTimeoutMilliseconds: number;
  };
}

/**
 * Decisiones transversales del laboratorio.
 *
 * Este entorno es efímero: `cdk destroy` elimina todos los recursos, incluida
 * la tabla. Producción debe declarar otra configuración con RETAIN, protección
 * contra borrado y PITR; esas medidas también generan costo de respaldo.
 */
export const USERS_LAB_CONFIGURATION: UsersLabConfiguration = {
  tags: { environment: 'dev', system: 'users', owner: 'backend', 'managed-by': 'cdk' },
  table: {
    activeUsersIndexName: 'GSI1',
    removalPolicy: cdk.RemovalPolicy.DESTROY,
    deletionProtection: false,
    pointInTimeRecoveryEnabled: false
  },
  lambda: {
    memorySizeMiB: 256,
    timeoutSeconds: 10,
    logRetention: logs.RetentionDays.THREE_DAYS,
    logRemovalPolicy: cdk.RemovalPolicy.DESTROY
  },
  api: {
    name: 'users-http-api',
    logRetention: logs.RetentionDays.THREE_DAYS,
    logRemovalPolicy: cdk.RemovalPolicy.DESTROY,
    // Debe ser menor que el timeout de Lambda para devolver un 504 controlado.
    integrationTimeoutMilliseconds: 9000
  }
};

/**
 * Catálogo único de operaciones HTTP. API, integración y permiso Lambda se
 * derivan de estos datos, evitando que método, ruta e IAM diverjan.
 */
export const USERS_OPERATIONS: UsersOperation[] = [
  {
    id: 'CreateUser',
    directory: 'functions/create-user',
    environmentVariableName: 'USERS_TABLE_NAME',
    dynamoDb: { actions: ['dynamodb:PutItem', 'dynamodb:TransactWriteItems'], target: 'table' },
    route: { method: 'POST', path: '/users' }
  },
  {
    id: 'AuthenticateUser',
    directory: 'functions/authenticate-user',
    environmentVariableName: 'TABLE_NAME',
    dynamoDb: { actions: ['dynamodb:GetItem'], target: 'table' },
    route: { method: 'POST', path: '/auth/login' }
  },
  {
    id: 'ListActiveUsers',
    directory: 'functions/list-active-users',
    environmentVariableName: 'TABLE_NAME',
    dynamoDb: { actions: ['dynamodb:Query'], target: 'activeUsersIndex' },
    route: { method: 'GET', path: '/users/active' }
  }
];
