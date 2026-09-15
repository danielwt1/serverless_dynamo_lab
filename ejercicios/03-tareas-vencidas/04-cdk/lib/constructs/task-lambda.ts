import * as cdk from 'aws-cdk-lib';
import * as ecrAssets from 'aws-cdk-lib/aws-ecr-assets';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as path from 'node:path';
import { Construct } from 'constructs';
import { TASKS_CONFIGURATION } from '../config/tasks-config';

export interface TaskLambdaProps {
  readonly imageBaseDirectory: string;
  readonly directory: string;
  readonly environment: Record<string, string>;
  readonly timeoutSeconds?: number;
  readonly memorySizeMiB?: number;
  readonly reservedConcurrentExecutions?: number;
}

export class TaskLambda extends Construct {
  public readonly function: lambda.DockerImageFunction;

  public constructor(scope: Construct, id: string, props: TaskLambdaProps) {
    super(scope, id);
    const logGroup = new logs.LogGroup(this, 'Logs', {
      retention: TASKS_CONFIGURATION.logRetention,
      removalPolicy: TASKS_CONFIGURATION.removalPolicy
    });
    this.function = new lambda.DockerImageFunction(this, 'Function', {
      code: lambda.DockerImageCode.fromImageAsset(path.join(props.imageBaseDirectory, props.directory), { platform: ecrAssets.Platform.LINUX_ARM64 }),
      architecture: lambda.Architecture.ARM_64,
      environment: props.environment,
      timeout: cdk.Duration.seconds(props.timeoutSeconds ?? 10),
      memorySize: props.memorySizeMiB ?? 256,
      reservedConcurrentExecutions: props.reservedConcurrentExecutions,
      logGroup,
      loggingFormat: lambda.LoggingFormat.JSON,
      applicationLogLevelV2: lambda.ApplicationLogLevel.INFO,
      systemLogLevelV2: lambda.SystemLogLevel.INFO
    });
  }
}
