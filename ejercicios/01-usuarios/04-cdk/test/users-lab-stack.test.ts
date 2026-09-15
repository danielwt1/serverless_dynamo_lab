import * as cdk from 'aws-cdk-lib';
import { Match, Template } from 'aws-cdk-lib/assertions';
import { test } from 'node:test';
import { UsersLabStack } from '../lib/users-lab-stack';

/**
 * Pruebas de plantilla CloudFormation, sin acceder a AWS. Previenen regresiones
 * de seguridad y topología antes de un despliegue real.
 */
function synthesizeTemplate(): Template {
  const app = new cdk.App();
  const stack = new UsersLabStack(app, 'UsersLabStackForTest');
  return Template.fromStack(stack);
}

test('mantiene el laboratorio eliminable sin recursos retenidos', () => {
  const template = synthesizeTemplate();
  template.hasResourceProperties('AWS::DynamoDB::Table', {
    DeletionProtectionEnabled: false,
    PointInTimeRecoverySpecification: { PointInTimeRecoveryEnabled: false }
  });
});

test('mantiene una Lambda y un permiso de invocación por operación', () => {
  const template = synthesizeTemplate();
  template.resourceCountIs('AWS::Lambda::Function', 3);
  template.resourceCountIs('AWS::Lambda::Permission', 3);
  template.resourceCountIs('AWS::ApiGatewayV2::Route', 3);
});

test('deja margen entre el timeout de la API y el de Lambda', () => {
  const template = synthesizeTemplate();
  template.hasResourceProperties('AWS::ApiGatewayV2::Integration', {
    TimeoutInMillis: 9000,
    PayloadFormatVersion: '2.0',
    IntegrationType: 'AWS_PROXY',
    IntegrationUri: Match.anyValue()
  });
});
