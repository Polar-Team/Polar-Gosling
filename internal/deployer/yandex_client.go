package deployer

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/polar-gosling/gosling/internal/spinner"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/access"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/containerregistry/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	apigwpb "github.com/yandex-cloud/go-genproto/yandex/cloud/serverless/apigateway/v1"
	containerspb "github.com/yandex-cloud/go-genproto/yandex/cloud/serverless/containers/v1"
	triggerspb "github.com/yandex-cloud/go-genproto/yandex/cloud/serverless/triggers/v1"
	ydbpb "github.com/yandex-cloud/go-genproto/yandex/cloud/ydb/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/protobuf/types/known/durationpb"
)

// serviceAccountInfo holds the resolved cloud ID and the roles bound to a service account.
type serviceAccountInfo struct {
	Name  string
	ID    string
	Roles []string
}

// YandexCloudClient wraps the Yandex Cloud Go SDK for deploying backend infrastructure.
// Individual runner deployment is handled by MotherGoose using OpenTofu.
type YandexCloudClient struct {
	sdk      *ycsdk.SDK
	folderID string
	cloudID  string

	// Populated during deployment for cross-step references
	serviceAccounts  []serviceAccountInfo // resolved SA entries (ID + bound roles)
	databaseEndpoint string
	registryID       string
	mgContainerID    string
	mgContainerURL   string
	apiGatewayURL    string
}

// NewYandexCloudClient creates a new Yandex Cloud client.
func NewYandexCloudClient(ctx context.Context, folderID, cloudID string) (*YandexCloudClient, error) {
	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Yandex Cloud SDK: %w", err)
	}
	return &YandexCloudClient{
		sdk:      sdk,
		folderID: folderID,
		cloudID:  cloudID,
	}, nil
}

// DeployBackendInfrastructure deploys all bootstrap infrastructure using the parsed MG/UF configs.
func (c *YandexCloudClient) DeployBackendInfrastructure(ctx context.Context, mgCfg *MGConfig, ufCfg *UFConfig) error {
	steps := []struct {
		msg string
		fn  func(context.Context, *MGConfig, *UFConfig) error
	}{
		{"Creating service accounts", c.stepServiceAccounts},
		{"Creating YDB serverless database", c.stepYDBDatabase},
		{"Creating storage buckets", c.stepStorageBuckets},
		{"Creating message queues", c.stepMessageQueues},
		{"Creating container registry and pushing images", c.stepContainerRegistry},
		{"Deploying MotherGoose containers", c.stepMGContainers},
		{"Deploying UglyFox containers", c.stepUFContainers},
		{"Creating API Gateway", c.stepAPIGateway},
		{"Creating timer triggers", c.stepTimerTriggers},
		{"Triggering initial Git sync", c.stepInitialSync},
	}
	for _, s := range steps {
		if err := spinner.Run(s.msg, func() error {
			return s.fn(ctx, mgCfg, ufCfg)
		}); err != nil {
			return err
		}
	}
	return nil
}

// saIDByName returns the cloud ID for the service account with the given name,
// or an empty string if not found.
func (c *YandexCloudClient) saIDByName(name string) string {
	for _, sa := range c.serviceAccounts {
		if sa.Name == name {
			return sa.ID
		}
	}
	return ""
}

// GetStatus retrieves the status of infrastructure resources.
func (c *YandexCloudClient) GetStatus(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("not yet implemented")
}

// ---------------------------------------------------------------------------
// 40.1 — Service account creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepServiceAccounts(ctx context.Context, mgCfg *MGConfig, ufCfg *UFConfig) error {
	accounts := append([]ServiceAccountConfig{}, mgCfg.ServiceAccounts...)
	if ufCfg.ServiceAccount.Name != "" {
		accounts = append(accounts, ufCfg.ServiceAccount)
	}
	for _, sa := range accounts {
		id, err := c.ensureServiceAccount(ctx, sa)
		if err != nil {
			return fmt.Errorf("service account %q: %w", sa.Name, err)
		}
		if err := c.ensureRoleBindings(ctx, id, sa.Roles); err != nil {
			return fmt.Errorf("roles for %q: %w", sa.Name, err)
		}
		c.serviceAccounts = append(c.serviceAccounts, serviceAccountInfo{
			Name:  sa.Name,
			ID:    id,
			Roles: sa.Roles,
		})
	}
	return nil
}

