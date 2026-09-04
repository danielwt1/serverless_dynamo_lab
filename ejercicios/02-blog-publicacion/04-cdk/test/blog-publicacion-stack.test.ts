import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { test } from 'node:test';
import { BlogPublicacionStack } from '../lib/blog-publicacion-stack';

function template(): Template { const app = new cdk.App(); return Template.fromStack(new BlogPublicacionStack(app, 'BlogPublicacionStackForTest')); }

test('crea la topología principal del laboratorio', () => {
  const synthesized = template();
  synthesized.resourceCountIs('AWS::DynamoDB::Table', 2);
  synthesized.resourceCountIs('AWS::SQS::Queue', 4);
  synthesized.resourceCountIs('AWS::Lambda::Function', 4);
  synthesized.resourceCountIs('AWS::Lambda::EventSourceMapping', 2);
  synthesized.resourceCountIs('AWS::ApiGatewayV2::Route', 3);
});
test('mantiene Stream, GSI, DLQ y respuesta parcial SQS', () => {
  const synthesized = template();
  synthesized.hasResourceProperties('AWS::DynamoDB::Table', { StreamSpecification: { StreamViewType: 'NEW_AND_OLD_IMAGES' }, GlobalSecondaryIndexes: Match.arrayWith([Match.objectLike({ IndexName: 'draft_gsi' })]) });
  synthesized.hasResourceProperties('AWS::Lambda::EventSourceMapping', { FunctionResponseTypes: ['ReportBatchItemFailures'], BatchSize: 10 });
});
