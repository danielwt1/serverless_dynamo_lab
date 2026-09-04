import * as cdk from 'aws-cdk-lib';
import * as ecrAssets from 'aws-cdk-lib/aws-ecr-assets';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import * as path from 'node:path';
import { BLOG_CONFIGURATION } from '../config/blog-config';

export interface BlogLambdaProps { readonly directory: string; readonly imageBaseDirectory: string; readonly environment: Record<string, string>; }

/** Empaqueta una Lambda Go como imagen y le asigna un rol y logs aislados. */
export class BlogLambda extends Construct {
  public readonly function: lambda.DockerImageFunction;
  public constructor(scope: Construct, id: string, props: BlogLambdaProps) {
    super(scope, id);
    const logGroup = new logs.LogGroup(this, 'Logs', { retention: BLOG_CONFIGURATION.lambda.logRetention, removalPolicy: BLOG_CONFIGURATION.lambda.logRemovalPolicy });
    const role = new iam.Role(this, 'Role', { assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com') });
    role.addToPolicy(new iam.PolicyStatement({ actions: ['logs:CreateLogStream', 'logs:PutLogEvents'], resources: [`${logGroup.logGroupArn}:*`] }));
    this.function = new lambda.DockerImageFunction(this, 'Function', {
      code: lambda.DockerImageCode.fromImageAsset(path.join(props.imageBaseDirectory, props.directory), { platform: ecrAssets.Platform.LINUX_ARM64 }), architecture: lambda.Architecture.ARM_64, role,
      environment: props.environment, memorySize: BLOG_CONFIGURATION.lambda.memorySizeMiB, timeout: cdk.Duration.seconds(BLOG_CONFIGURATION.lambda.timeoutSeconds), logGroup,
      loggingFormat: lambda.LoggingFormat.JSON, applicationLogLevelV2: lambda.ApplicationLogLevel.INFO, systemLogLevelV2: lambda.SystemLogLevel.INFO
    });
  }
}