func (c *YandexCloudClient) ensureServiceAccount(ctx context.Context, sa ServiceAccountConfig) (string, error) {
	// List all SAs whose name starts with the config name as a prefix (e.g. "mg-sa-")
	resp, err := c.sdk.IAM().ServiceAccount().List(ctx, &iam.ListServiceAccountsRequest{
		FolderId: c.folderID,
	})
	if err != nil {
		return "", fmt.Errorf("list: %w", err)
	}

	prefix := sa.Name + "-"
	var matches []*iam.ServiceAccount
	for _, existing := range resp.ServiceAccounts {
		if strings.HasPrefix(existing.Name, prefix) {
			matches = append(matches, existing)
		}
	}

	switch len(matches) {
	case 0:
		// None found — create a new one with a 5-char random postfix
		postfix, err := randomHex(5)
		if err != nil {
			return "", fmt.Errorf("generate postfix: %w", err)
		}
		fullName := prefix + postfix
		op, err := c.sdk.WrapOperation(c.sdk.IAM().ServiceAccount().Create(ctx, &iam.CreateServiceAccountRequest{
			FolderId:    c.folderID,
			Name:        fullName,
			Description: sa.Description,
		}))
		if err != nil {
			return "", fmt.Errorf("create: %w", err)
		}
		if err := op.Wait(ctx); err != nil {
			return "", fmt.Errorf("wait: %w", err)
		}
		res, err := op.Response()
		if err != nil {
			return "", fmt.Errorf("response: %w", err)
		}
		return res.(*iam.ServiceAccount).Id, nil

	case 1:
		// Exactly one match — reuse it
		return matches[0].Id, nil

	default:
		// Multiple matches — ask the user to pick one
		return promptUserPickSA(sa.Name, matches)
	}
}

// randomHex returns n random hex characters (n bytes → 2n hex chars, truncated to n).
func randomHex(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

// promptUserPickSA prints the duplicate SA list and asks the user to choose one interactively.
func promptUserPickSA(configName string, matches []*iam.ServiceAccount) (string, error) {
	fmt.Fprintf(os.Stderr, "\nMultiple service accounts found with prefix %q:\n", configName+"-")
	for i, sa := range matches {
		fmt.Fprintf(os.Stderr, "  [%d] %s  (id: %s)\n", i+1, sa.Name, sa.Id)
	}
	fmt.Fprintf(os.Stderr, "Enter number to use: ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		var choice int
		if _, err := fmt.Sscanf(line, "%d", &choice); err != nil || choice < 1 || choice > len(matches) {
			fmt.Fprintf(os.Stderr, "Invalid choice, enter 1-%d: ", len(matches))
			continue
		}
		return matches[choice-1].Id, nil
	}
	return "", fmt.Errorf("no selection made for service account %q", configName)
}

func (c *YandexCloudClient) ensureRoleBindings(ctx context.Context, saID string, roles []string) error {
	if len(roles) == 0 {
		return nil
	}
	deltas := make([]*access.AccessBindingDelta, 0, len(roles))
	for _, role := range roles {
		deltas = append(deltas, &access.AccessBindingDelta{
			Action: access.AccessBindingAction_ADD,
			AccessBinding: &access.AccessBinding{
				RoleId: role,
				Subject: &access.Subject{
					Id:   saID,
					Type: "serviceAccount",
				},
			},
		})
	}
	op, err := c.sdk.WrapOperation(c.sdk.ResourceManager().Folder().UpdateAccessBindings(ctx, &access.UpdateAccessBindingsRequest{
		ResourceId:          c.folderID,
		AccessBindingDeltas: deltas,
	}))
	if err != nil {
		return fmt.Errorf("update bindings: %w", err)
	}
	return op.Wait(ctx)
}

// ---------------------------------------------------------------------------
// 40.2 — YDB serverless database creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepYDBDatabase(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	if mgCfg.Database.Name == "" {
		return nil
	}
	// Idempotent: check if already exists
	list, err := c.sdk.YDB().Database().List(ctx, &ydbpb.ListDatabasesRequest{FolderId: c.folderID})
	if err != nil {
		return fmt.Errorf("list YDB databases: %w", err)
	}
	for _, db := range list.Databases {
		if db.Name == mgCfg.Database.Name {
			c.databaseEndpoint = db.Endpoint
			return nil
		}
	}

	op, err := c.sdk.WrapOperation(c.sdk.YDB().Database().Create(ctx, &ydbpb.CreateDatabaseRequest{
		FolderId: c.folderID,
		Name:     mgCfg.Database.Name,
		DatabaseType: &ydbpb.CreateDatabaseRequest_ServerlessDatabase{
			ServerlessDatabase: &ydbpb.ServerlessDatabase{},
		},
	}))
	if err != nil {
		return fmt.Errorf("create YDB: %w", err)
	}
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := op.Wait(pollCtx); err != nil {
		return fmt.Errorf("wait YDB: %w", err)
	}
	res, err := op.Response()
	if err != nil {
		return fmt.Errorf("YDB response: %w", err)
	}
	c.databaseEndpoint = res.(*ydbpb.Database).Endpoint
	return nil
}

