"""Yandex Cloud gRPC API specs for the MCP server.

Covers the YC services most relevant to Polar Gosling, parsed from the
official protobuf definitions at https://github.com/yandex-cloud/cloudapi.
"""

from typing import Any

YC_GRPC_API_OVERVIEW: dict[str, Any] = {
    "description": (
        "Yandex Cloud gRPC API reference for services used by Polar Gosling. "
        "All APIs follow the standard YC patterns: protobuf-defined, async operations "
        "via operation.Operation, pagination via page_size/page_token, and IAM-based auth. "
        "Source: https://github.com/yandex-cloud/cloudapi"
    ),
    "common_patterns": {
        "endpoint_format": "<service>.api.cloud.yandex.net:443",
        "auth": "IAM token in Authorization header or service-account key via gRPC metadata",
        "async_operations": (
            "Mutating RPCs return yandex.cloud.operation.Operation. "
            "Poll via OperationService.Get until done=true. "
            "metadata field contains *Metadata message, response contains the result."
        ),
        "pagination": "page_size (int64, 0-1000, default 100) + page_token (string)",
        "standard_methods": ["Get", "List", "Create", "Update", "Delete", "ListOperations"],
        "access_bindings": "Most resources support ListAccessBindings / SetAccessBindings / UpdateAccessBindings",
    },
    "protobuf_repo": "https://github.com/yandex-cloud/cloudapi",
    "docs_base_url": "https://yandex.cloud/en/docs",
}

YC_API_GATEWAY: dict[str, Any] = {
    "service_name": "API Gateway",
    "package": "yandex.cloud.serverless.apigateway.v1",
    "proto_path": "yandex/cloud/serverless/apigateway/v1/apigateway_service.proto",
    "endpoint": "serverless-apigateway.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/api-gateway/apigateway/api-ref/grpc/",
    "description": "Manages API Gateway resources — OpenAPI-spec-based HTTP gateways that route to Cloud Functions, Containers, Object Storage, etc.",
    "services": {
        "ApiGatewayService": {
            "description": "CRUD and lifecycle management for API gateways.",
            "rpcs": {
                "Get": {
                    "request": "GetApiGatewayRequest",
                    "response": "ApiGateway",
                    "http": "GET /apigateways/v1/apigateways/{api_gateway_id}",
                    "description": "Returns the specified API gateway.",
                },
                "List": {
                    "request": "ListApiGatewayRequest",
                    "response": "ListApiGatewayResponse",
                    "http": "GET /apigateways/v1/apigateways",
                    "description": "Retrieves the list of API gateways in the specified folder.",
                },
                "Create": {
                    "request": "CreateApiGatewayRequest",
                    "response": "operation.Operation",
                    "http": "POST /apigateways/v1/apigateways",
                    "description": "Creates an API gateway in the specified folder.",
                    "operation_metadata": "CreateApiGatewayMetadata",
                    "operation_response": "ApiGateway",
                },
                "Update": {
                    "request": "UpdateApiGatewayRequest",
                    "response": "operation.Operation",
                    "http": "PATCH /apigateways/v1/apigateways/{api_gateway_id}",
                    "description": "Updates the specified API gateway.",
                    "operation_metadata": "UpdateApiGatewayMetadata",
                    "operation_response": "ApiGateway",
                },
                "Delete": {
                    "request": "DeleteApiGatewayRequest",
                    "response": "operation.Operation",
                    "http": "DELETE /apigateways/v1/apigateways/{api_gateway_id}",
                    "description": "Deletes the specified API gateway.",
                    "operation_metadata": "DeleteApiGatewayMetadata",
                    "operation_response": "google.protobuf.Empty",
                },
                "AddDomain": {
                    "request": "AddDomainRequest",
                    "response": "operation.Operation",
                    "description": "Attaches a custom domain to the API gateway.",
                },
                "RemoveDomain": {
                    "request": "RemoveDomainRequest",
                    "response": "operation.Operation",
                    "description": "Detaches a custom domain from the API gateway.",
                },
                "GetOpenapiSpec": {
                    "request": "GetOpenapiSpecRequest",
                    "response": "GetOpenapiSpecResponse",
                    "description": "Returns the OpenAPI spec of the specified API gateway.",
                },
                "ListOperations": {
                    "request": "ListOperationsRequest",
                    "response": "ListOperationsResponse",
                    "description": "Lists operations for the specified API gateway.",
                },
                "ListAccessBindings": {
                    "request": "access.ListAccessBindingsRequest",
                    "response": "access.ListAccessBindingsResponse",
                },
                "SetAccessBindings": {
                    "request": "access.SetAccessBindingsRequest",
                    "response": "operation.Operation",
                },
                "UpdateAccessBindings": {
                    "request": "access.UpdateAccessBindingsRequest",
                    "response": "operation.Operation",
                },
            },
        },
    },
    "key_messages": {
        "ApiGateway": {
            "fields": [
                "id (string)", "folder_id (string)", "name (string)", "description (string)",
                "labels (map<string,string>)", "status (Status enum)", "domain (string)",
                "log_group_id (string)", "attached_domains (repeated AttachedDomain)",
                "connectivity (Connectivity)", "log_options (LogOptions)",
                "variables (map<string,VariableInput>)", "canary (Canary)",
                "execution_timeout (google.protobuf.Duration)",
            ],
            "status_enum": ["STATUS_UNSPECIFIED", "CREATING", "ACTIVE", "DELETING", "ERROR", "UPDATING"],
        },
    },
}


