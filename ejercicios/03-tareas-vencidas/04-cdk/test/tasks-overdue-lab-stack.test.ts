import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { test } from 'node:test';
import { TasksOverdueLabStack } from '../lib/tasks-overdue-lab-stack';

function template(): Template {
  const app = new cdk.App();
  return Template.fromStack(new TasksOverdueLabStack(app, 'TasksOverdueLabStackTest'));
}

test('crea el flujo completo del ejercicio', () => {
  const synthesized = template();
  synthesized.resourceCountIs('AWS::DynamoDB::Table', 2);
  synthesized.resourceCountIs('AWS::Lambda::Function', 5);
  synthesized.resourceCountIs('AWS::ApiGatewayV2::Route', 4);
  synthesized.resourceCountIs('AWS::SNS::Topic', 1);
  synthesized.resourceCountIs('AWS::Events::Rule', 1);
  synthesized.resourceCountIs('AWS::SQS::Queue', 1);
});

test('configura el índice de vencidas y TTL de checkpoints', () => {
  const synthesized = template();
  synthesized.hasResourceProperties('AWS::DynamoDB::Table', {
    GlobalSecondaryIndexes: Match.arrayWith([Match.objectLike({ IndexName: 'task_gsi_pending' })])
  });
  synthesized.hasResourceProperties('AWS::DynamoDB::Table', {
    TimeToLiveSpecification: { AttributeName: 'ttl', Enabled: true },
    GlobalSecondaryIndexes: Match.arrayWith([Match.objectLike({ IndexName: 'process_gsi_open' })])
  });
});

test('reserva una sola ejecución concurrente para el batch', () => {
  template().hasResourceProperties('AWS::Lambda::Function', {
    ReservedConcurrentExecutions: 1,
    Timeout: 900,
    Environment: { Variables: Match.objectLike({ TASK_OVERDUE_TOPIC_ARN: Match.anyValue() }) }
  });
});

test('configura reintentos y DLQ para el schedule', () => {
  template().hasResourceProperties('AWS::Events::Rule', {
    State: 'ENABLED',
    Targets: Match.arrayWith([Match.objectLike({ RetryPolicy: { MaximumEventAgeInSeconds: 7200, MaximumRetryAttempts: 2 }, DeadLetterConfig: Match.anyValue() })])
  });
});
