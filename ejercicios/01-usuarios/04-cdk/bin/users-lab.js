#!/usr/bin/env node

const cdk = require('aws-cdk-lib');
const { UsersLabStack } = require('../lib/users-lab-stack');

const app = new cdk.App();

new UsersLabStack(app, 'UsersLabStack', {
  description: 'Laboratorio Usuarios: DynamoDB, Lambdas Docker, API Gateway HTTP e IAM.'
});