YC_OBJECT_STORAGE: dict[str, Any] = {
    "service_name": "Object Storage (S3)",
    "package": "yandex.cloud.storage.v1",
    "proto_path": "yandex/cloud/storage/v1/bucket_service.proto",
    "endpoint": "storage.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/storage/s3/",
    "description": (
        "Manages S3-compatible Object Storage buckets via gRPC. "
        "For object-level operations (PutObject, GetObject, etc.) use the S3-compatible REST API. "
        "The gRPC API handles bucket-level management, HTTPS config, access bindings, and inventory."
    ),
    "services": {
        "BucketService": {
            "description": "CRUD for buckets, HTTPS config, access bindings, inventory configurations.",
            "rpcs": {
                "List": {
                    "request": "ListBucketsRequest",
                    "response": "ListBucketsResponse",
                    "http": "GET /storage/v1/buckets",
                    "description": "Retrieves the list of buckets in the specified folder.",
                },
                "Get": {
                    "request": "GetBucketRequest",
                    "response": "Bucket",
                    "http": "GET /storage/v1/buckets/{name}",
                    "description": "Returns the specified bucket. Supports VIEW_BASIC, VIEW_ACL, VIEW_FULL.",
                },
                "Create": {
                    "request": "CreateBucketRequest",
                    "response": "operation.Operation",
                    "http": "POST /storage/v1/buckets",
                    "description": "Creates a bucket in the specified folder.",
                    "operation_metadata": "CreateBucketMetadata",
                    "operation_response": "Bucket",
                },
                "Update": {
                    "request": "UpdateBucketRequest",
                    "response": "operation.Operation",
                    "http": "PATCH /storage/v1/buckets/{name}",
                    "description": "Updates the specified bucket (CORS, lifecycle, ACL, versioning, website, policy, encryption, tags).",
                    "operation_metadata": "UpdateBucketMetadata",
                    "operation_response": "Bucket",
                },
                "Delete": {
                    "request": "DeleteBucketRequest",
                    "response": "operation.Operation",
                    "http": "DELETE /storage/v1/buckets/{name}",
                    "description": "Deletes the specified bucket.",
                    "operation_metadata": "DeleteBucketMetadata",
                    "operation_response": "google.protobuf.Empty",
                },
                "GetStats": {
                    "request": "GetBucketStatsRequest",
                    "response": "BucketStats",
                    "http": "GET /storage/v1/buckets/{name}:getStats",
                    "description": "Returns statistics for the specified bucket.",
                },
                "GetHTTPSConfig": {
                    "request": "GetBucketHTTPSConfigRequest",
                    "response": "HTTPSConfig",
                    "http": "GET /storage/v1/buckets/{name}:getHttpsConfig",
                    "description": "Returns the HTTPS configuration for the specified bucket.",
                },
                "SetHTTPSConfig": {
                    "request": "SetBucketHTTPSConfigRequest",
                    "response": "operation.Operation",
                    "http": "POST /storage/v1/buckets/{name}:setHttpsConfig",
                    "description": "Updates the HTTPS configuration (self-managed PEM or Certificate Manager).",
                },
                "DeleteHTTPSConfig": {
                    "request": "DeleteBucketHTTPSConfigRequest",
                    "response": "operation.Operation",
                    "description": "Deletes the HTTPS configuration for the specified bucket.",
                },
                "ListAccessBindings": {
                    "request": "access.ListAccessBindingsRequest",
                    "response": "access.ListAccessBindingsResponse",
                },
                "SetAccessBindings": {
                    "request": "access.SetAccessBindingsRequest",
                    "response": "operation.Operation",
                },
                "UpdateAccessBindings": {
                    "request": "access.UpdateAccessBindingsRequest",
                    "response": "operation.Operation",
                },
                "CreateInventoryConfiguration": {
                    "request": "CreateBucketInventoryConfigurationRequest",
                    "response": "operation.Operation",
                    "description": "Creates or updates an inventory configuration.",
                },
                "GetInventoryConfiguration": {
                    "request": "GetBucketInventoryConfigurationRequest",
                    "response": "InventoryConfiguration",
                },
                "DeleteInventoryConfiguration": {
                    "request": "DeleteBucketInventoryConfigurationRequest",
                    "response": "operation.Operation",
                },
                "ListInventoryConfigurations": {
                    "request": "ListBucketInventoryConfigurationsRequest",
                    "response": "ListBucketInventoryConfigurationsResponse",
                },
            },
        },
        "PresignService": {
            "description": "Generates pre-signed URLs for S3 operations.",
            "rpcs": {
                "Sign": {
                    "request": "SignRequest",
                    "response": "SignResponse",
                    "description": "Generates a pre-signed URL for an S3 operation.",
                },
            },
        },
    },
    "key_messages": {
        "Bucket": {
            "fields": [
                "name (string)", "folder_id (string)", "anonymous_access_flags (AnonymousAccessFlags)",
                "default_storage_class (string: STANDARD|COLD|ICE)", "versioning (Versioning enum)",
                "max_size (int64)", "policy (google.protobuf.Struct)", "acl (ACL)",
                "created_at (google.protobuf.Timestamp)", "cors (repeated CorsRule)",
                "website_settings (WebsiteSettings)", "lifecycle_rules (repeated LifecycleRule)",
                "tags (repeated Tag)", "object_lock (ObjectLock)", "encryption (Encryption)",
            ],
        },
        "storage_classes": ["STANDARD", "COLD (STANDARD_IA, NEARLINE)", "ICE (GLACIER)"],
    },
    "s3_compatible_rest_api": {
        "endpoint": "storage.yandexcloud.net",
        "description": "Object-level operations use the S3-compatible REST API, not gRPC.",
        "key_operations": [
            "PutObject", "GetObject", "DeleteObject", "ListObjectsV2",
            "CopyObject", "CreateMultipartUpload", "HeadObject",
        ],
    },
}

