#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { BlogPublicacionStack } from '../lib/blog-publicacion-stack';

const app = new cdk.App();

new BlogPublicacionStack(app, 'BlogPublicacionStack', {
  description: 'Laboratorio Blog: DynamoDB, Lambdas Go, SQS, Streams, API Gateway e IAM.'
});