// ---------------------------------------------------------------------------
// 40.3 — Object Storage (S3-compatible) bucket creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepStorageBuckets(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	if mgCfg.Storage.BucketName == "" {
		return nil
	}

	s3c, err := c.newS3Client(ctx, mgCfg.Storage.Region)
	if err != nil {
		return err
	}

	return c.ensureBucket(ctx, s3c, mgCfg.Storage.BucketName)
}

func (c *YandexCloudClient) ensureBucket(ctx context.Context, client *s3.Client, name string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(name)})
	if err == nil {
		return nil
	}
	// Only attempt creation when the bucket genuinely does not exist.
	// HeadBucket returns HTTP 404 / NoSuchBucket for missing buckets;
	// any other error (403, timeout, throttle) should surface immediately.
	var respErr interface{ HTTPStatusCode() int }
	if ok := errors.As(err, &respErr); !ok || respErr.HTTPStatusCode() != http.StatusNotFound {
		return fmt.Errorf("check bucket %q: %w", name, err)
	}
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(name)})
	if err != nil {
		return fmt.Errorf("create bucket %q: %w", name, err)
	}
	return nil
}

func (c *YandexCloudClient) newS3Client(ctx context.Context, region string) (*s3.Client, error) {
	if region == "" {
		// Parser validation ensures region is set for parsed configs.
		// This fallback guards against manually-constructed MGConfig in
		// tests or internal packages, where region may be left empty.
		region = "ru-central1"
	}
	tok, err := c.sdk.CreateIAMToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("IAM token for S3: %w", err)
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(tok.IamToken, "", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("AWS config for YC S3: %w", err)
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://storage.yandexcloud.net")
		o.UsePathStyle = true
	}), nil
}

// ---------------------------------------------------------------------------
// 40.4 — YMQ message queue creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepMessageQueues(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	sqsc, err := c.newSQSClient(ctx)
	if err != nil {
		return err
	}

	// Create DLQ first, then task queue referencing it
	dlqURL := ""
	if mgCfg.Queues.DLQ.Name != "" {
		dlqURL, err = c.ensureQueue(ctx, sqsc, mgCfg.Queues.DLQ, "")
		if err != nil {
			return fmt.Errorf("dlq: %w", err)
		}
	}
	if mgCfg.Queues.TaskQueue.Name != "" {
		if _, err := c.ensureQueue(ctx, sqsc, mgCfg.Queues.TaskQueue, dlqURL); err != nil {
			return fmt.Errorf("task queue: %w", err)
		}
	}
	return nil
}

