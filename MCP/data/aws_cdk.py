"""AWS CDK v2 construct specs for the MCP server.

Covers the AWS services most relevant to Polar Gosling, documented in a
gRPC + protobuf style matching the Yandex Cloud module pattern.
Source: https://docs.aws.amazon.com/cdk/api/v2/docs
"""

from typing import Any

AWS_CDK_OVERVIEW: dict[str, Any] = {
    "description": (
        "AWS CDK v2 construct reference for services used by Polar Gosling. "
        "All constructs follow the standard CDK L2 pattern: typed props, "
        "grant helpers for IAM, CloudWatch metric methods, and removal-policy support. "
        "Source: https://docs.aws.amazon.com/cdk/api/v2/docs"
    ),
    "common_patterns": {
        "construct_format": "new <Construct>(scope, id, props)",
        "auth": "IAM-based — use grant*() helpers or addToRolePolicy()",
        "removal_policy": (
            "applyRemovalPolicy(RemovalPolicy.RETAIN | DESTROY | SNAPSHOT). "
            "Controls what happens when the resource is removed from the stack."
        ),
        "metrics": "metric*() methods return aws_cloudwatch.Metric objects for CloudWatch dashboards and alarms",
        "import_pattern": "fromXxxArn() / fromXxxName() / fromXxxAttributes() — import existing resources",
        "tagging": "Tags.of(construct).add(key, value) — all resources support tagging via aspects",
    },
    "package": "aws-cdk-lib",
    "docs_base_url": "https://docs.aws.amazon.com/cdk/api/v2/docs",
    "protobuf_style_note": (
        "This reference documents AWS CDK constructs in a gRPC/protobuf-inspired format "
        "for consistency with the Yandex Cloud module. 'rpcs' map to construct methods, "
        "'key_messages' map to construct props and output types."
    ),
}


AWS_CDK_SQS: dict[str, Any] = {
    "service_name": "Amazon SQS",
    "package": "aws-cdk-lib.aws_sqs",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_sqs-readme.html",
    "description": (
        "Manages SQS queues — standard and FIFO. Used by Polar Gosling as the Celery "
        "message broker (CELERY_BROKER_URL) for task distribution between MotherGoose and UglyFox."
    ),
    "services": {
        "Queue": {
            "description": "L2 construct for creating and managing SQS queues.",
            "rpcs": {
                "constructor": {
                    "signature": "Queue(scope, id, props?)",
                    "description": "Creates a new SQS queue.",
                },
                "grantSendMessages": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant the given principal permissions to send messages to this queue.",
                },
                "grantConsumeMessages": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant the given principal permissions to consume messages from this queue.",
                },
                "grantPurge": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant the given principal permissions to purge all messages from this queue.",
                },
                "addToResourcePolicy": {
                    "params": "statement: PolicyStatement",
                    "returns": "AddToResourcePolicyResult",
                    "description": "Adds a statement to the IAM resource policy associated with this queue.",
                },
                "metricApproximateNumberOfMessagesVisible": {
                    "returns": "Metric",
                    "description": "Approximate number of messages available for retrieval from the queue.",
                },
                "metricApproximateAgeOfOldestMessage": {
                    "returns": "Metric",
                    "description": "Age of the oldest non-deleted message in the queue.",
                },
                "metricNumberOfMessagesSent": {
                    "returns": "Metric",
                    "description": "Number of messages added to the queue.",
                },
                "metricNumberOfMessagesReceived": {
                    "returns": "Metric",
                    "description": "Number of messages returned by calls to ReceiveMessage.",
                },
                "metricSentMessageSize": {
                    "returns": "Metric",
                    "description": "Size of messages added to the queue.",
                },
                "fromQueueArn": {
                    "params": "scope, id, queueArn: string",
                    "returns": "IQueue",
                    "description": "Import an existing queue by ARN.",
                    "static": True,
                },
                "fromQueueAttributes": {
                    "params": "scope, id, attrs: QueueAttributes",
                    "returns": "IQueue",
                    "description": "Import an existing queue by attributes.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "QueueProps": {
            "fields": [
                "queueName (string — physical queue name)",
                "fifo (boolean — FIFO queue, default false)",
                "contentBasedDeduplication (boolean — for FIFO queues)",
                "visibilityTimeout (Duration — default 30s)",
                "retentionPeriod (Duration — default 4 days, max 14 days)",
                "deliveryDelay (Duration — default 0s)",
                "receiveMessageWaitTime (Duration — long-poll wait, default 0s)",
                "deadLetterQueue (DeadLetterQueue — { queue: IQueue, maxReceiveCount: number })",
                "encryption (QueueEncryption — UNENCRYPTED | KMS_MANAGED | KMS | SQS_MANAGED)",
                "encryptionMasterKey (IKey — custom KMS key)",
                "dataKeyReuse (Duration — KMS data key reuse period)",
                "removalPolicy (RemovalPolicy — RETAIN | DESTROY)",
                "enforceSSL (boolean — require HTTPS for queue access)",
            ],
        },
        "Queue_outputs": {
            "fields": [
                "queueArn (string)", "queueUrl (string)", "queueName (string)",
                "fifo (boolean)", "encryptionMasterKey (IKey, optional)",
                "encryptionType (QueueEncryption)",
            ],
        },
        "DeadLetterQueue": {
            "fields": [
                "queue (IQueue — the DLQ)",
                "maxReceiveCount (number — messages moved to DLQ after this many failed receives)",
            ],
        },
    },
    "polar_gosling_usage": (
        "SQS is the Celery broker on AWS. MotherGoose enqueues tasks (git_sync, runners, webhooks, "
        "maintenance) and UglyFox consumes lifecycle tasks (health, pruning, pool management). "
        "Configured via CELERY_BROKER_URL env var pointing to the SQS queue URL."
    ),
}