YC_YDB: dict[str, Any] = {
    "service_name": "Managed YDB",
    "package": "yandex.cloud.ydb.v1",
    "proto_path": "yandex/cloud/ydb/v1/database_service.proto",
    "endpoint": "ydb.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/ydb/api-ref/grpc/",
    "description": "Manages YDB database instances (Dedicated and Serverless). For data-plane operations (queries, transactions) use the native YDB SDK, not this management API.",
    "services": {
        "DatabaseService": {
            "description": "CRUD and lifecycle management for YDB databases.",
            "rpcs": {
                "Get": {
                    "request": "GetDatabaseRequest",
                    "response": "Database",
                    "http": "GET /ydb/v1/databases/{database_id}",
                    "description": "Returns the specified database.",
                },
                "List": {
                    "request": "ListDatabasesRequest",
                    "response": "ListDatabasesResponse",
                    "http": "GET /ydb/v1/databases",
                    "description": "Retrieves a list of databases in the specified folder.",
                },
                "Create": {
                    "request": "CreateDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "POST /ydb/v1/databases",
                    "description": "Creates a new database (Dedicated or Serverless).",
                    "operation_metadata": "CreateDatabaseMetadata",
                    "operation_response": "Database",
                    "key_fields": [
                        "folder_id", "name", "resource_preset_id", "storage_config",
                        "scale_policy", "network_id", "subnet_ids",
                        "dedicated_database | serverless_database (oneof database_type)",
                        "assign_public_ips", "backup_config", "deletion_protection",
                    ],
                },
                "Update": {
                    "request": "UpdateDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "PATCH /ydb/v1/databases/{database_id}",
                    "description": "Modifies the specified database.",
                },
                "Delete": {
                    "request": "DeleteDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "DELETE /ydb/v1/databases/{database_id}",
                    "description": "Deletes the specified database.",
                },
                "Start": {
                    "request": "StartDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "POST /ydb/v1/databases/{database_id}:start",
                    "description": "Starts the specified database.",
                },
                "Stop": {
                    "request": "StopDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "POST /ydb/v1/databases/{database_id}:stop",
                    "description": "Stops the specified database.",
                },
                "Move": {
                    "request": "MoveDatabaseRequest",
                    "response": "operation.Operation",
                    "http": "POST /ydb/v1/databases/{database_id}:move",
                    "description": "Moves the database to another folder.",
                },
                "Backup": {
                    "request": "BackupDatabaseRequest",
                    "response": "operation.Operation",
                    "description": "Creates a backup of the specified database.",
                },
                "Restore": {
                    "request": "RestoreBackupRequest",
                    "response": "operation.Operation",
                    "http": "POST /ydb/v1/databases:restore",
                    "description": "Restores a database from the specified backup.",
                },
                "ListAccessBindings": {
                    "request": "access.ListAccessBindingsRequest",
                    "response": "access.ListAccessBindingsResponse",
                },
                "SetAccessBindings": {
                    "request": "access.SetAccessBindingsRequest",
                    "response": "operation.Operation",
                },
                "UpdateAccessBindings": {
                    "request": "access.UpdateAccessBindingsRequest",
                    "response": "operation.Operation",
                },
            },
        },
        "BackupService": {
            "description": "Manages YDB backups.",
            "rpcs": {
                "Get": {"description": "Returns the specified backup."},
                "List": {"description": "Lists backups in the specified folder."},
                "Delete": {"description": "Deletes the specified backup."},
                "ListPaths": {"description": "Lists paths included in the backup."},
            },
        },
        "LocationService": {
            "description": "Lists available locations for YDB databases.",
            "rpcs": {
                "Get": {"description": "Returns the specified location."},
                "List": {"description": "Lists available locations."},
            },
        },
        "ResourcePresetService": {
            "description": "Lists available resource presets for Dedicated databases.",
            "rpcs": {
                "Get": {"description": "Returns the specified resource preset."},
                "List": {"description": "Lists available resource presets."},
            },
        },
        "StorageTypeService": {
            "description": "Lists available storage types.",
            "rpcs": {
                "Get": {"description": "Returns the specified storage type."},
                "List": {"description": "Lists available storage types."},
            },
        },
    },
    "key_messages": {
        "Database": {
            "fields": [
                "id (string)", "folder_id (string)", "name (string)", "description (string)",
                "status (Status enum)", "endpoint (string)", "resource_preset_id (string)",
                "storage_config (StorageConfig)", "scale_policy (ScalePolicy)",
                "network_id (string)", "subnet_ids (repeated string)",
                "dedicated_database | serverless_database (oneof database_type)",
                "assign_public_ips (bool)", "location_id (string)", "labels (map)",
                "backup_config (BackupConfig)", "monitoring_config (MonitoringConfig)",
                "deletion_protection (bool)",
            ],
        },
        "database_types": {
            "DedicatedDatabase": "Fixed-resource database with explicit compute/storage allocation",
            "ServerlessDatabase": "Auto-scaling database billed per request unit",
        },
    },
    "data_plane_note": (
        "This is the management-plane API. For data operations (queries, transactions, "
        "table management) use the native YDB SDK: pip install ydb / go get github.com/ydb-platform/ydb-go-sdk"
    ),
}


YC_SERVERLESS_CONTAINERS: dict[str, Any] = {
    "service_name": "Serverless Containers",
    "package": "yandex.cloud.serverless.containers.v1",
    "proto_path": "yandex/cloud/serverless/containers/v1/container_service.proto",
    "endpoint": "serverless-containers.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/serverless-containers/containers/api-ref/grpc/",
    "description": "Manages Serverless Containers — auto-scaling container instances invoked via HTTP or triggers. Used by Polar Gosling for deploying SERVERLESS runner type.",
    "services": {
        "ContainerService": {
            "description": "CRUD and revision management for serverless containers.",
            "rpcs": {
                "Get": {
                    "request": "GetContainerRequest",
                    "response": "Container",
                    "http": "GET /containers/v1/containers/{container_id}",
                    "description": "Returns the specified container.",
                },
                "List": {
                    "request": "ListContainersRequest",
                    "response": "ListContainersResponse",
                    "http": "GET /containers/v1/containers",
                    "description": "Retrieves the list of containers in the specified folder.",
                },
                "Create": {
                    "request": "CreateContainerRequest",
                    "response": "operation.Operation",
                    "http": "POST /containers/v1/containers",
                    "description": "Creates a container in the specified folder.",
                    "operation_metadata": "CreateContainerMetadata",
                    "operation_response": "Container",
                },
                "Update": {
                    "request": "UpdateContainerRequest",
                    "response": "operation.Operation",
                    "http": "PATCH /containers/v1/containers/{container_id}",
                    "description": "Updates the specified container.",
                },
                "Delete": {
                    "request": "DeleteContainerRequest",
                    "response": "operation.Operation",
                    "http": "DELETE /containers/v1/containers/{container_id}",
                    "description": "Deletes the specified container.",
                },
                "DeployRevision": {
                    "request": "DeployContainerRevisionRequest",
                    "response": "operation.Operation",
                    "http": "POST /containers/v1/revisions:deploy",
                    "description": "Deploys a new revision of the container with the specified image and settings.",
                    "operation_metadata": "DeployContainerRevisionMetadata",
                    "operation_response": "Revision",
                    "key_fields": [
                        "container_id", "image_spec (ImageSpec: image_url, command, args, environment, working_dir)",
                        "resources (Resources: memory, cores, core_fraction)",
                        "execution_timeout", "service_account_id", "concurrency",
                        "secrets (repeated Secret)", "connectivity (Connectivity)",
                        "provision_policy (ProvisionPolicy: min_instances)",
                        "log_options (LogOptions)",
                    ],
                },
                "Rollback": {
                    "request": "RollbackContainerRequest",
                    "response": "operation.Operation",
                    "description": "Rolls back the container to the specified revision.",
                },
                "GetRevision": {
                    "request": "GetContainerRevisionRequest",
                    "response": "Revision",
                    "description": "Returns the specified revision of a container.",
                },
                "ListRevisions": {
                    "request": "ListContainersRevisionsRequest",
                    "response": "ListContainersRevisionsResponse",
                    "description": "Retrieves the list of revisions for the specified container or folder.",
                },
                "ListOperations": {
                    "request": "ListContainerOperationsRequest",
                    "response": "ListContainerOperationsResponse",
                },
                "ListAccessBindings": {
                    "request": "access.ListAccessBindingsRequest",
                    "response": "access.ListAccessBindingsResponse",
                },
                "SetAccessBindings": {
                    "request": "access.SetAccessBindingsRequest",
                    "response": "operation.Operation",
                },
                "UpdateAccessBindings": {
                    "request": "access.UpdateAccessBindingsRequest",
                    "response": "operation.Operation",
                },
            },
        },
    },
    "key_messages": {
        "Container": {
            "fields": [
                "id (string)", "folder_id (string)", "name (string)", "description (string)",
                "labels (map<string,string>)", "url (string — invoke URL)",
                "status (Status enum)", "created_at (Timestamp)", "updated_at (Timestamp)",
            ],
            "status_enum": ["STATUS_UNSPECIFIED", "CREATING", "ACTIVE", "DELETING", "ERROR"],
        },
        "Revision": {
            "fields": [
                "id (string)", "container_id (string)", "description (string)",
                "created_at (Timestamp)", "image (Image)", "resources (Resources)",
                "execution_timeout (Duration)", "concurrency (int64)",
                "service_account_id (string)", "status (Status enum)",
                "secrets (repeated Secret)", "connectivity (Connectivity)",
                "provision_policy (ProvisionPolicy)", "log_options (LogOptions)",
            ],
        },
        "Resources": {
            "fields": ["memory (int64, bytes)", "cores (int64)", "core_fraction (int64, percent)"],
        },
    },
}

