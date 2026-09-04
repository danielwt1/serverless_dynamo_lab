import * as cdk from 'aws-cdk-lib';
import * as logs from 'aws-cdk-lib/aws-logs';

export interface BlogOperation {
  readonly id: string;
  readonly directory: string;
  readonly route: { readonly method: 'GET' | 'POST'; readonly path: string };
  readonly dynamoActions: readonly string[];
  readonly includeDraftIndex?: boolean;
}

export const BLOG_CONFIGURATION = {
  tags: { environment: 'dev', system: 'blog-publicacion', owner: 'backend', 'managed-by': 'cdk' },
  table: { draftIndexName: 'draft_gsi', removalPolicy: cdk.RemovalPolicy.DESTROY, deletionProtection: false, pointInTimeRecoveryEnabled: false },
  queue: { maxReceiveCount: 5, visibilityTimeoutSeconds: 60, retentionDays: 4, removalPolicy: cdk.RemovalPolicy.DESTROY },
  lambda: { memorySizeMiB: 256, timeoutSeconds: 30, logRetention: logs.RetentionDays.THREE_DAYS, logRemovalPolicy: cdk.RemovalPolicy.DESTROY, fanoutLeaseSeconds: 120 },
  api: { name: 'blog-publicacion-http-api', integrationTimeoutMilliseconds: 29_000, logRetention: logs.RetentionDays.THREE_DAYS, logRemovalPolicy: cdk.RemovalPolicy.DESTROY }
} as const;

/** Catálogo único de las rutas HTTP y de sus permisos mínimos. */
export const BLOG_HTTP_OPERATIONS: readonly BlogOperation[] = [
  { id: 'CreatePost', directory: 'lambda_create_post', route: { method: 'POST', path: '/authors/{authorId}/posts' }, dynamoActions: ['dynamodb:PutItem'] },
  { id: 'GetPosts', directory: 'lambda_get_posts', route: { method: 'GET', path: '/authors/{authorId}/posts' }, dynamoActions: ['dynamodb:Query'] },
  { id: 'GetDrafts', directory: 'lambda_get_posts', route: { method: 'GET', path: '/authors/{authorId}/drafts' }, dynamoActions: ['dynamodb:Query'], includeDraftIndex: true }
];