AWS_CDK_DYNAMODB: dict[str, Any] = {
    "service_name": "Amazon DynamoDB",
    "package": "aws-cdk-lib.aws_dynamodb",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_dynamodb-readme.html",
    "description": (
        "Manages DynamoDB tables — the AWS database backend for Polar Gosling. "
        "Stores runners, egg_configs, sync_history, deployment_plans, audit_logs, "
        "tofu_versions, and gosling_version tables."
    ),
    "services": {
        "Table": {
            "description": "L2 construct for DynamoDB tables. Use TableV2 for global tables with replicas.",
            "rpcs": {
                "constructor": {
                    "signature": "Table(scope, id, props)",
                    "description": "Creates a new DynamoDB table.",
                },
                "addGlobalSecondaryIndex": {
                    "params": "props: GlobalSecondaryIndexProps",
                    "description": "Add a global secondary index to the table.",
                },
                "addLocalSecondaryIndex": {
                    "params": "props: LocalSecondaryIndexProps",
                    "description": "Add a local secondary index to the table.",
                },
                "grantReadData": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Permits BatchGetItem, GetRecords, GetShardIterator, Query, GetItem, Scan, DescribeTable.",
                },
                "grantWriteData": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Permits BatchWriteItem, PutItem, UpdateItem, DeleteItem, DescribeTable.",
                },
                "grantReadWriteData": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Permits all read and write data operations.",
                },
                "grantFullAccess": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Permits all DynamoDB operations (dynamodb:*).",
                },
                "grantStreamRead": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Permits DescribeStream, GetRecords, GetShardIterator, ListStreams on the table stream.",
                },
                "autoScaleReadCapacity": {
                    "params": "props: EnableScalingProps",
                    "returns": "IScalableTableAttribute",
                    "description": "Enable read capacity auto-scaling (provisioned mode only).",
                },
                "autoScaleWriteCapacity": {
                    "params": "props: EnableScalingProps",
                    "returns": "IScalableTableAttribute",
                    "description": "Enable write capacity auto-scaling (provisioned mode only).",
                },
                "metricConsumedReadCapacityUnits": {
                    "returns": "Metric",
                    "description": "Consumed read capacity units (sum over 5 min).",
                },
                "metricConsumedWriteCapacityUnits": {
                    "returns": "Metric",
                    "description": "Consumed write capacity units (sum over 5 min).",
                },
                "metricSuccessfulRequestLatency": {
                    "returns": "Metric",
                    "description": "Successful request latency (average over 5 min).",
                },
                "metricThrottledRequestsForOperations": {
                    "returns": "IMetric",
                    "description": "Throttled requests summed across all operations.",
                },
                "fromTableArn": {
                    "params": "scope, id, tableArn: string",
                    "returns": "ITable",
                    "description": "Import an existing table by ARN.",
                    "static": True,
                },
                "fromTableName": {
                    "params": "scope, id, tableName: string",
                    "returns": "ITable",
                    "description": "Import an existing table by name.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "TableProps": {
            "fields": [
                "partitionKey (Attribute — { name: string, type: AttributeType })",
                "sortKey (Attribute, optional)",
                "tableName (string, optional — physical table name)",
                "billingMode (BillingMode — PROVISIONED | PAY_PER_REQUEST)",
                "readCapacity (number — default 5, provisioned mode only)",
                "writeCapacity (number — default 5, provisioned mode only)",
                "stream (StreamViewType — NEW_IMAGE | OLD_IMAGE | NEW_AND_OLD_IMAGES | KEYS_ONLY)",
                "encryption (TableEncryption — DEFAULT | AWS_MANAGED | CUSTOMER_MANAGED)",
                "encryptionKey (IKey — custom KMS key for CUSTOMER_MANAGED)",
                "timeToLiveAttribute (string — TTL attribute name)",
                "pointInTimeRecoverySpecification (PointInTimeRecoverySpecification)",
                "removalPolicy (RemovalPolicy — default RETAIN)",
                "replicationRegions (string[] — global table replica regions)",
                "tableClass (TableClass — STANDARD | STANDARD_INFREQUENT_ACCESS)",
                "deletionProtection (boolean — default false)",
                "kinesisStream (IStream — Kinesis Data Stream for CDC)",
                "importSource (ImportSourceSpecification — S3 import)",
            ],
        },
        "Table_outputs": {
            "fields": [
                "tableArn (string)", "tableName (string)", "tableStreamArn (string, optional)",
                "encryptionKey (IKey, optional)", "hasIndex (boolean)",
            ],
        },
        "AttributeType_enum": ["STRING", "NUMBER", "BINARY"],
        "BillingMode_enum": ["PROVISIONED", "PAY_PER_REQUEST"],
        "StreamViewType_enum": ["NEW_IMAGE", "OLD_IMAGE", "NEW_AND_OLD_IMAGES", "KEYS_ONLY"],
    },
    "polar_gosling_usage": (
        "DynamoDB is the AWS database backend. Tables: runners, egg_configs, sync_history, "
        "deployment_plans, audit_logs, tofu_versions, gosling_version. "
        "Accessed via boto3/aioboto3 DynamoDB client in MotherGoose and UglyFox services."
    ),
}