YC_COMPUTE: dict[str, Any] = {
    "service_name": "Compute Cloud",
    "package": "yandex.cloud.compute.v1",
    "proto_path": "yandex/cloud/compute/v1/instance_service.proto",
    "endpoint": "compute.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/compute/api-ref/grpc/",
    "description": "Manages Compute Cloud VMs, disks, images, snapshots, and placement groups. Used by Polar Gosling for deploying APEX/NADIR VM-based runners.",
    "services": {
        "InstanceService": {
            "description": "CRUD and lifecycle management for VM instances.",
            "rpcs": {
                "Get": {"description": "Returns the specified instance."},
                "List": {"description": "Retrieves the list of instances in the specified folder."},
                "Create": {
                    "description": "Creates an instance in the specified folder.",
                    "key_fields": [
                        "folder_id", "name", "zone_id", "platform_id",
                        "resources_spec (ResourcesSpec: memory, cores, core_fraction, gpus)",
                        "boot_disk_spec (AttachedDiskSpec)", "network_interface_specs",
                        "metadata (map<string,string> — cloud-init / user-data)",
                        "service_account_id", "scheduling_policy (SchedulingPolicy: preemptible)",
                    ],
                },
                "Update": {"description": "Updates the specified instance."},
                "Delete": {"description": "Deletes the specified instance."},
                "Start": {"description": "Starts the stopped instance."},
                "Stop": {"description": "Stops the running instance."},
                "Restart": {"description": "Restarts the running instance."},
                "AttachDisk": {"description": "Attaches a disk to the instance."},
                "DetachDisk": {"description": "Detaches a disk from the instance."},
                "AttachFilesystem": {"description": "Attaches a filesystem to the instance."},
                "DetachFilesystem": {"description": "Detaches a filesystem from the instance."},
                "AttachNetworkInterface": {"description": "Attaches a network interface to the instance."},
                "DetachNetworkInterface": {"description": "Detaches a network interface from the instance."},
                "AddOneToOneNat": {"description": "Adds a one-to-one NAT to the interface."},
                "RemoveOneToOneNat": {"description": "Removes a one-to-one NAT from the interface."},
                "UpdateMetadata": {"description": "Updates the metadata of the specified instance."},
                "GetSerialPortOutput": {"description": "Returns the serial port output of the instance."},
                "SimulateMaintenanceEvent": {"description": "Simulates a maintenance event for the instance."},
                "Move": {"description": "Moves the instance to another folder."},
                "Relocate": {"description": "Moves the instance to another availability zone."},
                "ListOperations": {"description": "Lists operations for the specified instance."},
                "ListAccessBindings": {"description": "Lists access bindings for the instance."},
                "SetAccessBindings": {"description": "Sets access bindings for the instance."},
                "UpdateAccessBindings": {"description": "Updates access bindings for the instance."},
            },
        },
        "DiskService": {
            "description": "CRUD for disks.",
            "rpcs": {
                "Get": {"description": "Returns the specified disk."},
                "List": {"description": "Retrieves the list of disks in the specified folder."},
                "Create": {"description": "Creates a disk in the specified folder."},
                "Update": {"description": "Updates the specified disk."},
                "Delete": {"description": "Deletes the specified disk."},
                "Move": {"description": "Moves the disk to another folder."},
                "Relocate": {"description": "Moves the disk to another availability zone."},
                "ListSnapshotSchedules": {"description": "Lists snapshot schedules for the disk."},
            },
        },
        "ImageService": {
            "description": "CRUD for images.",
            "rpcs": {
                "Get": {"description": "Returns the specified image."},
                "List": {"description": "Retrieves the list of images in the specified folder."},
                "GetLatestByFamily": {"description": "Returns the latest image from the specified family."},
                "Create": {"description": "Creates an image in the specified folder."},
                "Update": {"description": "Updates the specified image."},
                "Delete": {"description": "Deletes the specified image."},
            },
        },
        "SnapshotService": {
            "description": "CRUD for snapshots.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
            },
        },
        "ZoneService": {
            "description": "Lists availability zones.",
            "rpcs": {
                "Get": {"description": "Returns the specified availability zone."},
                "List": {"description": "Lists availability zones."},
            },
        },
    },
    "key_messages": {
        "Instance": {
            "fields": [
                "id", "folder_id", "name", "description", "labels", "zone_id",
                "platform_id", "resources (Resources)", "status (Status enum)",
                "metadata (map<string,string>)", "metadata_options",
                "boot_disk (AttachedDisk)", "secondary_disks", "local_disks",
                "filesystems", "network_interfaces (repeated NetworkInterface)",
                "serial_port_settings", "gpu_settings", "fqdn",
                "scheduling_policy (SchedulingPolicy)", "service_account_id",
                "network_settings", "placement_policy", "host_group_id",
                "host_id", "maintenance_policy", "maintenance_grace_period",
            ],
            "status_enum": [
                "STATUS_UNSPECIFIED", "PROVISIONING", "RUNNING", "STOPPING",
                "STOPPED", "STARTING", "RESTARTING", "UPDATING", "ERROR",
                "CRASHED", "DELETING",
            ],
        },
    },
}

