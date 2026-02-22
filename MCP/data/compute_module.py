"""Polar-Gosling-Compute-Module data for the MCP server."""

from typing import Any

COMPUTE_MODULE_VARIABLES: dict[str, list[dict[str, Any]]] = {
    "aws": [
        {"name": "create_aws_resources", "type": "bool", "default": False, "description": "Enable AWS resource creation"},
        {"name": "aws_runner_type", "type": "string", "values": ["ec2", "ecs"], "description": "AWS runner deployment type: EC2 VM or ECS Fargate"},
        {"name": "aws_region", "type": "string", "description": "AWS region (e.g. us-east-1)"},
        {"name": "aws_instance_type", "type": "string", "default": "t3.medium", "description": "EC2 instance type (ec2 runner only)"},
        {"name": "aws_ami_ssm_param", "type": "string", "description": "SSM parameter path for the runner AMI ID"},
        {"name": "aws_subnet_id", "type": "string", "description": "VPC subnet ID for EC2/ECS placement"},
        {"name": "aws_security_group_ids", "type": "list(string)", "description": "Security group IDs to attach"},
        {"name": "aws_iam_instance_profile", "type": "string", "description": "IAM instance profile ARN for EC2"},
        {"name": "aws_ecs_cluster_arn", "type": "string", "description": "ECS cluster ARN (ecs runner only)"},
        {"name": "aws_ecs_task_cpu", "type": "number", "default": 512, "description": "ECS task CPU units (256/512/1024/2048/4096)"},
        {"name": "aws_ecs_task_memory", "type": "number", "default": 1024, "description": "ECS task memory in MiB"},
        {"name": "aws_container_image", "type": "string", "description": "Container image URI for ECS runner"},
        {"name": "aws_runner_name", "type": "string", "description": "GitLab runner name tag"},
        {"name": "aws_tags", "type": "map(string)", "default": {}, "description": "Additional AWS resource tags"},
        {"name": "aws_key_pair_name", "type": "string", "default": "", "description": "EC2 key pair name for SSH access"},
        {"name": "aws_root_volume_size", "type": "number", "default": 20, "description": "EC2 root EBS volume size in GB"},
    ],
    "yandex_cloud": [
        {"name": "create_yc_resources", "type": "bool", "default": False, "description": "Enable Yandex Cloud resource creation"},
        {"name": "yc_runner_type", "type": "string", "values": ["vm", "serverless"], "description": "YC runner type: Compute VM or Serverless Container"},
        {"name": "yc_folder_id", "type": "string", "description": "Yandex Cloud folder ID"},
        {"name": "yc_zone", "type": "string", "default": "ru-central1-a", "description": "Availability zone"},
        {"name": "yc_subnet_id", "type": "string", "description": "VPC subnet ID"},
        {"name": "yc_platform_id", "type": "string", "default": "standard-v3", "description": "VM platform (standard-v1/v2/v3)"},
        {"name": "yc_cores", "type": "number", "default": 2, "description": "Number of vCPUs for VM"},
        {"name": "yc_memory", "type": "number", "default": 2, "description": "RAM in GB for VM"},
        {"name": "yc_core_fraction", "type": "number", "default": 100, "description": "CPU core fraction % (20/50/100)"},
        {"name": "yc_disk_size", "type": "number", "default": 20, "description": "Boot disk size in GB"},
        {"name": "yc_disk_type", "type": "string", "default": "network-ssd", "description": "Disk type (network-hdd/network-ssd)"},
        {"name": "yc_image_family", "type": "string", "default": "ubuntu-2204-lts", "description": "OS image family"},
        {"name": "yc_container_image", "type": "string", "description": "Container image for Serverless Container"},
        {"name": "yc_serverless_memory", "type": "number", "default": 512, "description": "Serverless Container memory in MB"},
        {"name": "yc_serverless_cores", "type": "number", "default": 1, "description": "Serverless Container vCPU count"},
        {"name": "yc_service_account_id", "type": "string", "description": "Service account ID for the runner"},
        {"name": "yc_runner_name", "type": "string", "description": "Resource name prefix"},
        {"name": "yc_labels", "type": "map(string)", "default": {}, "description": "Yandex Cloud resource labels"},
        {"name": "yc_ssh_public_key", "type": "string", "default": "", "description": "SSH public key for VM access"},
        {"name": "yc_user_data", "type": "string", "default": "", "description": "Cloud-init user data script"},
    ],
    "common": [
        {"name": "gitlab_runner_token", "type": "string", "sensitive": True, "description": "GitLab runner registration token"},
        {"name": "gitlab_server_url", "type": "string", "default": "https://gitlab.com", "description": "GitLab instance URL"},
        {"name": "runner_tags", "type": "list(string)", "default": [], "description": "GitLab runner tags"},
        {"name": "runner_concurrent", "type": "number", "default": 1, "description": "Max concurrent jobs per runner"},
        {"name": "random_suffix_length", "type": "number", "default": 6, "description": "Length of random suffix appended to resource names"},
    ],
}

COMPUTE_MODULE_OUTPUTS: list[dict[str, Any]] = [
    {
        "name": "hostname",
        "type": "string",
        "description": "Public DNS hostname of the deployed runner. For EC2: public DNS name. For YC VM: FQDN. For serverless: container endpoint URL.",
        "sensitive": False,
    },
    {
        "name": "public_ip",
        "type": "string",
        "description": "Public IP address of the runner. Empty string for serverless runners that have no static IP.",
        "sensitive": False,
    },
    {
        "name": "private_ip",
        "type": "string",
        "description": "Private IP address within the VPC/subnet. Available for both VM and ECS task runners.",
        "sensitive": False,
    },
    {
        "name": "id",
        "type": "string",
        "description": "Cloud resource ID. EC2: instance ID (i-xxx). ECS: task ARN. YC VM: instance ID. YC Serverless: container ID.",
        "sensitive": False,
    },
]

COMPUTE_MODULE_PROVIDERS: dict[str, Any] = {
    "required_providers": {
        "yandex": {
            "source": "yandex-cloud/yandex",
            "version": ">= 0.170.0",
            "description": "Yandex Cloud provider for VM and Serverless Container resources",
        },
        "aws": {
            "source": "hashicorp/aws",
            "version": ">= 4.66",
            "description": "AWS provider for EC2 and ECS resources",
        },
        "random": {
            "source": "hashicorp/random",
            "version": ">= 3.4.3",
            "description": "Used to generate unique resource name suffixes",
        },
    },
    "terraform_required_version": ">= 1.3.5",
    "opentofu_compatible": True,
    "notes": [
        "Both aws and yandex providers are always declared but only activated when create_aws_resources or create_yc_resources is true.",
        "The random provider generates a suffix appended to all resource names to avoid collisions.",
        "Provider credentials are expected via standard env vars: AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or YC_TOKEN/YC_SERVICE_ACCOUNT_KEY_FILE.",
    ],
    "test_suites": [
        "tests/tests_aws_ecs_base/",
        "tests/tests_aws_vm_base/",
        "tests/tests_yc_serverless_base/",
        "tests/tests_yc_vm_base/",
    ],
}