AWS_CDK_S3: dict[str, Any] = {
    "service_name": "Amazon S3",
    "package": "aws-cdk-lib.aws_s3",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_s3-readme.html",
    "description": (
        "Manages S3 buckets for object storage. Used by Polar Gosling for Rift image cache, "
        "Gosling/OpenTofu binary storage, and S3 artifact cache in MotherGoose."
    ),
    "services": {
        "Bucket": {
            "description": "L2 construct for S3 buckets with lifecycle rules, encryption, CORS, and event notifications.",
            "rpcs": {
                "constructor": {
                    "signature": "Bucket(scope, id, props?)",
                    "description": "Creates a new S3 bucket.",
                },
                "grantRead": {
                    "params": "identity: IGrantable, objectsKeyPattern?: any",
                    "returns": "Grant",
                    "description": "Grant read permissions (s3:GetObject*, s3:GetBucket*, s3:List*).",
                },
                "grantWrite": {
                    "params": "identity: IGrantable, objectsKeyPattern?: any, allowedActionPatterns?: string[]",
                    "returns": "Grant",
                    "description": "Grant write permissions (s3:PutObject*, s3:Abort*, s3:DeleteObject*).",
                },
                "grantReadWrite": {
                    "params": "identity: IGrantable, objectsKeyPattern?: any",
                    "returns": "Grant",
                    "description": "Grant read + write permissions.",
                },
                "grantPut": {
                    "params": "identity: IGrantable, objectsKeyPattern?: any",
                    "returns": "Grant",
                    "description": "Grant s3:PutObject and s3:PutObjectTagging.",
                },
                "grantDelete": {
                    "params": "identity: IGrantable, objectsKeyPattern?: any",
                    "returns": "Grant",
                    "description": "Grant s3:DeleteObject*.",
                },
                "grantPublicAccess": {
                    "params": "keyPrefix?: string, ...allowedActions: string[]",
                    "returns": "Grant",
                    "description": "Allow public access to specific objects.",
                },
                "addEventNotification": {
                    "params": "event: EventType, dest: IBucketNotificationDestination, ...filters: NotificationKeyFilter[]",
                    "description": "Add a notification for S3 events (e.g., OBJECT_CREATED, OBJECT_REMOVED).",
                },
                "addObjectCreatedNotification": {
                    "params": "dest: IBucketNotificationDestination, ...filters: NotificationKeyFilter[]",
                    "description": "Shorthand for OBJECT_CREATED event notification.",
                },
                "addObjectRemovedNotification": {
                    "params": "dest: IBucketNotificationDestination, ...filters: NotificationKeyFilter[]",
                    "description": "Shorthand for OBJECT_REMOVED event notification.",
                },
                "addLifecycleRule": {
                    "params": "rule: LifecycleRule",
                    "description": "Add a lifecycle rule (expiration, transitions, abort incomplete multipart).",
                },
                "addCorsRule": {
                    "params": "rule: CorsRule",
                    "description": "Add a CORS configuration rule.",
                },
                "addToResourcePolicy": {
                    "params": "permission: PolicyStatement",
                    "returns": "AddToResourcePolicyResult",
                    "description": "Add a statement to the bucket policy.",
                },
                "transferAccelerationUrlForObject": {
                    "params": "key?: string, options?: TransferAccelerationUrlOptions",
                    "returns": "string",
                    "description": "URL for transfer-accelerated access to an object.",
                },
                "arnForObjects": {
                    "params": "keyPattern: string",
                    "returns": "string",
                    "description": "ARN for objects matching the pattern.",
                },
                "fromBucketArn": {
                    "params": "scope, id, bucketArn: string",
                    "returns": "IBucket",
                    "description": "Import an existing bucket by ARN.",
                    "static": True,
                },
                "fromBucketName": {
                    "params": "scope, id, bucketName: string",
                    "returns": "IBucket",
                    "description": "Import an existing bucket by name.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "BucketProps": {
            "fields": [
                "bucketName (string, optional — physical bucket name)",
                "versioned (boolean — enable versioning, default false)",
                "encryption (BucketEncryption — UNENCRYPTED | S3_MANAGED | KMS_MANAGED | KMS)",
                "encryptionKey (IKey — custom KMS key for KMS encryption)",
                "blockPublicAccess (BlockPublicAccess — BLOCK_ALL | BLOCK_ACLS | etc.)",
                "publicReadAccess (boolean — default false)",
                "removalPolicy (RemovalPolicy — default RETAIN)",
                "autoDeleteObjects (boolean — auto-delete on stack removal, default false)",
                "lifecycleRules (LifecycleRule[] — expiration, transitions)",
                "cors (CorsRule[] — CORS configuration)",
                "serverAccessLogsBucket (IBucket — access logging target)",
                "serverAccessLogsPrefix (string — log prefix)",
                "enforceSSL (boolean — require HTTPS)",
                "minimumTLSVersion (number — minimum TLS version)",
                "objectOwnership (ObjectOwnership — BUCKET_OWNER_ENFORCED | BUCKET_OWNER_PREFERRED | OBJECT_WRITER)",
                "transferAcceleration (boolean — enable transfer acceleration)",
                "eventBridgeEnabled (boolean — send events to EventBridge)",
                "intelligentTieringConfigurations (IntelligentTieringConfiguration[])",
                "inventories (Inventory[] — S3 inventory configuration)",
            ],
        },
        "Bucket_outputs": {
            "fields": [
                "bucketArn (string)", "bucketName (string)",
                "bucketWebsiteUrl (string)", "bucketDomainName (string)",
                "bucketRegionalDomainName (string)", "bucketDualStackDomainName (string)",
                "encryptionKey (IKey, optional)",
            ],
        },
        "EventType_enum": [
            "OBJECT_CREATED", "OBJECT_CREATED_PUT", "OBJECT_CREATED_POST",
            "OBJECT_CREATED_COPY", "OBJECT_CREATED_COMPLETE_MULTIPART_UPLOAD",
            "OBJECT_REMOVED", "OBJECT_REMOVED_DELETE", "OBJECT_REMOVED_DELETE_MARKER_CREATED",
            "OBJECT_RESTORE_POST", "OBJECT_RESTORE_COMPLETED", "OBJECT_RESTORE_DELETE",
        ],
    },
    "polar_gosling_usage": (
        "S3 stores Gosling and OpenTofu binaries, Rift image cache tarballs, "
        "and serves as the S3 artifact cache for MotherGoose. "
        "Accessed via boto3/aioboto3 S3 client."
    ),
}


AWS_CDK_LAMBDA: dict[str, Any] = {
    "service_name": "AWS Lambda",
    "package": "aws-cdk-lib.aws_lambda",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_lambda-readme.html",
    "description": (
        "Manages Lambda functions — serverless compute for event-driven workloads. "
        "Can be used as EventBridge targets, SQS consumers, API Gateway integrations, "
        "and S3 event handlers in the Polar Gosling infrastructure."
    ),
    "services": {
        "Function": {
            "description": "L2 construct for Lambda functions with automatic IAM role, logging, and event source support.",
            "rpcs": {
                "constructor": {
                    "signature": "Function(scope, id, props)",
                    "description": "Creates a new Lambda function.",
                    "required_props": ["code", "handler", "runtime"],
                },
                "addEventSource": {
                    "params": "source: IEventSource",
                    "description": "Add an event source (SQS, DynamoDB Streams, Kinesis, S3, etc.).",
                },
                "addEventSourceMapping": {
                    "params": "id: string, options: EventSourceMappingOptions",
                    "returns": "EventSourceMapping",
                    "description": "Add an event source mapping (lower-level than addEventSource).",
                },
                "addEnvironment": {
                    "params": "key: string, value: string, options?: EnvironmentOptions",
                    "returns": "Function",
                    "description": "Add an environment variable.",
                },
                "addLayers": {
                    "params": "...layers: ILayerVersion[]",
                    "description": "Add Lambda layers to the function.",
                },
                "addPermission": {
                    "params": "id: string, permission: Permission",
                    "description": "Add a resource-based policy statement (e.g., allow API Gateway to invoke).",
                },
                "addFunctionUrl": {
                    "params": "options?: FunctionUrlOptions",
                    "returns": "FunctionUrl",
                    "description": "Add a Lambda Function URL for direct HTTPS invocation.",
                },
                "addAlias": {
                    "params": "aliasName: string, options?: AliasOptions",
                    "returns": "Alias",
                    "description": "Create an alias pointing to currentVersion.",
                },
                "addToRolePolicy": {
                    "params": "statement: PolicyStatement",
                    "description": "Add a statement to the function's execution role.",
                },
                "grantInvoke": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant lambda:InvokeFunction to the given principal.",
                },
                "grantInvokeUrl": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant permissions to invoke the function URL.",
                },
                "configureAsyncInvoke": {
                    "params": "options: EventInvokeConfigOptions",
                    "description": "Configure async invocation (maxEventAge, retryAttempts, onSuccess, onFailure).",
                },
                "metricInvocations": {
                    "returns": "Metric",
                    "description": "How often this Lambda is invoked (sum over 5 min).",
                },
                "metricErrors": {
                    "returns": "Metric",
                    "description": "How many invocations fail (sum over 5 min).",
                },
                "metricDuration": {
                    "returns": "Metric",
                    "description": "Execution duration (average over 5 min).",
                },
                "metricThrottles": {
                    "returns": "Metric",
                    "description": "How often this Lambda is throttled (sum over 5 min).",
                },
                "fromFunctionArn": {
                    "params": "scope, id, functionArn: string",
                    "returns": "IFunction",
                    "description": "Import an existing function by ARN.",
                    "static": True,
                },
                "fromFunctionName": {
                    "params": "scope, id, functionName: string",
                    "returns": "IFunction",
                    "description": "Import an existing function by name.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "FunctionProps": {
            "required": [
                "code (Code — Code.fromAsset(), Code.fromBucket(), Code.fromInline(), Code.fromDockerBuild())",
                "handler (string — e.g., 'index.handler', use Handler.FROM_IMAGE for containers)",
                "runtime (Runtime — PYTHON_3_12, NODEJS_20_X, JAVA_21, DOTNET_8, FROM_IMAGE, etc.)",
            ],
            "optional": [
                "functionName (string)", "description (string)",
                "memorySize (number — MB, default 128)", "timeout (Duration — default 3s)",
                "ephemeralStorageSize (Size — /tmp size, default 512 MiB)",
                "environment (map<string, string>)", "environmentEncryption (IKey)",
                "architecture (Architecture — X86_64 | ARM_64)",
                "layers (ILayerVersion[])", "vpc (IVpc)", "vpcSubnets (SubnetSelection)",
                "securityGroups (ISecurityGroup[])",
                "role (IRole — execution role, auto-created if omitted)",
                "deadLetterQueue (IQueue)", "deadLetterQueueEnabled (boolean)",
                "deadLetterTopic (ITopic)",
                "events (IEventSource[] — event sources)",
                "tracing (Tracing — DISABLED | ACTIVE | PASS_THROUGH)",
                "reservedConcurrentExecutions (number)",
                "retryAttempts (number — 0-2, default 2)",
                "maxEventAge (Duration — 60s to 6h, default 6h)",
                "onFailure (IDestination)", "onSuccess (IDestination)",
                "loggingFormat (LoggingFormat — TEXT | JSON)",
                "logGroup (ILogGroup)", "snapStart (SnapStartConf)",
                "recursiveLoop (RecursiveLoop — Terminate | Allow)",
            ],
        },
        "Function_outputs": {
            "fields": [
                "functionArn (string)", "functionName (string)",
                "role (IRole, optional)", "currentVersion (Version)",
                "latestVersion (IVersion)", "logGroup (ILogGroup)",
                "runtime (Runtime)", "architecture (Architecture)",
                "isBoundToVpc (boolean)", "deadLetterQueue (IQueue, optional)",
                "timeout (Duration, optional)",
            ],
        },
        "Runtime_enum": [
            "PYTHON_3_12", "PYTHON_3_13", "NODEJS_20_X", "NODEJS_22_X",
            "JAVA_21", "DOTNET_8", "GO_1_X (deprecated)", "PROVIDED_AL2023",
            "FROM_IMAGE (container image)",
        ],
    },
    "event_sources_note": (
        "Event sources from aws-cdk-lib/aws-lambda-event-sources: "
        "SqsEventSource, DynamoEventSource, KinesisEventSource, S3EventSource, "
        "ManagedKafkaEventSource, SelfManagedKafkaEventSource."
    ),
}


AWS_CDK_ECS_FARGATE: dict[str, Any] = {
    "service_name": "Amazon ECS (Fargate)",
    "package": "aws-cdk-lib.aws_ecs",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_ecs-readme.html",
    "description": (
        "Manages ECS clusters, task definitions, and Fargate services. "
        "Used by Polar Gosling for deploying SERVERLESS runner type on AWS via ECS Fargate."
    ),
    "services": {
        "Cluster": {
            "description": "ECS cluster — logical grouping of tasks and services.",
            "rpcs": {
                "constructor": {
                    "signature": "Cluster(scope, id, props?)",
                    "description": "Creates a new ECS cluster.",
                },
                "addCapacity": {
                    "params": "id: string, options: AddCapacityOptions",
                    "returns": "AutoScalingGroup",
                    "description": "Add EC2 capacity (Auto Scaling Group) to the cluster.",
                },
                "enableFargateCapacityProviders": {
                    "description": "Enable Fargate and Fargate Spot capacity providers.",
                },
                "metricCpuReservation": {
                    "returns": "Metric",
                    "description": "CPU reservation metric for the cluster.",
                },
                "metricMemoryReservation": {
                    "returns": "Metric",
                    "description": "Memory reservation metric for the cluster.",
                },
                "fromClusterArn": {
                    "params": "scope, id, clusterArn: string",
                    "returns": "ICluster",
                    "description": "Import an existing cluster by ARN.",
                    "static": True,
                },
                "fromClusterAttributes": {
                    "params": "scope, id, attrs: ClusterAttributes",
                    "returns": "ICluster",
                    "description": "Import an existing cluster by attributes.",
                    "static": True,
                },
            },
        },
        "FargateTaskDefinition": {
            "description": "Task definition for Fargate launch type — defines containers, resources, IAM roles.",
            "rpcs": {
                "constructor": {
                    "signature": "FargateTaskDefinition(scope, id, props?)",
                    "description": "Creates a Fargate task definition.",
                },
                "addContainer": {
                    "params": "id: string, props: ContainerDefinitionOptions",
                    "returns": "ContainerDefinition",
                    "description": "Add a container to the task definition.",
                },
                "addVolume": {
                    "params": "volume: Volume",
                    "description": "Add a volume to the task definition.",
                },
            },
        },
        "FargateService": {
            "description": "L2 construct for ECS Fargate services — manages desired count, networking, load balancing.",
            "rpcs": {
                "constructor": {
                    "signature": "FargateService(scope, id, props)",
                    "description": "Creates a Fargate service.",
                    "required_props": ["cluster", "taskDefinition"],
                },
                "autoScaleTaskCount": {
                    "params": "props: EnableScalingProps",
                    "returns": "ScalableTaskCount",
                    "description": "Configure auto-scaling for the service task count.",
                },
                "enableCloudMap": {
                    "params": "options: CloudMapOptions",
                    "returns": "Service",
                    "description": "Enable Cloud Map service discovery.",
                },
                "enableServiceConnect": {
                    "params": "config?: ServiceConnectProps",
                    "description": "Enable ECS Service Connect.",
                },
                "enableDeploymentAlarms": {
                    "params": "alarmNames: string[], options?: DeploymentAlarmOptions",
                    "description": "Configure CloudWatch alarms to monitor deployments.",
                },
                "attachToApplicationTargetGroup": {
                    "params": "targetGroup: IApplicationTargetGroup",
                    "returns": "LoadBalancerTargetProps",
                    "description": "Attach to an ALB target group.",
                },
                "attachToNetworkTargetGroup": {
                    "params": "targetGroup: INetworkTargetGroup",
                    "returns": "LoadBalancerTargetProps",
                    "description": "Attach to an NLB target group.",
                },
                "loadBalancerTarget": {
                    "params": "options: LoadBalancerTargetOptions",
                    "returns": "IEcsLoadBalancerTarget",
                    "description": "Return a load balancing target for a specific container and port.",
                },
                "metricCpuUtilization": {
                    "returns": "Metric",
                    "description": "CPU utilization metric for the service.",
                },
                "metricMemoryUtilization": {
                    "returns": "Metric",
                    "description": "Memory utilization metric for the service.",
                },
                "fromFargateServiceArn": {
                    "params": "scope, id, fargateServiceArn: string",
                    "returns": "IFargateService",
                    "description": "Import an existing Fargate service by ARN.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "ClusterProps": {
            "fields": [
                "clusterName (string, optional)", "vpc (IVpc, optional — auto-created if omitted)",
                "defaultCloudMapNamespace (CloudMapNamespaceOptions, optional)",
                "containerInsightsV2 (ContainerInsights — ENABLED | ENHANCED | DISABLED)",
                "enableFargateCapacityProviders (boolean — default false)",
                "executeCommandConfiguration (ExecuteCommandConfiguration, optional)",
            ],
        },
        "FargateTaskDefinitionProps": {
            "fields": [
                "cpu (number — 256, 512, 1024, 2048, 4096, 8192, 16384)",
                "memoryLimitMiB (number — depends on cpu: 512-30720 for 4096 cpu)",
                "executionRole (IRole — role for ECS agent, auto-created)",
                "taskRole (IRole — role for the containers, auto-created)",
                "family (string — task definition family name)",
                "runtimePlatform (RuntimePlatform — { cpuArchitecture, operatingSystemFamily })",
                "ephemeralStorageGiB (number — 21-200 GiB, default 20)",
                "volumes (Volume[])",
            ],
        },
        "FargateServiceProps": {
            "required": ["cluster (ICluster)", "taskDefinition (TaskDefinition)"],
            "optional": [
                "serviceName (string)", "desiredCount (number — default 1)",
                "assignPublicIp (boolean — default false)",
                "securityGroups (ISecurityGroup[])", "vpcSubnets (SubnetSelection)",
                "platformVersion (FargatePlatformVersion — default LATEST)",
                "circuitBreaker (DeploymentCircuitBreaker)",
                "minHealthyPercent (number — default 50)", "maxHealthyPercent (number — default 200)",
                "capacityProviderStrategies (CapacityProviderStrategy[])",
                "enableExecuteCommand (boolean)", "propagateTags (PropagatedTagSource)",
                "cloudMapOptions (CloudMapOptions)", "serviceConnectConfiguration (ServiceConnectProps)",
                "deploymentController (DeploymentController)",
                "healthCheckGracePeriod (Duration)",
                "volumeConfigurations (ServiceManagedVolume[])",
            ],
        },
        "FargateService_outputs": {
            "fields": [
                "serviceArn (string)", "serviceName (string)",
                "cluster (ICluster)", "taskDefinition (TaskDefinition)",
                "connections (Connections)", "cloudMapService (IService, optional)",
            ],
        },
        "ContainerDefinitionOptions": {
            "fields": [
                "image (ContainerImage — ContainerImage.fromRegistry(), .fromAsset(), .fromEcrRepository())",
                "memoryLimitMiB (number)", "cpu (number)",
                "essential (boolean — default true)", "command (string[])",
                "environment (map<string, string>)", "secrets (map<string, Secret>)",
                "portMappings (PortMapping[])", "logging (LogDriver)",
                "healthCheck (HealthCheck)", "startTimeout (Duration)", "stopTimeout (Duration)",
            ],
        },
    },
    "polar_gosling_usage": (
        "ECS Fargate deploys SERVERLESS runner type on AWS. MotherGoose renders "
        "OpenTofu + Jinja2 templates that provision ECS task definitions and services "
        "via the Compute Module. Also used for MotherGoose and UglyFox containers themselves."
    ),
}


AWS_CDK_EVENTS: dict[str, Any] = {
    "service_name": "Amazon EventBridge",
    "package": "aws-cdk-lib.aws_events",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_events-readme.html",
    "description": (
        "Manages EventBridge rules, event buses, and targets. Used by Polar Gosling "
        "for scheduled triggers (e.g., 5-minute Git sync timer) and event-driven workflows."
    ),
    "services": {
        "Rule": {
            "description": "EventBridge rule — matches events by pattern or schedule and routes to targets.",
            "rpcs": {
                "constructor": {
                    "signature": "Rule(scope, id, props?)",
                    "description": "Creates an EventBridge rule.",
                },
                "addTarget": {
                    "params": "target?: IRuleTarget",
                    "description": "Add a target to the rule (Lambda, SQS, ECS, Step Functions, API Gateway, etc.).",
                },
                "addEventPattern": {
                    "params": "eventPattern: EventPattern",
                    "description": "Add an event pattern filter to the rule.",
                },
                "fromEventRuleArn": {
                    "params": "scope, id, eventRuleArn: string",
                    "returns": "IRule",
                    "description": "Import an existing rule by ARN.",
                    "static": True,
                },
            },
        },
        "EventBus": {
            "description": "Custom event bus for application-specific events.",
            "rpcs": {
                "constructor": {
                    "signature": "EventBus(scope, id, props?)",
                    "description": "Creates a custom event bus.",
                },
                "grantPutEventsTo": {
                    "params": "grantee: IGrantable",
                    "returns": "Grant",
                    "description": "Grant events:PutEvents to the given principal.",
                },
                "archive": {
                    "params": "id: string, props: BaseArchiveProps",
                    "returns": "Archive",
                    "description": "Create an archive for events on this bus.",
                },
                "fromEventBusArn": {
                    "params": "scope, id, eventBusArn: string",
                    "returns": "IEventBus",
                    "description": "Import an existing event bus by ARN.",
                    "static": True,
                },
                "fromEventBusName": {
                    "params": "scope, id, eventBusName: string",
                    "returns": "IEventBus",
                    "description": "Import an existing event bus by name.",
                    "static": True,
                },
            },
        },
        "Schedule": {
            "description": "Schedule expressions for EventBridge rules.",
            "rpcs": {
                "rate": {
                    "params": "duration: Duration",
                    "returns": "Schedule",
                    "description": "Rate-based schedule (e.g., Schedule.rate(Duration.minutes(5))).",
                    "static": True,
                },
                "cron": {
                    "params": "options: CronOptions",
                    "returns": "Schedule",
                    "description": "Cron-based schedule (e.g., Schedule.cron({ minute: '0', hour: '*/6' })).",
                    "static": True,
                },
                "expression": {
                    "params": "expression: string",
                    "returns": "Schedule",
                    "description": "Raw schedule expression string.",
                    "static": True,
                },
            },
        },
    },
    "key_messages": {
        "RuleProps": {
            "fields": [
                "ruleName (string, optional)", "description (string, optional)",
                "enabled (boolean — default true)",
                "schedule (Schedule — rate or cron expression)",
                "eventPattern (EventPattern — filter by source, detail-type, detail, etc.)",
                "eventBus (IEventBus — default: default event bus)",
                "targets (IRuleTarget[] — targets to invoke)",
            ],
        },
        "EventPattern": {
            "fields": [
                "source (string[] — e.g., ['aws.ec2', 'aws.ecs'])",
                "detailType (string[] — event detail type)",
                "detail (map — nested field matching)",
                "account (string[])", "region (string[])",
                "resources (string[] — ARN patterns)",
            ],
        },
        "CronOptions": {
            "fields": [
                "minute (string — default '*')", "hour (string — default '*')",
                "day (string — day of month, default '*')", "month (string — default '*')",
                "weekDay (string — day of week, default '?')", "year (string — default '*')",
            ],
        },
    },
    "targets_package": {
        "package": "aws-cdk-lib.aws_events_targets",
        "available_targets": [
            "LambdaFunction — invoke a Lambda function",
            "SqsQueue — send message to SQS queue",
            "EcsTask — run an ECS task",
            "ApiGateway — invoke API Gateway REST API",
            "ApiDestination — invoke an HTTP endpoint",
            "SfnStateMachine — start a Step Functions execution",
            "CodeBuildProject — start a CodeBuild build",
            "SnsTopic — publish to SNS topic",
            "KinesisStream — put record to Kinesis stream",
            "CloudWatchLogGroup — send to CloudWatch Logs",
        ],
    },
    "polar_gosling_usage": (
        "EventBridge provides the AWS equivalent of YC Timer triggers. "
        "A Schedule.rate(Duration.minutes(5)) rule triggers POST /internal/sync-git "
        "on MotherGoose for periodic Nest Git sync. Also used for event-driven "
        "runner lifecycle events."
    ),
}


AWS_CDK_API_GATEWAY: dict[str, Any] = {
    "service_name": "Amazon API Gateway",
    "package": "aws-cdk-lib.aws_apigateway / aws-cdk-lib.aws_apigatewayv2",
    "docs_url": "https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.aws_apigateway-readme.html",
    "description": (
        "Manages REST APIs (API Gateway v1) and HTTP APIs (API Gateway v2). "
        "Used by Polar Gosling for the MotherGoose REST API endpoint — "
        "receives webhooks, internal sync triggers, and serves the eggs/runners/binaries API."
    ),
    "services": {
        "RestApi": {
            "description": "L2 construct for API Gateway REST APIs (v1) — resource hierarchy, methods, stages, authorizers.",
            "rpcs": {
                "constructor": {
                    "signature": "RestApi(scope, id, props?)",
                    "description": "Creates a new REST API.",
                },
                "root_addResource": {
                    "signature": "api.root.addResource(pathPart, options?)",
                    "returns": "Resource",
                    "description": "Add a child resource to the API root (e.g., /eggs, /runners, /webhooks).",
                },
                "resource_addMethod": {
                    "signature": "resource.addMethod(httpMethod, integration?, options?)",
                    "returns": "Method",
                    "description": "Add an HTTP method (GET, POST, etc.) with an integration (Lambda, HTTP, Mock).",
                },
                "addGatewayResponse": {
                    "params": "id: string, options: GatewayResponseOptions",
                    "returns": "GatewayResponse",
                    "description": "Add a custom gateway response (4xx/5xx error templates).",
                },
                "addUsagePlan": {
                    "params": "id: string, props?: UsagePlanProps",
                    "returns": "UsagePlan",
                    "description": "Add a usage plan with throttling and quota limits.",
                },
                "addApiKey": {
                    "params": "id: string, options?: ApiKeyOptions",
                    "returns": "IApiKey",
                    "description": "Add an API key for the REST API.",
                },
                "addDomainName": {
                    "params": "id: string, options: DomainNameOptions",
                    "returns": "DomainName",
                    "description": "Attach a custom domain name.",
                },
                "fromRestApiId": {
                    "params": "scope, id, restApiId: string",
                    "returns": "IRestApi",
                    "description": "Import an existing REST API by ID.",
                    "static": True,
                },
            },
        },
        "HttpApi": {
            "description": "L2 construct for API Gateway HTTP APIs (v2) — simpler, lower-latency, lower-cost than REST APIs.",
            "rpcs": {
                "constructor": {
                    "signature": "HttpApi(scope, id, props?)",
                    "description": "Creates a new HTTP API.",
                },
                "addRoutes": {
                    "params": "options: AddRoutesOptions",
                    "description": "Add routes with path, methods, and integration.",
                },
                "addStage": {
                    "params": "id: string, options: HttpStageOptions",
                    "returns": "HttpStage",
                    "description": "Add a deployment stage.",
                },
                "fromHttpApiAttributes": {
                    "params": "scope, id, attrs: HttpApiAttributes",
                    "returns": "IHttpApi",
                    "description": "Import an existing HTTP API.",
                    "static": True,
                },
            },
        },
        "LambdaRestApi": {
            "description": "Convenience construct — REST API backed by a single Lambda function (proxy integration).",
            "rpcs": {
                "constructor": {
                    "signature": "LambdaRestApi(scope, id, props)",
                    "description": "Creates a REST API with Lambda proxy integration for all routes.",
                    "required_props": ["handler (IFunction)"],
                },
            },
        },
    },
    "key_messages": {
        "RestApiProps": {
            "fields": [
                "restApiName (string, optional)", "description (string, optional)",
                "deployOptions (StageOptions — stage name, throttling, logging, tracing)",
                "endpointTypes (EndpointType[] — REGIONAL | EDGE | PRIVATE)",
                "policy (PolicyDocument — resource policy)",
                "defaultCorsPreflightOptions (CorsOptions — CORS config)",
                "defaultIntegration (Integration — default backend)",
                "defaultMethodOptions (MethodOptions — auth, API key, etc.)",
                "cloudWatchRole (boolean — auto-create CW role, default true)",
                "binaryMediaTypes (string[] — e.g., ['image/*', 'application/octet-stream'])",
                "minCompressionSize (Size — enable response compression)",
                "apiKeySourceType (ApiKeySourceType — HEADER | AUTHORIZER)",
                "domainName (DomainNameOptions, optional)",
                "disableExecuteApiEndpoint (boolean — disable default execute-api endpoint)",
            ],
        },
        "HttpApiProps": {
            "fields": [
                "apiName (string, optional)", "description (string, optional)",
                "corsPreflight (CorsPreflightOptions)",
                "defaultAuthorizer (IHttpRouteAuthorizer)",
                "defaultAuthorizationScopes (string[])",
                "defaultDomainMapping (DomainMappingOptions)",
                "createDefaultStage (boolean — default true)",
                "disableExecuteApiEndpoint (boolean)",
            ],
        },
        "RestApi_outputs": {
            "fields": [
                "restApiId (string)", "restApiName (string)",
                "url (string — invoke URL for the default stage)",
                "deploymentStage (Stage)", "root (IResource — root resource '/')",
            ],
        },
        "HttpApi_outputs": {
            "fields": [
                "httpApiId (string)", "apiEndpoint (string — invoke URL)",
                "apiId (string)",
            ],
        },
        "Integration_types": [
            "LambdaIntegration — proxy or custom Lambda integration",
            "HttpIntegration — HTTP proxy to external URL",
            "MockIntegration — mock response without backend",
            "AwsIntegration — direct AWS service integration",
            "StepFunctionsIntegration — Step Functions state machine",
        ],
        "Authorizer_types": [
            "TokenAuthorizer — Lambda token authorizer (API key / JWT in header)",
            "RequestAuthorizer — Lambda request authorizer (headers, query, context)",
            "CognitoUserPoolsAuthorizer — Cognito User Pool authorizer",
            "HttpIamAuthorizer — IAM-based auth (v2)",
            "HttpJwtAuthorizer — JWT authorizer (v2)",
            "HttpLambdaAuthorizer — Lambda authorizer (v2)",
        ],
    },
    "polar_gosling_usage": (
        "API Gateway fronts the MotherGoose FastAPI service on AWS. "
        "Routes: /eggs, /runners, /webhooks/gitlab (X-Gitlab-Token auth), "
        "/internal/sync-git (secret token), /health, /binaries. "
        "Configured in MG/config.fly in the Nest repository."
    ),
}

AWS_CDK_SERVICES_INDEX: dict[str, Any] = {
    "description": (
        "Index of all AWS CDK construct modules covered by this MCP server, "
        "with tool names and relevance to Polar Gosling."
    ),
    "services": [
        {
            "name": "Amazon SQS",
            "tool": "get_aws_cdk_sqs",
            "package": "aws-cdk-lib.aws_sqs",
            "polar_gosling_role": "Celery message broker (task queue between MotherGoose and UglyFox)",
        },
        {
            "name": "Amazon DynamoDB",
            "tool": "get_aws_cdk_dynamodb",
            "package": "aws-cdk-lib.aws_dynamodb",
            "polar_gosling_role": "AWS database backend (runners, egg_configs, sync_history, etc.)",
        },
        {
            "name": "Amazon S3",
            "tool": "get_aws_cdk_s3",
            "package": "aws-cdk-lib.aws_s3",
            "polar_gosling_role": "Binary storage, Rift image cache, artifact cache",
        },
        {
            "name": "AWS Lambda",
            "tool": "get_aws_cdk_lambda",
            "package": "aws-cdk-lib.aws_lambda",
            "polar_gosling_role": "Serverless compute for event-driven integrations and API handlers",
        },
        {
            "name": "Amazon ECS (Fargate)",
            "tool": "get_aws_cdk_ecs_fargate",
            "package": "aws-cdk-lib.aws_ecs",
            "polar_gosling_role": "SERVERLESS runner deployment, MotherGoose/UglyFox container hosting",
        },
        {
            "name": "Amazon EventBridge",
            "tool": "get_aws_cdk_events",
            "package": "aws-cdk-lib.aws_events",
            "polar_gosling_role": "Scheduled triggers (5-min Git sync), event-driven workflows",
        },
        {
            "name": "Amazon API Gateway",
            "tool": "get_aws_cdk_api_gateway",
            "package": "aws-cdk-lib.aws_apigateway / aws_apigatewayv2",
            "polar_gosling_role": "MotherGoose REST API frontend (webhooks, sync, eggs, runners)",
        },
    ],
    "docs_base_url": "https://docs.aws.amazon.com/cdk/api/v2/docs",
}