YC_LOCKBOX: dict[str, Any] = {
    "service_name": "Lockbox",
    "package": "yandex.cloud.lockbox.v1",
    "proto_path": "yandex/cloud/lockbox/v1/secret_service.proto",
    "endpoint": "lockbox.api.cloud.yandex.net:443",
    "payload_endpoint": "payload.lockbox.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/lockbox/api-ref/grpc/",
    "description": "Manages secrets and their versions. Used by Polar Gosling via yc-lockbox:// URI scheme for resolving runner tokens and other secrets.",
    "services": {
        "SecretService": {
            "description": "CRUD for secrets and their versions (management plane).",
            "rpcs": {
                "Get": {"description": "Returns the specified secret (metadata only, not payload)."},
                "List": {"description": "Retrieves the list of secrets in the specified folder."},
                "Create": {
                    "description": "Creates a secret in the specified folder.",
                    "key_fields": [
                        "folder_id", "name", "description", "labels", "kms_key_id",
                        "version_description", "version_payload_entries (repeated PayloadEntryChange)",
                        "deletion_protection",
                    ],
                },
                "Update": {"description": "Updates the specified secret metadata."},
                "Delete": {"description": "Deletes the specified secret."},
                "Activate": {"description": "Activates the specified secret."},
                "Deactivate": {"description": "Deactivates the specified secret."},
                "AddVersion": {
                    "description": "Adds a new version with the specified payload entries.",
                    "key_fields": [
                        "secret_id", "description",
                        "payload_entries (repeated PayloadEntryChange: key, text_value | binary_value)",
                        "base_version_id",
                    ],
                },
                "ListVersions": {"description": "Lists versions of the specified secret."},
                "ScheduleVersionDestruction": {"description": "Schedules destruction of a secret version."},
                "CancelVersionDestruction": {"description": "Cancels scheduled destruction."},
                "ListOperations": {},
                "ListAccessBindings": {},
                "SetAccessBindings": {},
                "UpdateAccessBindings": {},
            },
        },
        "PayloadService": {
            "description": "Retrieves secret payloads (data plane). Uses a separate endpoint.",
            "endpoint": "payload.lockbox.api.cloud.yandex.net:443",
            "rpcs": {
                "Get": {
                    "request": "GetPayloadRequest",
                    "response": "Payload",
                    "description": "Returns the payload of the specified secret version. Fields: secret_id, version_id (optional, defaults to latest).",
                },
            },
        },
    },
    "key_messages": {
        "Secret": {
            "fields": [
                "id", "folder_id", "name", "description", "labels",
                "status (Status enum: CREATING, ACTIVE, INACTIVE)",
                "current_version (Version)", "deletion_protection (bool)",
                "kms_key_id (string — optional KMS encryption key)",
            ],
        },
        "Payload": {
            "fields": [
                "version_id (string)",
                "entries (repeated Entry: key (string), text_value (string) | binary_value (bytes))",
            ],
        },
    },
    "polar_gosling_usage": (
        "MotherGoose SecretManager resolves yc-lockbox://{secret_id}/{key} URIs by calling "
        "PayloadService.Get(secret_id=secret_id) and extracting the entry matching the key. "
        "Requires lockbox.payloadViewer role on the secret."
    ),
}

YC_VPC: dict[str, Any] = {
    "service_name": "Virtual Private Cloud",
    "package": "yandex.cloud.vpc.v1",
    "proto_path": "yandex/cloud/vpc/v1/",
    "endpoint": "vpc.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/vpc/api-ref/grpc/",
    "description": "Manages VPC networks, subnets, security groups, route tables, and gateways.",
    "services": {
        "NetworkService": {
            "description": "CRUD for VPC networks.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "ListSubnets": {"description": "Lists subnets in the specified network."},
                "ListSecurityGroups": {"description": "Lists security groups in the specified network."},
                "ListRouteTables": {"description": "Lists route tables in the specified network."},
                "Move": {"description": "Moves the network to another folder."},
                "ListOperations": {},
            },
        },
        "SubnetService": {
            "description": "CRUD for subnets.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "Move": {}, "Relocate": {}, "ListOperations": {},
                "ListUsedAddresses": {"description": "Lists used addresses in the subnet."},
            },
        },
        "SecurityGroupService": {
            "description": "CRUD for security groups and their rules.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "Move": {}, "UpdateRules": {"description": "Updates rules of the security group."},
                "ListOperations": {},
            },
        },
        "RouteTableService": {
            "description": "CRUD for route tables.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}, "Move": {}},
        },
        "GatewayService": {
            "description": "CRUD for NAT gateways.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}, "Move": {}},
        },
        "AddressService": {
            "description": "CRUD for static public IP addresses.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}, "Move": {}},
        },
    },
}

YC_IAM: dict[str, Any] = {
    "service_name": "Identity and Access Management",
    "package": "yandex.cloud.iam.v1",
    "proto_path": "yandex/cloud/iam/v1/",
    "endpoint": "iam.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/iam/api-ref/grpc/",
    "description": "Manages IAM tokens, service accounts, keys, and access bindings.",
    "services": {
        "IamTokenService": {
            "description": "Creates IAM tokens.",
            "rpcs": {
                "Create": {
                    "description": "Creates an IAM token for the specified identity.",
                    "key_fields": ["yandex_passport_oauth_token | jwt (oneof identity)"],
                },
                "CreateForServiceAccount": {
                    "description": "Creates an IAM token for a service account.",
                },
            },
        },
        "ServiceAccountService": {
            "description": "CRUD for service accounts.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "ListOperations": {},
                "ListAccessBindings": {}, "SetAccessBindings": {}, "UpdateAccessBindings": {},
            },
        },
        "KeyService": {
            "description": "Manages IAM keys for service accounts.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}},
        },
        "ApiKeyService": {
            "description": "Manages API keys for service accounts.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}},
        },
        "UserAccountService": {
            "description": "Manages user accounts.",
            "rpcs": {"Get": {"description": "Returns the specified user account."}},
        },
    },
}