func (c *YandexCloudClient) ensureQueue(ctx context.Context, client *sqs.Client, mq QueueConfig, dlqURL string) (string, error) {
	existing, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(mq.Name)})
	if err == nil && existing.QueueUrl != nil {
		return *existing.QueueUrl, nil
	}
	attrs := map[string]string{}
	if mq.VisibilityTimeout > 0 {
		attrs["VisibilityTimeout"] = fmt.Sprintf("%d", mq.VisibilityTimeout)
	}
	if mq.MessageRetention > 0 {
		attrs["MessageRetentionPeriod"] = fmt.Sprintf("%d", mq.MessageRetention)
	}
	if mq.MaxMessageSize > 0 {
		attrs["MaximumMessageSize"] = fmt.Sprintf("%d", mq.MaxMessageSize)
	}
	if mq.ReceiveWaitTime > 0 {
		attrs["ReceiveMessageWaitTimeSeconds"] = fmt.Sprintf("%d", mq.ReceiveWaitTime)
	}
	if dlqURL != "" {
		attrs["RedrivePolicy"] = fmt.Sprintf(`{"deadLetterTargetArn":"%s","maxReceiveCount":"5"}`, dlqURL)
	}
	out, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName:  aws.String(mq.Name),
		Attributes: attrs,
	})
	if err != nil {
		return "", fmt.Errorf("create queue %q: %w", mq.Name, err)
	}
	return *out.QueueUrl, nil
}

func (c *YandexCloudClient) newSQSClient(ctx context.Context) (*sqs.Client, error) {
	tok, err := c.sdk.CreateIAMToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("IAM token for YMQ: %w", err)
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("ru-central1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(tok.IamToken, "", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("AWS config for YMQ: %w", err)
	}
	return sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String("https://message-queue.api.cloud.yandex.net")
	}), nil
}

