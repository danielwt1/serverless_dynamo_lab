const cdk = require('aws-cdk-lib');
const dynamodb = require('aws-cdk-lib/aws-dynamodb');
const ecrAssets = require('aws-cdk-lib/aws-ecr-assets');
const iam = require('aws-cdk-lib/aws-iam');
const lambda = require('aws-cdk-lib/aws-lambda');
const logs = require('aws-cdk-lib/aws-logs');
const apigatewayv2 = require('aws-cdk-lib/aws-apigatewayv2');
const path = require('path');

class UsersLabStack extends cdk.Stack {
  constructor(scope, id, props) {
    super(scope, id, props);

    cdk.Tags.of(this).add('environment', 'dev');
    cdk.Tags.of(this).add('system', 'users');
    cdk.Tags.of(this).add('owner', 'backend');
    cdk.Tags.of(this).add('managed-by', 'cdk');

    const table = new dynamodb.Table(this, 'UsersTable', {
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: cdk.RemovalPolicy.DESTROY
    });

    table.addGlobalSecondaryIndex({
      indexName: 'GSI1',
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.INCLUDE,
      nonKeyAttributes: ['userId', 'name', 'lastName', 'email', 'state', 'created_at']
    });

    const createRole = this.createExecutionRole('CreateUserRole');
    createRole.addToPolicy(new iam.PolicyStatement({
      actions: ['dynamodb:PutItem', 'dynamodb:TransactWriteItems'],
      resources: [table.tableArn]
    }));

    const authenticateRole = this.createExecutionRole('AuthenticateUserRole');
    authenticateRole.addToPolicy(new iam.PolicyStatement({
      actions: ['dynamodb:GetItem'],
      resources: [table.tableArn]
    }));

    const listRole = this.createExecutionRole('ListActiveUsersRole');
    listRole.addToPolicy(new iam.PolicyStatement({
      actions: ['dynamodb:Query'],
      resources: [`${table.tableArn}/index/GSI1`]
    }));

    const createUser = this.createFunction('CreateUserFunction', {
      directory: '../../02-lambda/functions/create-user',
      role: createRole,
      environment: { USERS_TABLE_NAME: table.tableName }
    });
    const authenticateUser = this.createFunction('AuthenticateUserFunction', {
      directory: '../../02-lambda/functions/authenticate-user',
      role: authenticateRole,
      environment: { TABLE_NAME: table.tableName }
    });
    const listActiveUsers = this.createFunction('ListActiveUsersFunction', {
      directory: '../../02-lambda/functions/list-active-users',
      role: listRole,
      environment: { TABLE_NAME: table.tableName }
    });

    this.createLambdaLogGroup('CreateUserLogs', createUser);
    this.createLambdaLogGroup('AuthenticateUserLogs', authenticateUser);
    this.createLambdaLogGroup('ListActiveUsersLogs', listActiveUsers);

    const apiLogs = new logs.LogGroup(this, 'HttpApiAccessLogs', {
      logGroupName: `/aws/apigateway/${this.stackName}-users-http-api`,
      retention: logs.RetentionDays.THREE_DAYS,
      removalPolicy: cdk.RemovalPolicy.DESTROY
    });
    const api = new apigatewayv2.CfnApi(this, 'UsersHttpApi', {
      name: 'users-http-api',
      protocolType: 'HTTP'
    });

    const createIntegration = this.createIntegration('CreateUserIntegration', api, createUser);
    const authenticateIntegration = this.createIntegration('AuthenticateUserIntegration', api, authenticateUser);
    const listIntegration = this.createIntegration('ListActiveUsersIntegration', api, listActiveUsers);
    const createRoute = this.createRoute('CreateUserRoute', api, 'POST /users', createIntegration);
    const authenticateRoute = this.createRoute('AuthenticateUserRoute', api, 'POST /auth/login', authenticateIntegration);
    const listRoute = this.createRoute('ListActiveUsersRoute', api, 'GET /users/active', listIntegration);

    const stage = new apigatewayv2.CfnStage(this, 'DefaultStage', {
      apiId: api.ref,
      stageName: '$default',
      autoDeploy: true,
      accessLogSettings: {
        destinationArn: apiLogs.logGroupArn,
        format: JSON.stringify({
          requestId: '$context.requestId',
          status: '$context.status',
          routeKey: '$context.routeKey',
          integrationError: '$context.integrationErrorMessage'
        })
      }
    });
    stage.node.addDependency(createRoute);
    stage.node.addDependency(authenticateRoute);
    stage.node.addDependency(listRoute);

    this.allowApiGateway('AllowApiGatewayCreateUser', api, createUser, 'POST/users');
    this.allowApiGateway('AllowApiGatewayAuthenticateUser', api, authenticateUser, 'POST/auth/login');
    this.allowApiGateway('AllowApiGatewayListActiveUsers', api, listActiveUsers, 'GET/users/active');

    new cdk.CfnOutput(this, 'ApiEndpoint', {
      value: api.attrApiEndpoint,
      description: 'URL base de la HTTP API; úsala como baseUrl de Postman.'
    });
    new cdk.CfnOutput(this, 'UsersTableName', {
      value: table.tableName,
      description: 'Nombre físico de la tabla DynamoDB creada por este stack.'
    });
  }

  createExecutionRole(id) {
    return new iam.Role(this, id, {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole')]
    });
  }

  createFunction(id, { directory, role, environment }) {
    return new lambda.DockerImageFunction(this, id, {
      code: lambda.DockerImageCode.fromImageAsset(path.join(__dirname, directory), {
        platform: ecrAssets.Platform.LINUX_ARM64
      }),
      architecture: lambda.Architecture.ARM_64,
      role,
      environment,
      memorySize: 256,
      timeout: cdk.Duration.seconds(10)
    });
  }

  createLambdaLogGroup(id, fn) {
    return new logs.LogGroup(this, id, {
      logGroupName: `/aws/lambda/${fn.functionName}`,
      retention: logs.RetentionDays.THREE_DAYS,
      removalPolicy: cdk.RemovalPolicy.DESTROY
    });
  }

  createIntegration(id, api, fn) {
    return new apigatewayv2.CfnIntegration(this, id, {
      apiId: api.ref,
      integrationType: 'AWS_PROXY',
      integrationUri: `arn:${cdk.Stack.of(this).partition}:apigateway:${cdk.Stack.of(this).region}:lambda:path/2015-03-31/functions/${fn.functionArn}/invocations`,
      payloadFormatVersion: '2.0',
      timeoutInMillis: 10000
    });
  }

  createRoute(id, api, routeKey, integration) {
    return new apigatewayv2.CfnRoute(this, id, {
      apiId: api.ref,
      routeKey,
      target: `integrations/${integration.ref}`
    });
  }

  allowApiGateway(id, api, fn, methodAndPath) {
    return fn.addPermission(id, {
      action: 'lambda:InvokeFunction',
      principal: new iam.ServicePrincipal('apigateway.amazonaws.com'),
      sourceArn: cdk.Fn.join('', [
        `arn:${cdk.Stack.of(this).partition}:execute-api:`,
        cdk.Stack.of(this).region,
        ':',
        cdk.Stack.of(this).account,
        ':',
        api.ref,
        '/*/',
        methodAndPath
      ])
    });
  }
}

module.exports = { UsersLabStack };
