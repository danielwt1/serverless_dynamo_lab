import * as cdk from 'aws-cdk-lib';
import * as sqs from 'aws-cdk-lib/aws-sqs';
import { Construct } from 'constructs';
import { BLOG_CONFIGURATION } from '../config/blog-config';

/** Colas de trabajo con una DLQ propia para cada tipo de mensaje. */
export class BlogQueues extends Construct {
  public readonly fanout: sqs.Queue;
  public readonly fanoutDlq: sqs.Queue;
  public readonly notifications: sqs.Queue;
  public readonly notificationsDlq: sqs.Queue;

  public constructor(scope: Construct, id: string) {
    super(scope, id);
    this.fanoutDlq = this.createDlq('FanoutDlq');
    this.notificationsDlq = this.createDlq('NotificationsDlq');
    this.fanout = this.createWorkQueue('FanoutJobs', this.fanoutDlq);
    this.notifications = this.createWorkQueue('NotificationJobs', this.notificationsDlq);
  }
  private createDlq(id: string): sqs.Queue {
    return new sqs.Queue(this, id, { retentionPeriod: cdk.Duration.days(BLOG_CONFIGURATION.queue.retentionDays), removalPolicy: BLOG_CONFIGURATION.queue.removalPolicy });
  }
  private createWorkQueue(id: string, deadLetterQueue: sqs.IQueue): sqs.Queue {
    return new sqs.Queue(this, id, {
      visibilityTimeout: cdk.Duration.seconds(BLOG_CONFIGURATION.queue.visibilityTimeoutSeconds), retentionPeriod: cdk.Duration.days(BLOG_CONFIGURATION.queue.retentionDays),
      deadLetterQueue: { queue: deadLetterQueue, maxReceiveCount: BLOG_CONFIGURATION.queue.maxReceiveCount }, removalPolicy: BLOG_CONFIGURATION.queue.removalPolicy
    });
  }
}