YC_CONTAINER_REGISTRY: dict[str, Any] = {
    "service_name": "Container Registry",
    "package": "yandex.cloud.containerregistry.v1",
    "proto_path": "yandex/cloud/containerregistry/v1/",
    "endpoint": "container-registry.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/container-registry/api-ref/grpc/",
    "description": "Manages Docker container registries, repositories, images, and lifecycle policies.",
    "services": {
        "RegistryService": {
            "description": "CRUD for container registries.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}},
        },
        "RepositoryService": {
            "description": "Manages repositories within a registry.",
            "rpcs": {"Get": {}, "List": {}, "GetByName": {}, "Upsert": {}, "Delete": {}},
        },
        "ImageService": {
            "description": "Manages Docker images.",
            "rpcs": {"Get": {}, "List": {}, "Delete": {}},
        },
        "LifecyclePolicyService": {
            "description": "Manages image lifecycle policies (auto-cleanup).",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "DryRun": {"description": "Simulates the lifecycle policy without deleting images."},
                "ListDryRunResults": {},
                "ListDryRunResultAffectedImages": {},
            },
        },
        "ScannerService": {
            "description": "Vulnerability scanning for Docker images.",
            "rpcs": {
                "Scan": {"description": "Scans the specified Docker image for vulnerabilities."},
                "Get": {}, "GetLast": {}, "List": {}, "ListVulnerabilities": {},
            },
        },
    },
}

