#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { UsersLabStack } from '../lib/users-lab-stack';

/** Punto de entrada: aquí se elige qué stacks forman la aplicación CDK. */
const app = new cdk.App();

new UsersLabStack(app, 'UsersLabStack', {
  description: 'Laboratorio Usuarios: DynamoDB, Lambdas Docker, API Gateway HTTP e IAM.'
});
