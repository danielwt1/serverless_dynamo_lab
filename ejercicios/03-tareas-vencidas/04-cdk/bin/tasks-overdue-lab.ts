#!/usr/bin/env node
import * as cdk from 'aws-cdk-lib';
import { TasksOverdueLabStack } from '../lib/tasks-overdue-lab-stack';

const app = new cdk.App();
new TasksOverdueLabStack(app, 'TasksOverdueLabStack');
