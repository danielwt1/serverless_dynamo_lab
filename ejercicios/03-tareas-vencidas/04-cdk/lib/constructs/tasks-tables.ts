import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';
import { TASKS_CONFIGURATION } from '../config/tasks-config';

export class TasksTables extends Construct {
  public readonly tasks: dynamodb.Table;
  public readonly processes: dynamodb.Table;

  public constructor(scope: Construct, id: string) {
    super(scope, id);
    this.tasks = new dynamodb.Table(this, 'Tasks', {
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: TASKS_CONFIGURATION.removalPolicy,
      deletionProtection: false,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: false }
    });
    this.tasks.addGlobalSecondaryIndex({
      indexName: TASKS_CONFIGURATION.taskPendingIndexName,
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL
    });

    this.processes = new dynamodb.Table(this, 'Processes', {
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      timeToLiveAttribute: 'ttl',
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: TASKS_CONFIGURATION.removalPolicy,
      deletionProtection: false,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: false }
    });
    this.processes.addGlobalSecondaryIndex({
      indexName: TASKS_CONFIGURATION.processOpenIndexName,
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL
    });
  }
}
