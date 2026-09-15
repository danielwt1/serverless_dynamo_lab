import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import { TASKS_CONFIGURATION, type HttpMethod } from '../config/tasks-config';

export interface RouteBinding { readonly id: string; readonly method: HttpMethod; readonly path: string; readonly fn: cdk.aws_lambda.IFunction; }

export class TasksHttpApi extends Construct {
  public readonly api: apigatewayv2.HttpApi;

  public constructor(scope: Construct, id: string, bindings: readonly RouteBinding[]) {
    super(scope, id);
    const accessLogs = new logs.LogGroup(this, 'AccessLogs', { retention: TASKS_CONFIGURATION.logRetention, removalPolicy: TASKS_CONFIGURATION.removalPolicy });
    this.api = new apigatewayv2.HttpApi(this, 'Resource', { apiName: 'tasks-overdue-api', createDefaultStage: false });
    for (const binding of bindings) {
      new apigatewayv2.HttpRoute(this, `${binding.id}Route`, {
        httpApi: this.api,
        routeKey: apigatewayv2.HttpRouteKey.with(binding.path, toMethod(binding.method)),
        integration: new integrations.HttpLambdaIntegration(`${binding.id}Integration`, binding.fn, {
          payloadFormatVersion: apigatewayv2.PayloadFormatVersion.VERSION_2_0,
          timeout: cdk.Duration.seconds(9)
        })
      });
    }
    new apigatewayv2.HttpStage(this, 'DefaultStage', {
      httpApi: this.api,
      stageName: '$default',
      autoDeploy: true,
      accessLogSettings: {
        destination: new apigatewayv2.LogGroupLogDestination(accessLogs),
        format: apigateway.AccessLogFormat.custom(JSON.stringify({ requestId: '$context.requestId', status: '$context.status', routeKey: '$context.routeKey', integrationError: '$context.integrationErrorMessage' }))
      }
    });
  }
}

function toMethod(method: HttpMethod): apigatewayv2.HttpMethod {
  if (method === 'POST') return apigatewayv2.HttpMethod.POST;
  if (method === 'PATCH') return apigatewayv2.HttpMethod.PATCH;
  return apigatewayv2.HttpMethod.GET;
}
