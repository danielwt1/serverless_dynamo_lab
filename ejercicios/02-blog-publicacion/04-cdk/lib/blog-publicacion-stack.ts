import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as lambdaEvents from 'aws-cdk-lib/aws-lambda-event-sources';
import * as path from 'node:path';
import { BLOG_CONFIGURATION, BLOG_HTTP_OPERATIONS } from './config/blog-config';
import { BlogHttpApi } from './constructs/blog-http-api';
import { BlogLambda } from './constructs/blog-lambda';
import { BlogQueues } from './constructs/blog-queues';
import { BlogTables } from './constructs/blog-tables';

/** Conecta persistencia, colas, funciones y la entrada HTTP del laboratorio. */
export class BlogPublicacionStack extends cdk.Stack {
  public constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    for (const [key, value] of Object.entries(BLOG_CONFIGURATION.tags)) cdk.Tags.of(this).add(key, value);

    const tables = new BlogTables(this, 'Tables');
    const queues = new BlogQueues(this, 'Queues');
    // Al ejecutarse compilado, __dirname es 04-cdk/dist/lib. Tres niveles suben a 02-blog-publicacion.
    const imageBaseDirectory = path.resolve(__dirname, '../../../02-lambda/functions');

    const createPost = this.createHttpLambda('CreatePostLambda', 'lambda_create_post', imageBaseDirectory, { TABLENAME: tables.posts.tableName });
    this.allow(createPost, ['dynamodb:PutItem'], [tables.posts.tableArn]);
    const getPosts = this.createHttpLambda('GetPostsLambda', 'lambda_get_posts', imageBaseDirectory, { TABLENAME: tables.posts.tableName });
    this.allow(getPosts, ['dynamodb:Query'], [tables.posts.tableArn, `${tables.posts.tableArn}/index/${BLOG_CONFIGURATION.table.draftIndexName}`]);

    const starter = new BlogLambda(this, 'FanoutStarterLambda', {
      directory: 'fanout_starter', imageBaseDirectory,
      environment: { FANOUT_PROGRESS_TABLE: tables.progress.tableName, FANOUT_JOBS_QUEUE_URL: queues.fanout.queueUrl }
    });
    this.allow(starter, ['dynamodb:PutItem', 'dynamodb:UpdateItem'], [tables.progress.tableArn]);
    queues.fanout.grantSendMessages(starter.function);
    starter.function.addEventSource(new lambdaEvents.DynamoEventSource(tables.posts, { startingPosition: lambda.StartingPosition.LATEST, batchSize: 10, bisectBatchOnError: true, retryAttempts: 5 }));

    const worker = new BlogLambda(this, 'FollowerFanoutWorkerLambda', {
      directory: 'follower_fanout_worker', imageBaseDirectory,
      environment: {
        FOLLOWERS_TABLE: tables.posts.tableName, FANOUT_PROGRESS_TABLE: tables.progress.tableName,
        FANOUT_JOBS_QUEUE_URL: queues.fanout.queueUrl, NOTIFICATION_JOBS_QUEUE_URL: queues.notifications.queueUrl,
        FANOUT_LEASE_SECONDS: BLOG_CONFIGURATION.lambda.fanoutLeaseSeconds.toString()
      }
    });
    this.allow(worker, ['dynamodb:Query'], [tables.posts.tableArn]);
    this.allow(worker, ['dynamodb:GetItem', 'dynamodb:UpdateItem', 'dynamodb:TransactWriteItems'], [tables.progress.tableArn]);
    queues.fanout.grantSendMessages(worker.function);
    queues.notifications.grantSendMessages(worker.function);
    worker.function.addEventSource(new lambdaEvents.SqsEventSource(queues.fanout, { batchSize: 10, maxConcurrency: 2, reportBatchItemFailures: true }));

    const api = new BlogHttpApi(this, 'HttpApi', {
      operations: [
        { operation: BLOG_HTTP_OPERATIONS[0]!, fn: createPost.function },
        { operation: BLOG_HTTP_OPERATIONS[1]!, fn: getPosts.function },
        { operation: BLOG_HTTP_OPERATIONS[2]!, fn: getPosts.function }
      ]
    });
    new cdk.CfnOutput(this, 'ApiEndpoint', { value: api.api.apiEndpoint, description: 'URL base de la HTTP API del Blog.' });
    new cdk.CfnOutput(this, 'PostsTableName', { value: tables.posts.tableName, description: 'Nombre físico de la tabla de posts.' });
    new cdk.CfnOutput(this, 'FanoutProgressTableName', { value: tables.progress.tableName, description: 'Nombre físico de la tabla de progreso.' });
    new cdk.CfnOutput(this, 'FanoutQueueUrl', { value: queues.fanout.queueUrl, description: 'URL de la cola de trabajos de fanout.' });
  }

  private createHttpLambda(id: string, directory: string, imageBaseDirectory: string, environment: Record<string, string>): BlogLambda {
    return new BlogLambda(this, id, { directory, imageBaseDirectory, environment });
  }
  private allow(target: BlogLambda, actions: string[], resources: string[]): void {
    target.function.addToRolePolicy(new cdk.aws_iam.PolicyStatement({ actions, resources }));
  }
}
