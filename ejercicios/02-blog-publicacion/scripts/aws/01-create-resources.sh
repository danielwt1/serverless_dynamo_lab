#!/usr/bin/env bash
source "$(cd -- "$(dirname -- "$0")" && pwd)/_common.sh"

preflight
confirm "crear tablas, colas, ECR y roles del laboratorio Blog"
if ! aws_cmd dynamodb describe-table --table-name "$POSTS_TABLE" >/dev/null 2>&1; then
  aws_cmd dynamodb create-table --table-name "$POSTS_TABLE" \
    --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S AttributeName=GSI1PK,AttributeType=S AttributeName=GSI1SK,AttributeType=S \
    --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE --billing-mode PAY_PER_REQUEST \
    --global-secondary-indexes '[{"IndexName":"draft_gsi","KeySchema":[{"AttributeName":"GSI1PK","KeyType":"HASH"},{"AttributeName":"GSI1SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
    --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES >/dev/null
  aws_cmd dynamodb wait table-exists --table-name "$POSTS_TABLE"
fi
if ! aws_cmd dynamodb describe-table --table-name "$PROGRESS_TABLE" >/dev/null 2>&1; then
  aws_cmd dynamodb create-table --table-name "$PROGRESS_TABLE" --attribute-definitions AttributeName=PK,AttributeType=S AttributeName=SK,AttributeType=S --key-schema AttributeName=PK,KeyType=HASH AttributeName=SK,KeyType=RANGE --billing-mode PAY_PER_REQUEST >/dev/null
  aws_cmd dynamodb wait table-exists --table-name "$PROGRESS_TABLE"
fi
for queue in fanout-jobs-dlq notification-jobs-dlq; do aws_cmd sqs get-queue-url --queue-name "$queue" >/dev/null 2>&1 || aws_cmd sqs create-queue --queue-name "$queue" >/dev/null; done
for queue in fanout-jobs notification-jobs; do
  if ! aws_cmd sqs get-queue-url --queue-name "$queue" >/dev/null 2>&1; then
    dlq_url="$(aws_cmd sqs get-queue-url --queue-name "$queue-dlq" --query QueueUrl --output text)"; dlq_arn="$(aws_cmd sqs get-queue-attributes --queue-url "$dlq_url" --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"
    aws_cmd sqs create-queue --queue-name "$queue" --attributes "VisibilityTimeout=60,RedrivePolicy={\"deadLetterTargetArn\":\"$dlq_arn\",\"maxReceiveCount\":\"5\"}" >/dev/null
  fi
done
for repository in blog-create-post blog-get-posts blog-fanout-starter blog-follower-worker; do ensure_ecr "$repository"; done

POSTS_ARN="$(aws_cmd dynamodb describe-table --table-name "$POSTS_TABLE" --query 'Table.TableArn' --output text)"; PROGRESS_ARN="$(aws_cmd dynamodb describe-table --table-name "$PROGRESS_TABLE" --query 'Table.TableArn' --output text)"
FANOUT_URL="$(aws_cmd sqs get-queue-url --queue-name fanout-jobs --query QueueUrl --output text)"; NOTIFICATION_URL="$(aws_cmd sqs get-queue-url --queue-name notification-jobs --query QueueUrl --output text)"
FANOUT_ARN="$(aws_cmd sqs get-queue-attributes --queue-url "$FANOUT_URL" --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"; NOTIFICATION_ARN="$(aws_cmd sqs get-queue-attributes --queue-url "$NOTIFICATION_URL" --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)"
ensure_role blog-create-post-role "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"dynamodb:PutItem\"],\"Resource\":\"$POSTS_ARN\"}]}"
ensure_role blog-get-posts-role "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"dynamodb:Query\"],\"Resource\":[\"$POSTS_ARN\",\"$POSTS_ARN/index/draft_gsi\"]}]}"
ensure_role blog-fanout-starter-role "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"dynamodb:PutItem\",\"dynamodb:UpdateItem\"],\"Resource\":\"$PROGRESS_ARN\"},{\"Effect\":\"Allow\",\"Action\":[\"sqs:SendMessage\"],\"Resource\":\"$FANOUT_ARN\"}]}"
ensure_role blog-follower-worker-role "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":[\"dynamodb:Query\"],\"Resource\":\"$POSTS_ARN\"},{\"Effect\":\"Allow\",\"Action\":[\"dynamodb:TransactWriteItems\"],\"Resource\":\"$PROGRESS_ARN\"},{\"Effect\":\"Allow\",\"Action\":[\"sqs:ReceiveMessage\",\"sqs:DeleteMessage\",\"sqs:GetQueueAttributes\",\"sqs:SendMessage\"],\"Resource\":[\"$FANOUT_ARN\",\"$NOTIFICATION_ARN\"]}]}"
echo "Recursos base listos. El Stream se crea pero no se conecta aún: falta implementar DRAFT -> PUBLISHED."