// ---------------------------------------------------------------------------
// 40.5 — Container Registry creation and image push
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepContainerRegistry(ctx context.Context, mgCfg *MGConfig, ufCfg *UFConfig) error {
	registryName := mgCfg.Name

	list, err := c.sdk.ContainerRegistry().Registry().List(ctx, &containerregistry.ListRegistriesRequest{
		FolderId: c.folderID,
	})
	if err != nil {
		return fmt.Errorf("list registries: %w", err)
	}
	for _, r := range list.Registries {
		if r.Name == registryName {
			c.registryID = r.Id
			return c.pushImages(ctx, mgCfg, ufCfg)
		}
	}

	op, err := c.sdk.WrapOperation(c.sdk.ContainerRegistry().Registry().Create(ctx, &containerregistry.CreateRegistryRequest{
		FolderId: c.folderID,
		Name:     registryName,
	}))
	if err != nil {
		return fmt.Errorf("create registry: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("wait registry: %w", err)
	}
	res, err := op.Response()
	if err != nil {
		return fmt.Errorf("registry response: %w", err)
	}
	c.registryID = res.(*containerregistry.Registry).Id
	return c.pushImages(ctx, mgCfg, ufCfg)
}

func (c *YandexCloudClient) pushImages(ctx context.Context, mgCfg *MGConfig, ufCfg *UFConfig) error {
	version := mgCfg.ImageVersion
	if version == "" {
		version = "latest"
	}

	mgSource := fmt.Sprintf("%s:%s", GHCRMotherGooseImage, version)
	ufSource := fmt.Sprintf("%s:%s", GHCRUglyFoxImage, version)
	mgTarget := fmt.Sprintf("cr.yandex/%s/mothergoose:%s", c.registryID, version)
	ufTarget := fmt.Sprintf("cr.yandex/%s/uglyfox:%s", c.registryID, version)

	if err := dockerPullTagPush(ctx, mgSource, mgTarget); err != nil {
		return fmt.Errorf("mothergoose image: %w", err)
	}
	if err := dockerPullTagPush(ctx, ufSource, ufTarget); err != nil {
		return fmt.Errorf("uglyfox image: %w", err)
	}

	if mgCfg.FastAPIApp.Image == "" {
		mgCfg.FastAPIApp.Image = mgTarget
	}
	// CeleryWorkers inherits image from FastAPIApp — no separate image field
	if ufCfg != nil && ufCfg.Workers.Image == "" {
		ufCfg.Workers.Image = ufTarget
	}
	return nil
}

// dockerPullTagPush pulls a source image, re-tags it, and pushes to the target registry.
func dockerPullTagPush(ctx context.Context, source, target string) error {
	pull := exec.CommandContext(ctx, "docker", "pull", source)
	if out, err := pull.CombinedOutput(); err != nil {
		return fmt.Errorf("docker pull %s: %s: %w", source, string(out), err)
	}
	tag := exec.CommandContext(ctx, "docker", "tag", source, target)
	if out, err := tag.CombinedOutput(); err != nil {
		return fmt.Errorf("docker tag %s → %s: %s: %w", source, target, string(out), err)
	}
	push := exec.CommandContext(ctx, "docker", "push", target)
	if out, err := push.CombinedOutput(); err != nil {
		return fmt.Errorf("docker push %s: %s: %w", target, string(out), err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 40.6 — MotherGoose serverless container deployment
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepMGContainers(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	id, url, err := c.deployContainer(ctx, mgCfg.FastAPIApp)
	if err != nil {
		return fmt.Errorf("fastapi container: %w", err)
	}
	c.mgContainerID = id
	c.mgContainerURL = url

	// Celery workers share the same image, name prefix, and SA as FastAPIApp
	celeryContainer := ServerlessContainerConfig{
		Name:             mgCfg.FastAPIApp.Name + "-celery",
		Image:            mgCfg.FastAPIApp.Image,
		ServiceAccount:   mgCfg.FastAPIApp.ServiceAccount,
		Memory:           mgCfg.CeleryWorkers.Memory,
		Cores:            mgCfg.CeleryWorkers.Cores,
		CoreFraction:     mgCfg.CeleryWorkers.CoreFraction,
		ExecutionTimeout: mgCfg.CeleryWorkers.ExecutionTimeout,
		Concurrency:      mgCfg.CeleryWorkers.Concurrency,
		Environment:      mgCfg.FastAPIApp.Environment,
	}
	if celeryContainer.Memory == 0 {
		celeryContainer.Memory = mgCfg.FastAPIApp.Memory
	}
	if _, _, err := c.deployContainer(ctx, celeryContainer); err != nil {
		return fmt.Errorf("celery container: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 40.7 — UglyFox serverless container deployment
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepUFContainers(ctx context.Context, _ *MGConfig, ufCfg *UFConfig) error {
	if ufCfg.Workers.Name == "" {
		return nil
	}
	if _, _, err := c.deployContainer(ctx, ufCfg.Workers); err != nil {
		return fmt.Errorf("uglyfox container: %w", err)
	}
	return nil
}

// deployContainer creates (or finds) a serverless container and deploys a new revision.
// Returns (containerID, invokeURL, error).
func (c *YandexCloudClient) deployContainer(ctx context.Context, cfg ServerlessContainerConfig) (string, string, error) {
	if cfg.Name == "" {
		return "", "", nil
	}

	// Ensure container exists
	containerID, err := c.ensureContainer(ctx, cfg.Name)
	if err != nil {
		return "", "", err
	}

	// Build revision request
	mem := int64(cfg.Memory) * 1024 * 1024
	if mem == 0 {
		mem = 128 * 1024 * 1024
	}
	req := &containerspb.DeployContainerRevisionRequest{
		ContainerId: containerID,
		Resources: &containerspb.Resources{
			Memory: mem,
		},
		ImageSpec: &containerspb.ImageSpec{
			ImageUrl:    cfg.Image,
			Environment: cfg.Environment,
		},
	}
	if cfg.Cores > 0 {
		req.Resources.Cores = int64(cfg.Cores)
	}
	if cfg.CoreFraction > 0 {
		req.Resources.CoreFraction = int64(cfg.CoreFraction)
	}
	if saID := c.saIDByName(cfg.ServiceAccount); saID != "" {
		req.ServiceAccountId = saID
	}
	if cfg.Concurrency > 0 {
		req.Concurrency = int64(cfg.Concurrency)
	}
	if cfg.ExecutionTimeout != "" {
		if d, err := time.ParseDuration(cfg.ExecutionTimeout); err == nil {
			req.ExecutionTimeout = durationpb.New(d)
		}
	}

	op, err := c.sdk.WrapOperation(c.sdk.Serverless().Containers().Container().DeployRevision(ctx, req))
	if err != nil {
		return "", "", fmt.Errorf("deploy revision %q: %w", cfg.Name, err)
	}
	if err := op.Wait(ctx); err != nil {
		return "", "", fmt.Errorf("wait revision %q: %w", cfg.Name, err)
	}

	// Fetch container URL
	ct, err := c.sdk.Serverless().Containers().Container().Get(ctx, &containerspb.GetContainerRequest{
		ContainerId: containerID,
	})
	if err != nil {
		return containerID, "", fmt.Errorf("get container %q: %w", cfg.Name, err)
	}
	return containerID, ct.Url, nil
}

func (c *YandexCloudClient) ensureContainer(ctx context.Context, name string) (string, error) {
	list, err := c.sdk.Serverless().Containers().Container().List(ctx, &containerspb.ListContainersRequest{
		FolderId: c.folderID,
	})
	if err != nil {
		return "", fmt.Errorf("list containers: %w", err)
	}
	for _, ct := range list.Containers {
		if ct.Name == name {
			return ct.Id, nil
		}
	}

	op, err := c.sdk.WrapOperation(c.sdk.Serverless().Containers().Container().Create(ctx, &containerspb.CreateContainerRequest{
		FolderId: c.folderID,
		Name:     name,
	}))
	if err != nil {
		return "", fmt.Errorf("create container %q: %w", name, err)
	}
	if err := op.Wait(ctx); err != nil {
		return "", fmt.Errorf("wait container %q: %w", name, err)
	}
	res, err := op.Response()
	if err != nil {
		return "", fmt.Errorf("container response %q: %w", name, err)
	}
	return res.(*containerspb.Container).Id, nil
}

// ---------------------------------------------------------------------------
// 40.8 — API Gateway creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepAPIGateway(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	if mgCfg.APIGateway.Name == "" {
		return nil
	}

	// Idempotent: check if already exists
	list, err := c.sdk.Serverless().APIGateway().ApiGateway().List(ctx, &apigwpb.ListApiGatewayRequest{
		FolderId: c.folderID,
	})
	if err != nil {
		return fmt.Errorf("list API gateways: %w", err)
	}
	for _, gw := range list.ApiGateways {
		if gw.Name == mgCfg.APIGateway.Name {
			c.apiGatewayURL = "https://" + gw.Domain
			return nil
		}
	}

	specYAML := c.generateOpenAPISpec(mgCfg)

	saID := c.saIDByName(mgCfg.APIGateway.ServiceAccount)
	op, err := c.sdk.WrapOperation(c.sdk.Serverless().APIGateway().ApiGateway().Create(ctx, &apigwpb.CreateApiGatewayRequest{
		FolderId:    c.folderID,
		Name:        mgCfg.APIGateway.Name,
		Description: mgCfg.APIGateway.Description,
		Spec: &apigwpb.CreateApiGatewayRequest_OpenapiSpec{
			OpenapiSpec: specYAML,
		},
	}))
	if err != nil {
		return fmt.Errorf("create API gateway: %w", err)
	}
	_ = saID // reserved for future RBAC on gateway

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("wait API gateway: %w", err)
	}
	res, err := op.Response()
	if err != nil {
		return fmt.Errorf("API gateway response: %w", err)
	}
	gw := res.(*apigwpb.ApiGateway)
	c.apiGatewayURL = "https://" + gw.Domain
	return nil
}

// generateOpenAPISpec builds an OpenAPI 3.0 spec that routes webhook and
// internal endpoints to the MotherGoose serverless container.
func (c *YandexCloudClient) generateOpenAPISpec(mgCfg *MGConfig) string {
	saID := c.saIDByName(mgCfg.FastAPIApp.ServiceAccount)

	return fmt.Sprintf(`openapi: 3.0.0
info:
  title: %s
  version: "1.0"
paths:
  /webhooks/gitlab:
    post:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: webhookGitlab
      responses:
        '202':
          description: Accepted
  /internal/sync-git:
    post:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: syncGit
      responses:
        '202':
          description: Accepted
  /internal/health-check:
    post:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: healthCheck
      responses:
        '200':
          description: OK
  /eggs:
    get:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: listEggs
      responses:
        '200':
          description: OK
  /eggs/{name}:
    get:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: getEgg
      parameters:
        - name: name
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
  /runners:
    get:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: listRunners
      responses:
        '200':
          description: OK
  /health:
    get:
      x-yc-apigateway-integration:
        type: serverless_containers
        container_id: "%s"
        service_account_id: "%s"
      operationId: health
      responses:
        '200':
          description: OK
`,
		mgCfg.APIGateway.Name,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
		c.mgContainerID, saID,
	)
}

// ---------------------------------------------------------------------------
// 40.9 — Timer trigger creation
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepTimerTriggers(ctx context.Context, mgCfg *MGConfig, _ *UFConfig) error {
	tCfg := mgCfg.GitSyncTrigger
	if tCfg.Schedule == "" {
		return nil
	}

	const triggerName = "git-sync"

	// Idempotent: skip if already exists
	list, err := c.sdk.Serverless().Triggers().Trigger().List(ctx, &triggerspb.ListTriggersRequest{
		FolderId: c.folderID,
	})
	if err != nil {
		return fmt.Errorf("list triggers: %w", err)
	}
	for _, t := range list.Triggers {
		if t.Name == triggerName {
			return nil
		}
	}

	saID := c.saIDByName(tCfg.ServiceAccount)
	if saID == "" {
		for _, info := range c.serviceAccounts {
			saID = info.ID
			break
		}
	}

	op, err := c.sdk.WrapOperation(c.sdk.Serverless().Triggers().Trigger().Create(ctx, &triggerspb.CreateTriggerRequest{
		FolderId: c.folderID,
		Name:     triggerName,
		Rule: &triggerspb.Trigger_Rule{
			Rule: &triggerspb.Trigger_Rule_Timer{
				Timer: &triggerspb.Trigger_Timer{
					CronExpression: tCfg.Schedule,
					Action: &triggerspb.Trigger_Timer_InvokeContainerWithRetry{
						InvokeContainerWithRetry: &triggerspb.InvokeContainerWithRetry{
							ContainerId:      c.mgContainerID,
							Path:             "/internal/sync-git",
							ServiceAccountId: saID,
						},
					},
				},
			},
		},
	}))
	if err != nil {
		return fmt.Errorf("create git-sync trigger: %w", err)
	}
	return op.Wait(ctx)
}

// ---------------------------------------------------------------------------
// 40.10 — Initial Git sync trigger
// ---------------------------------------------------------------------------

func (c *YandexCloudClient) stepInitialSync(_ context.Context, _ *MGConfig, _ *UFConfig) error {
	if c.apiGatewayURL == "" {
		return fmt.Errorf("API gateway URL not set")
	}

	syncURL := c.apiGatewayURL + "/internal/sync-git"
	resp, err := http.Post(syncURL, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return fmt.Errorf("POST %s: %w", syncURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("sync-git returned %d", resp.StatusCode)
	}

	fmt.Printf("\nBootstrap complete.\n")
	fmt.Printf("API Gateway URL: %s\n", c.apiGatewayURL)
	fmt.Printf("Configure GitLab webhooks to: %s/webhooks/gitlab\n", c.apiGatewayURL)
	return nil
}
