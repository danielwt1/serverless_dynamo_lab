import * as cdk from 'aws-cdk-lib';
import * as logs from 'aws-cdk-lib/aws-logs';

export const TASKS_CONFIGURATION = {
  taskPendingIndexName: 'task_gsi_pending',
  processOpenIndexName: 'process_gsi_open',
  logRetention: logs.RetentionDays.ONE_WEEK,
  removalPolicy: cdk.RemovalPolicy.DESTROY,
  schedule: { minute: '0', hour: '5' },
  tags: { project: 'serverless-dynamo-lab', exercise: 'tasks-overdue' }
} as const;

export type HttpMethod = 'GET' | 'POST' | 'PATCH';
export interface HttpOperation {
  readonly id: string;
  readonly directory: string;
  readonly method: HttpMethod;
  readonly path: string;
  readonly dynamoActions: readonly string[];
}

export const HTTP_OPERATIONS: readonly HttpOperation[] = [
  { id: 'CreateTask', directory: 'lambda_create_task', method: 'POST', path: '/tasks/{ownerId}', dynamoActions: ['dynamodb:TransactWriteItems'] },
  { id: 'CompleteTask', directory: 'lambda_complete_task', method: 'PATCH', path: '/tasks/{ownerId}/{taskId}/complete', dynamoActions: ['dynamodb:GetItem', 'dynamodb:TransactWriteItems'] },
  { id: 'GetTasks', directory: 'get_tasks_querys', method: 'GET', path: '/tasks/{ownerId}', dynamoActions: ['dynamodb:Query'] },
  { id: 'GetPendingTasks', directory: 'get_tasks_querys', method: 'GET', path: '/tasks/{ownerId}/pending', dynamoActions: ['dynamodb:Query'] }
] as const;
