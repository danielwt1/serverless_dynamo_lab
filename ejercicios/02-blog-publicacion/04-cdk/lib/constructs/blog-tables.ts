import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import { Construct } from 'constructs';
import { BLOG_CONFIGURATION } from '../config/blog-config';

/** Las dos tablas del ejercicio y el Stream que inicia el fanout. */
export class BlogTables extends Construct {
  public readonly posts: dynamodb.Table;
  public readonly progress: dynamodb.Table;

  public constructor(scope: Construct, id: string) {
    super(scope, id);
    const table = BLOG_CONFIGURATION.table;
    this.posts = new dynamodb.Table(this, 'Posts', {
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING }, sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING }, billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      stream: dynamodb.StreamViewType.NEW_AND_OLD_IMAGES, removalPolicy: table.removalPolicy, deletionProtection: table.deletionProtection,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: table.pointInTimeRecoveryEnabled }
    });
    this.posts.addGlobalSecondaryIndex({ indexName: table.draftIndexName, partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING }, sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING }, projectionType: dynamodb.ProjectionType.ALL });
    this.progress = new dynamodb.Table(this, 'FanoutProgress', {
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING }, sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING }, billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: table.removalPolicy, deletionProtection: table.deletionProtection,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: table.pointInTimeRecoveryEnabled }
    });
  }
}
