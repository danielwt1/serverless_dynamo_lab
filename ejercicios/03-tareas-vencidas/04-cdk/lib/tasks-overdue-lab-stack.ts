import * as cdk from 'aws-cdk-lib';
import * as events from 'aws-cdk-lib/aws-events';
import * as targets from 'aws-cdk-lib/aws-events-targets';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as subscriptions from 'aws-cdk-lib/aws-sns-subscriptions';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import * as path from 'node:path';
import { HTTP_OPERATIONS, TASKS_CONFIGURATION } from './config/tasks-config';
import { TaskLambda } from './constructs/task-lambda';
import { TasksHttpApi, type RouteBinding } from './constructs/tasks-http-api';
import { TasksTables } from './constructs/tasks-tables';

export class TasksOverdueLabStack extends cdk.Stack {
  public constructor(scope: cdk.App, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    for (const [key, value] of Object.entries(TASKS_CONFIGURATION.tags)) cdk.Tags.of(this).add(key, value);

    const tables = new TasksTables(this, 'Tables');
    const overdueTopic = new sns.Topic(this, 'TaskOverdueTopic', { displayName: 'Task overdue events' });
    const imageBaseDirectory = path.resolve(__dirname, '../../../02-lambda/functions');

    const createTask = new TaskLambda(this, 'CreateTaskLambda', {
      directory: 'lambda_create_task', imageBaseDirectory, environment: { TASK_TABLE_NAME: tables.tasks.tableName }
    });
    allow(createTask, ['dynamodb:TransactWriteItems'], [tables.tasks.tableArn]);

    const completeTask = new TaskLambda(this, 'CompleteTaskLambda', {
      directory: 'lambda_complete_task', imageBaseDirectory, environment: { TASK_TABLE_NAME: tables.tasks.tableName }
    });
    allow(completeTask, ['dynamodb:GetItem', 'dynamodb:TransactWriteItems'], [tables.tasks.tableArn]);

    const getTasks = new TaskLambda(this, 'GetTasksLambda', {
      directory: 'get_tasks_querys', imageBaseDirectory, environment: { TASK_TABLE_NAME: tables.tasks.tableName }
    });
    allow(getTasks, ['dynamodb:Query'], [tables.tasks.tableArn]);

    const findExpired = new TaskLambda(this, 'FindExpiredTasksLambda', {
      directory: 'lambda_find_expired_task',
      imageBaseDirectory,
      environment: {
        TASK_TABLE_NAME: tables.tasks.tableName,
        PROCESS_TABLE_NAME: tables.processes.tableName,
        TASK_OVERDUE_TOPIC_ARN: overdueTopic.topicArn
      },
      timeoutSeconds: 900,
      memorySizeMiB: 512,
      reservedConcurrentExecutions: 1
    });
    allow(findExpired, ['dynamodb:Query'], [`${tables.tasks.tableArn}/index/${TASKS_CONFIGURATION.taskPendingIndexName}`]);
    allow(findExpired, ['dynamodb:GetItem', 'dynamodb:Query', 'dynamodb:UpdateItem', 'dynamodb:TransactWriteItems'], [tables.processes.tableArn, `${tables.processes.tableArn}/index/${TASKS_CONFIGURATION.processOpenIndexName}`]);
    overdueTopic.grantPublish(findExpired.function);

    const consumer = new TaskLambda(this, 'TaskOverdueConsumerLambda', {
      directory: 'lambda_task_overdue_consumer', imageBaseDirectory, environment: { PROCESS_TABLE_NAME: tables.processes.tableName }, timeoutSeconds: 30
    });
    allow(consumer, ['dynamodb:GetItem', 'dynamodb:UpdateItem'], [tables.processes.tableArn]);
    overdueTopic.addSubscription(new subscriptions.LambdaSubscription(consumer.function));

    const scheduleDLQ = new sqs.Queue(this, 'FindExpiredScheduleDLQ', {
      retentionPeriod: cdk.Duration.days(14),
      removalPolicy: TASKS_CONFIGURATION.removalPolicy
    });
    const schedule = new events.Rule(this, 'DailyOverdueSchedule', {
      schedule: events.Schedule.cron(TASKS_CONFIGURATION.schedule)
    });
    schedule.addTarget(new targets.LambdaFunction(findExpired.function, {
      event: events.RuleTargetInput.fromObject({ execution_time: events.EventField.time }),
      retryAttempts: 2,
      maxEventAge: cdk.Duration.hours(2),
      deadLetterQueue: scheduleDLQ
    }));

    const bindings: RouteBinding[] = [
      { ...HTTP_OPERATIONS[0]!, fn: createTask.function },
      { ...HTTP_OPERATIONS[1]!, fn: completeTask.function },
      { ...HTTP_OPERATIONS[2]!, fn: getTasks.function },
      { ...HTTP_OPERATIONS[3]!, fn: getTasks.function }
    ];
    const api = new TasksHttpApi(this, 'HttpApi', bindings);

    new cdk.CfnOutput(this, 'ApiEndpoint', { value: api.api.apiEndpoint });
    new cdk.CfnOutput(this, 'TasksTableName', { value: tables.tasks.tableName });
    new cdk.CfnOutput(this, 'ProcessesTableName', { value: tables.processes.tableName });
    new cdk.CfnOutput(this, 'TaskOverdueTopicArn', { value: overdueTopic.topicArn });
    new cdk.CfnOutput(this, 'ScheduleDLQUrl', { value: scheduleDLQ.queueUrl });
  }
}

function allow(target: TaskLambda, actions: string[], resources: string[]): void {
  target.function.addToRolePolicy(new iam.PolicyStatement({ actions, resources }));
}