YC_RESOURCE_MANAGER: dict[str, Any] = {
    "service_name": "Resource Manager",
    "package": "yandex.cloud.resourcemanager.v1",
    "proto_path": "yandex/cloud/resourcemanager/v1/",
    "endpoint": "resource-manager.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/resource-manager/api-ref/grpc/",
    "description": "Manages clouds and folders — the top-level resource hierarchy.",
    "services": {
        "CloudService": {
            "description": "Manages Cloud resources.",
            "rpcs": {"Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {}, "ListOperations": {}},
        },
        "FolderService": {
            "description": "Manages Folder resources.",
            "rpcs": {
                "Get": {}, "List": {}, "Create": {}, "Update": {}, "Delete": {},
                "ListOperations": {},
                "ListAccessBindings": {}, "SetAccessBindings": {}, "UpdateAccessBindings": {},
            },
        },
    },
}

YC_TRIGGERS: dict[str, Any] = {
    "service_name": "Serverless Triggers",
    "package": "yandex.cloud.serverless.triggers.v1",
    "proto_path": "yandex/cloud/serverless/triggers/v1/trigger_service.proto",
    "endpoint": "serverless-triggers.api.cloud.yandex.net:443",
    "docs_url": "https://yandex.cloud/en/docs/functions/triggers/api-ref/grpc/",
    "description": (
        "Manages triggers that automatically invoke Cloud Functions or Serverless Containers "
        "in response to events from various sources: timers, message queues (YMQ), Object Storage, "
        "Container Registry, IoT Core, Cloud Logging, Data Streams, billing budgets, and email. "
        "Used by Polar Gosling for YC Timer triggers that fire POST /internal/sync-git every 5 minutes."
    ),
    "services": {
        "TriggerService": {
            "description": "CRUD and lifecycle management for serverless triggers.",
            "rpcs": {
                "Get": {
                    "request": "GetTriggerRequest",
                    "response": "Trigger",
                    "http": "GET /triggers/v1/triggers/{trigger_id}",
                    "description": "Returns the specified trigger.",
                },
                "List": {
                    "request": "ListTriggersRequest",
                    "response": "ListTriggersResponse",
                    "http": "GET /triggers/v1/triggers",
                    "description": "Retrieves the list of triggers in the specified folder.",
                },
                "Create": {
                    "request": "CreateTriggerRequest",
                    "response": "operation.Operation",
                    "http": "POST /triggers/v1/triggers",
                    "description": "Creates a trigger in the specified folder.",
                    "operation_metadata": "CreateTriggerMetadata",
                    "operation_response": "Trigger",
                    "key_fields": [
                        "folder_id", "name", "description", "labels",
                        "rule (Trigger.Rule — oneof: timer, message_queue, iot_message, "
                        "object_storage, container_registry, cloud_logs, logging, "
                        "billing_budget, data_stream, mail)",
                    ],
                },
                "Update": {
                    "request": "UpdateTriggerRequest",
                    "response": "operation.Operation",
                    "http": "PATCH /triggers/v1/triggers/{trigger_id}",
                    "description": "Updates the specified trigger.",
                },
                "Delete": {
                    "request": "DeleteTriggerRequest",
                    "response": "operation.Operation",
                    "http": "DELETE /triggers/v1/triggers/{trigger_id}",
                    "description": "Deletes the specified trigger.",
                },
                "Pause": {
                    "request": "PauseTriggerRequest",
                    "response": "operation.Operation",
                    "http": "POST /triggers/v1/triggers/{trigger_id}:pause",
                    "description": "Pauses the specified trigger (stops firing events).",
                },
                "Resume": {
                    "request": "ResumeTriggerRequest",
                    "response": "operation.Operation",
                    "http": "POST /triggers/v1/triggers/{trigger_id}:resume",
                    "description": "Resumes the specified paused trigger.",
                },
                "ListOperations": {
                    "request": "ListTriggerOperationsRequest",
                    "response": "ListTriggerOperationsResponse",
                    "http": "GET /triggers/v1/triggers/{trigger_id}/operations",
                },
            },
        },
    },
    "trigger_types": {
        "TIMER": {
            "description": "Fires on a cron schedule. Used by Polar Gosling for periodic git sync.",
            "key_fields": [
                "cron_expression (string, required — cron syntax)",
                "payload (string, optional — passed to function/container)",
            ],
            "actions": [
                "invoke_function (InvokeFunctionOnce)",
                "invoke_function_with_retry (InvokeFunctionWithRetry)",
                "invoke_container_with_retry (InvokeContainerWithRetry)",
                "gateway_websocket_broadcast (GatewayWebsocketBroadcast)",
            ],
        },
        "MESSAGE_QUEUE": {
            "description": "Fires when messages arrive in a YMQ queue.",
            "key_fields": [
                "queue_id (string, required)",
                "service_account_id (string, required — needs read access to queue)",
                "batch_settings (BatchSettings: size 0-1000, cutoff duration)",
                "visibility_timeout (Duration, optional, max 12h)",
            ],
            "actions": [
                "invoke_function (InvokeFunctionOnce)",
                "invoke_container (InvokeContainerOnce)",
                "gateway_websocket_broadcast",
            ],
        },
        "OBJECT_STORAGE": {
            "description": "Fires on S3 bucket events (create/update/delete object).",
            "key_fields": [
                "event_type (repeated: CREATE_OBJECT, UPDATE_OBJECT, DELETE_OBJECT)",
                "bucket_id", "prefix (optional filter)", "suffix (optional filter)",
                "batch_settings",
            ],
        },
        "CONTAINER_REGISTRY": {
            "description": "Fires on Docker image events (create/delete image or tag).",
            "key_fields": [
                "event_type (repeated: CREATE_IMAGE, DELETE_IMAGE, CREATE_IMAGE_TAG, DELETE_IMAGE_TAG)",
                "registry_id", "image_name (optional)", "tag (optional)",
            ],
        },
        "IOT_MESSAGE": {
            "description": "Fires on IoT Core MQTT messages.",
            "key_fields": ["registry_id", "device_id (optional)", "mqtt_topic"],
        },
        "LOGGING": {
            "description": "Fires on Cloud Logging events matching filters.",
            "key_fields": [
                "log_group_id", "resource_type (repeated)", "resource_id (repeated)",
                "stream_name (repeated)", "levels (repeated LogLevel)",
                "batch_settings (LoggingBatchSettings)",
            ],
        },
        "DATA_STREAM (YDS)": {
            "description": "Fires on Yandex Data Streams events.",
            "key_fields": ["endpoint", "database", "stream", "service_account_id", "batch_settings"],
        },
        "BILLING_BUDGET": {
            "description": "Fires on billing budget threshold events.",
            "key_fields": ["billing_account_id", "budget_id"],
        },
        "MAIL": {
            "description": "Fires on incoming email to a trigger-assigned address.",
            "key_fields": ["email (auto-assigned)", "batch_settings", "attachments_bucket (optional)"],
        },
    },
    "key_messages": {
        "Trigger": {
            "fields": [
                "id (string)", "folder_id (string)", "name (string)", "description (string)",
                "labels (map<string,string>)", "rule (Rule)", "status (Status enum)",
                "created_at (Timestamp)",
            ],
            "status_enum": ["STATUS_UNSPECIFIED", "ACTIVE", "PAUSED"],
        },
        "RetrySettings": {
            "fields": ["retry_attempts (int64, 1-5)", "interval (Duration, 10s-1m)"],
        },
        "BatchSettings": {
            "fields": ["size (int64, 0-1000)", "cutoff (Duration, required)"],
        },
        "PutQueueMessage (DLQ)": {
            "fields": ["queue_id (string)", "service_account_id (string)"],
            "description": "Dead letter queue config — failed messages are sent here.",
        },
    },
    "polar_gosling_usage": (
        "MotherGoose uses a TIMER trigger with a 5-minute cron expression to invoke "
        "POST /internal/sync-git on the MotherGoose API Gateway. The trigger is defined "
        "in MG/config.fly in the Nest repo. MESSAGE_QUEUE triggers can also be used "
        "to bridge YMQ events to Serverless Containers."
    ),
}

YC_MESSAGE_QUEUE: dict[str, Any] = {
    "service_name": "Yandex Message Queue (YMQ)",
    "api_type": "SQS-compatible HTTP API (not gRPC)",
    "endpoint": "https://message-queue.api.cloud.yandex.net/",
    "docs_url": "https://yandex.cloud/en/docs/message-queue/api-ref/",
    "description": (
        "SQS-compatible message queue service. YMQ does NOT have a gRPC/protobuf API — "
        "it uses an HTTP API compatible with Amazon SQS. Used by Polar Gosling as the "
        "Celery broker (via CELERY_BROKER_URL) for task queues between MotherGoose and UglyFox."
    ),
    "auth": {
        "method": "AWS Signature V4 (same as SQS)",
        "credentials": "Static access keys from IAM service account (access_key_id + secret_access_key)",
        "required_role": "ymq.editor or ymq.admin on the folder",
    },
    "queue_actions": {
        "CreateQueue": {
            "description": "Creates a new standard or FIFO queue.",
            "key_params": [
                "QueueName (required)", "Attributes (optional map)",
                "FifoQueue (bool — append .fifo to name for FIFO queues)",
            ],
            "attributes": [
                "DelaySeconds (0-900, default 0)",
                "MaximumMessageSize (1024-262144, default 262144 = 256KB)",
                "MessageRetentionPeriod (60-1209600, default 345600 = 4 days)",
                "ReceiveMessageWaitTimeSeconds (0-20, default 0 — short polling)",
                "VisibilityTimeout (0-43200, default 30)",
                "ContentBasedDeduplication (FIFO only)",
            ],
        },
        "DeleteQueue": {
            "description": "Deletes the specified queue.",
            "key_params": ["QueueUrl (required)"],
        },
        "GetQueueUrl": {
            "description": "Returns the URL of an existing queue by name.",
            "key_params": ["QueueName (required)"],
        },
        "GetQueueAttributes": {
            "description": "Returns attributes of the specified queue.",
            "key_params": ["QueueUrl (required)", "AttributeNames (list)"],
            "extra_attributes": [
                "ApproximateNumberOfMessages",
                "ApproximateNumberOfMessagesDelayed",
                "ApproximateNumberOfMessagesNotVisible",
                "CreatedTimestamp", "LastModifiedTimestamp",
                "QueueArn",
            ],
        },
        "SetQueueAttributes": {
            "description": "Sets attributes for the specified queue.",
            "key_params": ["QueueUrl (required)", "Attributes (map)"],
        },
        "ListQueues": {
            "description": "Lists queues in the folder.",
            "key_params": ["QueueNamePrefix (optional filter)"],
        },
        "PurgeQueue": {
            "description": "Deletes all messages in the specified queue.",
            "key_params": ["QueueUrl (required)"],
        },
    },
    "message_actions": {
        "SendMessage": {
            "description": "Sends a message to the specified queue.",
            "key_params": [
                "QueueUrl (required)", "MessageBody (required, up to 256KB)",
                "DelaySeconds (optional, 0-900)",
                "MessageAttributes (optional, up to 10 custom attributes)",
                "MessageDeduplicationId (required for FIFO)",
                "MessageGroupId (required for FIFO)",
            ],
        },
        "SendMessageBatch": {
            "description": "Sends up to 10 messages in a single request.",
            "key_params": ["QueueUrl (required)", "Entries (up to 10 SendMessageBatchRequestEntry)"],
        },
        "ReceiveMessage": {
            "description": "Receives up to 10 messages from the queue.",
            "key_params": [
                "QueueUrl (required)",
                "MaxNumberOfMessages (1-10, default 1)",
                "VisibilityTimeout (0-43200, overrides queue default)",
                "WaitTimeSeconds (0-20 — long polling if > 0)",
                "MessageAttributeNames (list of attribute names to include)",
            ],
        },
        "DeleteMessage": {
            "description": "Deletes a message from the queue using its receipt handle.",
            "key_params": ["QueueUrl (required)", "ReceiptHandle (required)"],
        },
        "DeleteMessageBatch": {
            "description": "Deletes up to 10 messages in a single request.",
            "key_params": ["QueueUrl (required)", "Entries (up to 10)"],
        },
        "ChangeMessageVisibility": {
            "description": "Changes the visibility timeout of a received message.",
            "key_params": ["QueueUrl", "ReceiptHandle", "VisibilityTimeout (0-43200)"],
        },
        "ChangeMessageVisibilityBatch": {
            "description": "Changes visibility timeout for up to 10 messages.",
        },
    },
    "queue_types": {
        "standard": {
            "description": "Best-effort ordering, at-least-once delivery, nearly unlimited throughput.",
            "use_case": "Polar Gosling Celery task queues (default)",
        },
        "fifo": {
            "description": "Strict ordering, exactly-once processing, 300 msg/s throughput.",
            "naming": "Queue name must end with .fifo suffix",
            "deduplication": "ContentBasedDeduplication or explicit MessageDeduplicationId",
        },
    },
    "sdk_usage": {
        "python_boto3": {
            "description": "Use boto3 with custom endpoint_url for YMQ.",
            "example_config": {
                "endpoint_url": "https://message-queue.api.cloud.yandex.net",
                "region_name": "ru-central1",
                "aws_access_key_id": "<static_key_id>",
                "aws_secret_access_key": "<static_key_secret>",
            },
        },
        "celery_broker": {
            "description": "Polar Gosling uses YMQ as Celery broker via SQS transport.",
            "broker_url_format": "sqs://",
            "env_vars": [
                "CELERY_BROKER_URL=sqs://",
                "AWS_ACCESS_KEY_ID=<ymq_static_key_id>",
                "AWS_SECRET_ACCESS_KEY=<ymq_static_key_secret>",
                "CELERY_BROKER_TRANSPORT_OPTIONS={'region': 'ru-central1', "
                "'predefined_queues': {...}, 'endpoint_url': 'https://message-queue.api.cloud.yandex.net'}",
            ],
        },
    },
    "polar_gosling_usage": (
        "YMQ is the Celery message broker for both MotherGoose and UglyFox. "
        "Task queues (git_sync, runners, webhooks, maintenance, health, pruning, lifecycle) "
        "are standard YMQ queues accessed via boto3/Kombu SQS transport. "
        "The broker URL and static keys are configured via CELERY_BROKER_URL and AWS_* env vars."
    ),
}

# ── Aggregated index for the list_yc_services tool ──────────────────────────

YC_SERVICES_INDEX: dict[str, Any] = {
    "description": "Index of all Yandex Cloud gRPC API services covered by this module.",
    "source": "https://github.com/yandex-cloud/cloudapi",
    "services": {
        "api-gateway": {
            "name": "API Gateway",
            "endpoint": "serverless-apigateway.api.cloud.yandex.net:443",
            "tool": "get_yc_api_gateway",
        },
        "object-storage": {
            "name": "Object Storage (S3)",
            "endpoint": "storage.api.cloud.yandex.net:443",
            "tool": "get_yc_object_storage",
        },
        "ydb": {
            "name": "Managed YDB",
            "endpoint": "ydb.api.cloud.yandex.net:443",
            "tool": "get_yc_ydb",
        },
        "serverless-containers": {
            "name": "Serverless Containers",
            "endpoint": "serverless-containers.api.cloud.yandex.net:443",
            "tool": "get_yc_serverless_containers",
        },
        "compute": {
            "name": "Compute Cloud",
            "endpoint": "compute.api.cloud.yandex.net:443",
            "tool": "get_yc_compute",
        },
        "lockbox": {
            "name": "Lockbox (Secrets)",
            "endpoint": "lockbox.api.cloud.yandex.net:443",
            "tool": "get_yc_lockbox",
        },
        "vpc": {
            "name": "Virtual Private Cloud",
            "endpoint": "vpc.api.cloud.yandex.net:443",
            "tool": "get_yc_vpc",
        },
        "iam": {
            "name": "Identity and Access Management",
            "endpoint": "iam.api.cloud.yandex.net:443",
            "tool": "get_yc_iam",
        },
        "container-registry": {
            "name": "Container Registry",
            "endpoint": "container-registry.api.cloud.yandex.net:443",
            "tool": "get_yc_container_registry",
        },
        "resource-manager": {
            "name": "Resource Manager",
            "endpoint": "resource-manager.api.cloud.yandex.net:443",
            "tool": "get_yc_resource_manager",
        },
        "triggers": {
            "name": "Serverless Triggers",
            "endpoint": "serverless-triggers.api.cloud.yandex.net:443",
            "tool": "get_yc_triggers",
        },
        "message-queue": {
            "name": "Yandex Message Queue (YMQ)",
            "endpoint": "https://message-queue.api.cloud.yandex.net/",
            "api_type": "SQS-compatible HTTP (not gRPC)",
            "tool": "get_yc_message_queue",
        },
    },
    "all_available_services_in_cloudapi": [
        "access", "ai", "airflow", "apploadbalancer", "audittrails", "backup",
        "baremetal", "billing", "cdn", "certificatemanager", "cic", "cloudapps",
        "clouddesktop", "cloudregistry", "cloudrouter", "compute", "connectionmanager",
        "containerregistry", "datacatalog", "dataproc", "datasphere", "datatransfer",
        "devtools", "dns", "endpoint", "gitlab", "iam", "iot", "k8s", "kms",
        "loadbalancer", "loadtesting", "lockbox", "logging", "maintenance",
        "marketplace", "mdb", "metastore", "monitoring", "oauth", "operation",
        "organizationmanager", "quota", "quotamanager", "reference", "resourcemanager",
        "searchapi", "serverless", "smartcaptcha", "smartwebsecurity", "spark",
        "speechsense", "storage", "trino", "video", "vpc", "ydb", "ytsaurus",
    ],
}
