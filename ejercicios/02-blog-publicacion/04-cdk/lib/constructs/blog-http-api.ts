import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import { BLOG_CONFIGURATION, type BlogOperation } from '../config/blog-config';

export interface BlogHttpApiProps { readonly operations: readonly { readonly operation: BlogOperation; readonly fn: lambda.IFunction }[]; }

/** HTTP API, sus rutas y permisos de invocación restringidos a cada ruta. */
export class BlogHttpApi extends Construct {
  public readonly api: apigatewayv2.HttpApi;
  public constructor(scope: Construct, id: string, props: BlogHttpApiProps) {
    super(scope, id);
    const accessLogs = new logs.LogGroup(this, 'AccessLogs', { retention: BLOG_CONFIGURATION.api.logRetention, removalPolicy: BLOG_CONFIGURATION.api.logRemovalPolicy });
    this.api = new apigatewayv2.HttpApi(this, 'Resource', { apiName: BLOG_CONFIGURATION.api.name, createDefaultStage: false });
    for (const { operation, fn } of props.operations) {
      new apigatewayv2.HttpRoute(this, `${operation.id}Route`, {
        httpApi: this.api, routeKey: apigatewayv2.HttpRouteKey.with(operation.route.path, operation.route.method === 'GET' ? apigatewayv2.HttpMethod.GET : apigatewayv2.HttpMethod.POST),
        integration: new integrations.HttpLambdaIntegration(`${operation.id}Integration`, fn, { payloadFormatVersion: apigatewayv2.PayloadFormatVersion.VERSION_2_0, timeout: cdk.Duration.millis(BLOG_CONFIGURATION.api.integrationTimeoutMilliseconds), scopePermissionToRoute: true })
      });
    }
    new apigatewayv2.HttpStage(this, 'DefaultStage', {
      httpApi: this.api, stageName: '$default', autoDeploy: true,
      accessLogSettings: { destination: new apigatewayv2.LogGroupLogDestination(accessLogs), format: apigateway.AccessLogFormat.custom(JSON.stringify({ requestId: '$context.requestId', status: '$context.status', routeKey: '$context.routeKey', integrationError: '$context.integrationErrorMessage' })) }
    });
  }
}
