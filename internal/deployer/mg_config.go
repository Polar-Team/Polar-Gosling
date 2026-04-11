package deployer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/polar-gosling/gosling/internal/parser"
)

// CloudBlockConfig represents cloud provider configuration for MG/UF blocks
type CloudBlockConfig struct {
	Provider     CloudProvider
	YCFolderID   string
	YCCloudID    string
	AWSRegion    string
	AWSAccountID string
}

// APIGatewayConfig represents API Gateway configuration
type APIGatewayConfig struct {
	Name           string
	Description    string
	SpecPath       string
	ServiceAccount string
}

// ServerlessContainerConfig represents the main (FastAPI) serverless container.
// Name, Image, and ServiceAccount are required — they identify the container.
type ServerlessContainerConfig struct {
	Name             string
	Image            string
	Memory           int    // MB: 128–8192 (YC) / 512–30720 (Fargate)
	Cores            int    // vCPU: 0 means use core_fraction only (YC); 1/2/4/8 (Fargate)
	CoreFraction     int    // % of vCPU: 5/10/20/50/100 (YC only); must be 1–100
	ExecutionTimeout string // duration string, e.g. "300s"
	Concurrency      int    // 1–16 (YC); ignored on Fargate
	ServiceAccount   string
	Environment      map[string]string
}

// CeleryWorkersConfig holds only the tunable resource settings for the Celery
// worker container. Name, Image, and ServiceAccount are inherited from FastAPIApp
// at deploy time — they share the same container image and SA.
type CeleryWorkersConfig struct {
	Memory           int // MB: same valid ranges as ServerlessContainerConfig
	Cores            int
	CoreFraction     int    // % of vCPU, 1–100
	ExecutionTimeout string // duration string
	Concurrency      int
}

// GitSyncTriggerConfig holds the schedule settings for the built-in git-sync
// cloud timer trigger. The endpoint and method are fixed (/internal/sync-git POST).
type GitSyncTriggerConfig struct {
	Schedule       string // cron expression, e.g. "*/5 * * * *"
	ServiceAccount string // SA name used to invoke the container
}

// QueueConfig holds settings for a single SQS/YMQ queue.
type QueueConfig struct {
	Name              string
	VisibilityTimeout int
	MessageRetention  int
	MaxMessageSize    int
	ReceiveWaitTime   int
}

// MotherGooseQueuesConfig holds the task queue and its dead-letter queue settings.
// The DLQ is always created first; the task queue references it via RedrivePolicy.
type MotherGooseQueuesConfig struct {
	TaskQueue QueueConfig
	DLQ       QueueConfig
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Name           string
	Type           string
	ServerlessMode bool
}

// StorageConfig represents S3/object storage configuration
type StorageConfig struct {
	BucketName string
	Region     string
}

// ServiceAccountConfig represents a service account configuration
type ServiceAccountConfig struct {
	Name        string
	Description string
	Roles       []string
}

// MGConfig represents a parsed MotherGoose infrastructure configuration
type MGConfig struct {
	Name            string
	Cloud           CloudBlockConfig
	ImageVersion    string // Container image version tag (default: "latest")
	APIGateway      APIGatewayConfig
	FastAPIApp      ServerlessContainerConfig
	CeleryWorkers   CeleryWorkersConfig
	GitSyncTrigger  GitSyncTriggerConfig
	Queues          MotherGooseQueuesConfig
	Database        DatabaseConfig
	Storage         StorageConfig
	ServiceAccounts []ServiceAccountConfig
}

// UFConfig represents a parsed UglyFox configuration
type UFConfig struct {
	Name           string
	MotherGooseRef string
	Cloud          CloudBlockConfig
	Workers        ServerlessContainerConfig
	ServiceAccount ServiceAccountConfig
}

// ParseMGDirectory scans all *.fly files in dirPath and returns all mothergoose block configs.
func ParseMGDirectory(dirPath string) ([]*MGConfig, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.fly"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan MG directory %s: %w", dirPath, err)
	}

	var configs []*MGConfig
	for _, file := range files {
		parsed, err := parseMGFile(file)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", file, err)
		}
		configs = append(configs, parsed...)
	}
	return configs, nil
}

// ParseUFDirectory scans all *.fly files in dirPath and returns all uglyfox block configs.
// Each uglyfox block's mothergoose reference is resolved against mgConfigs.
func ParseUFDirectory(dirPath string, mgConfigs []*MGConfig) ([]*UFConfig, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.fly"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan UF directory %s: %w", dirPath, err)
	}

	mgMap := make(map[string]*MGConfig, len(mgConfigs))
	for _, mg := range mgConfigs {
		mgMap[mg.Name] = mg
	}

	var configs []*UFConfig
	for _, file := range files {
		parsed, err := parseUFFile(file, mgMap)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", file, err)
		}
		configs = append(configs, parsed...)
	}
	return configs, nil
}

func parseMGFile(filename string) ([]*MGConfig, error) {
	p := parser.NewParser()
	config, err := p.ParseFile(filename)
	if err != nil {
		return nil, err
	}

	var results []*MGConfig
	for i := range config.Blocks {
		if config.Blocks[i].Type != "mothergoose" {
			continue
		}
		mg, err := parseMGBlock(&config.Blocks[i])
		if err != nil {
			return nil, err
		}
		results = append(results, mg)
	}
	return results, nil
}

func parseUFFile(filename string, mgMap map[string]*MGConfig) ([]*UFConfig, error) {
	p := parser.NewParser()
	config, err := p.ParseFile(filename)
	if err != nil {
		return nil, err
	}

	var results []*UFConfig
	for i := range config.Blocks {
		if config.Blocks[i].Type != "uglyfox" {
			continue
		}
		uf, err := parseUFBlock(&config.Blocks[i], mgMap)
		if err != nil {
			return nil, err
		}
		results = append(results, uf)
	}
	return results, nil
}

func parseMGBlock(block *parser.Block) (*MGConfig, error) {
	if len(block.Labels) == 0 {
		return nil, fmt.Errorf("mothergoose block must have a name label")
	}

	mg := &MGConfig{Name: block.Labels[0]}

	cloudBlock, ok := block.GetBlock("cloud")
	if !ok {
		return nil, fmt.Errorf("mothergoose %q: missing required cloud block", mg.Name)
	}
	cloud, err := parseCloudBlockConfig(cloudBlock)
	if err != nil {
		return nil, fmt.Errorf("mothergoose %q: %w", mg.Name, err)
	}
	if err := validateCloudBlockConfig(cloud); err != nil {
		return nil, fmt.Errorf("mothergoose %q: %w", mg.Name, err)
	}
	mg.Cloud = cloud

	if v, ok := block.GetAttribute("image_version"); ok {
		s, err := v.AsString()
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q: invalid image_version: %w", mg.Name, err)
		}
		mg.ImageVersion = s
	}

	if b, ok := block.GetBlock("api_gateway"); ok {
		mg.APIGateway, err = parseAPIGatewayBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q api_gateway: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("fastapi_app"); ok {
		mg.FastAPIApp, err = parseServerlessContainerBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q fastapi_app: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("celery_workers"); ok {
		mg.CeleryWorkers, err = parseCeleryWorkersBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q celery_workers: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("git_sync_trigger"); ok {
		mg.GitSyncTrigger, err = parseGitSyncTriggerBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q git_sync_trigger: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("mothergoose_queues"); ok {
		mg.Queues, err = parseMotherGooseQueuesBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q mothergoose_queues: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("database"); ok {
		mg.Database, err = parseDatabaseBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q database: %w", mg.Name, err)
		}
	}

	if b, ok := block.GetBlock("storage"); ok {
		mg.Storage, err = parseStorageBlock(b)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q storage: %w", mg.Name, err)
		}
	}

	for _, saBlock := range block.GetBlocks("service_account") {
		sa, err := parseServiceAccountBlock(&saBlock)
		if err != nil {
			return nil, fmt.Errorf("mothergoose %q service_account: %w", mg.Name, err)
		}
		mg.ServiceAccounts = append(mg.ServiceAccounts, sa)
	}

	return mg, nil
}

func parseUFBlock(block *parser.Block, mgMap map[string]*MGConfig) (*UFConfig, error) {
	if len(block.Labels) == 0 {
		return nil, fmt.Errorf("uglyfox block must have a name label")
	}

	uf := &UFConfig{Name: block.Labels[0]}

	mgRefVal, ok := block.GetAttribute("mothergoose")
	if !ok {
		return nil, fmt.Errorf("uglyfox %q: missing required 'mothergoose' attribute", uf.Name)
	}
	mgRef, err := mgRefVal.AsString()
	if err != nil {
		return nil, fmt.Errorf("uglyfox %q: invalid mothergoose reference: %w", uf.Name, err)
	}
	uf.MotherGooseRef = mgRef

	mg, found := mgMap[mgRef]
	if !found {
		return nil, fmt.Errorf("uglyfox %q: referenced mothergoose instance %q not found", uf.Name, mgRef)
	}
	uf.Cloud = mg.Cloud

	if b, ok := block.GetBlock("workers"); ok {
		uf.Workers, err = parseServerlessContainerBlock(b)
		if err != nil {
			return nil, fmt.Errorf("uglyfox %q workers: %w", uf.Name, err)
		}
	}

	saBlocks := block.GetBlocks("service_account")
	if len(saBlocks) > 0 {
		sa, err := parseServiceAccountBlock(&saBlocks[0])
		if err != nil {
			return nil, fmt.Errorf("uglyfox %q service_account: %w", uf.Name, err)
		}
		uf.ServiceAccount = sa
	}

	return uf, nil
}

// parseCloudBlockConfig parses a cloud block into CloudBlockConfig
func parseCloudBlockConfig(block *parser.Block) (CloudBlockConfig, error) {
	c := CloudBlockConfig{}

	providerVal, ok := block.GetAttribute("provider")
	if !ok {
		return c, fmt.Errorf("cloud block missing required 'provider' attribute")
	}
	providerStr, err := providerVal.AsString()
	if err != nil {
		return c, fmt.Errorf("invalid provider: %w", err)
	}
	provider, err := parseCloudProvider(providerStr)
	if err != nil {
		return c, err
	}
	c.Provider = provider

	if v, ok := block.GetAttribute("yc_folder_id"); ok {
		s, err := v.AsString()
		if err != nil {
			return c, fmt.Errorf("invalid yc_folder_id: %w", err)
		}
		c.YCFolderID = s
	}
	if v, ok := block.GetAttribute("yc_cloud_id"); ok {
		s, err := v.AsString()
		if err != nil {
			return c, fmt.Errorf("invalid yc_cloud_id: %w", err)
		}
		c.YCCloudID = s
	}
	if v, ok := block.GetAttribute("aws_region"); ok {
		s, err := v.AsString()
		if err != nil {
			return c, fmt.Errorf("invalid aws_region: %w", err)
		}
		c.AWSRegion = s
	}
	if v, ok := block.GetAttribute("aws_account_id"); ok {
		s, err := v.AsString()
		if err != nil {
			return c, fmt.Errorf("invalid aws_account_id: %w", err)
		}
		c.AWSAccountID = s
	}

	return c, nil
}

// validateCloudBlockConfig checks provider-specific required fields
func validateCloudBlockConfig(c CloudBlockConfig) error {
	switch c.Provider {
	case CloudProviderYandex:
		if strings.TrimSpace(c.YCFolderID) == "" {
			return fmt.Errorf("yandex cloud requires non-empty yc_folder_id")
		}
		if strings.TrimSpace(c.YCCloudID) == "" {
			return fmt.Errorf("yandex cloud requires non-empty yc_cloud_id")
		}
	case CloudProviderAWS:
		if strings.TrimSpace(c.AWSRegion) == "" {
			return fmt.Errorf("aws cloud requires non-empty aws_region")
		}
	}
	return nil
}

func parseAPIGatewayBlock(block *parser.Block) (APIGatewayConfig, error) {
	a := APIGatewayConfig{}
	if v, ok := block.GetAttribute("name"); ok {
		s, err := v.AsString()
		if err != nil {
			return a, fmt.Errorf("invalid name: %w", err)
		}
		a.Name = s
	}
	if v, ok := block.GetAttribute("description"); ok {
		s, err := v.AsString()
		if err != nil {
			return a, fmt.Errorf("invalid description: %w", err)
		}
		a.Description = s
	}
	if v, ok := block.GetAttribute("spec_path"); ok {
		s, err := v.AsString()
		if err != nil {
			return a, fmt.Errorf("invalid spec_path: %w", err)
		}
		a.SpecPath = s
	}
	if v, ok := block.GetAttribute("service_account"); ok {
		s, err := v.AsString()
		if err != nil {
			return a, fmt.Errorf("invalid service_account: %w", err)
		}
		a.ServiceAccount = s
	}
	return a, nil
}

func parseServerlessContainerBlock(block *parser.Block) (ServerlessContainerConfig, error) {
	sc := ServerlessContainerConfig{Environment: make(map[string]string)}
	if v, ok := block.GetAttribute("name"); ok {
		s, err := v.AsString()
		if err != nil {
			return sc, fmt.Errorf("invalid name: %w", err)
		}
		sc.Name = s
	}
	if v, ok := block.GetAttribute("image"); ok {
		s, err := v.AsString()
		if err != nil {
			return sc, fmt.Errorf("invalid image: %w", err)
		}
		sc.Image = s
	}
	if v, ok := block.GetAttribute("memory"); ok {
		n, err := v.AsInt()
		if err != nil {
			return sc, fmt.Errorf("invalid memory: %w", err)
		}
		sc.Memory = n
	}
	if v, ok := block.GetAttribute("cores"); ok {
		n, err := v.AsInt()
		if err != nil {
			return sc, fmt.Errorf("invalid cores: %w", err)
		}
		sc.Cores = n
	}
	if v, ok := block.GetAttribute("core_fraction"); ok {
		n, err := v.AsInt()
		if err != nil {
			return sc, fmt.Errorf("invalid core_fraction: %w", err)
		}
		sc.CoreFraction = n
	}
	if v, ok := block.GetAttribute("execution_timeout"); ok {
		s, err := v.AsString()
		if err != nil {
			return sc, fmt.Errorf("invalid execution_timeout: %w", err)
		}
		sc.ExecutionTimeout = s
	}
	if v, ok := block.GetAttribute("concurrency"); ok {
		n, err := v.AsInt()
		if err != nil {
			return sc, fmt.Errorf("invalid concurrency: %w", err)
		}
		sc.Concurrency = n
	}
	if v, ok := block.GetAttribute("service_account"); ok {
		s, err := v.AsString()
		if err != nil {
			return sc, fmt.Errorf("invalid service_account: %w", err)
		}
		sc.ServiceAccount = s
	}
	if envBlock, ok := block.GetBlock("environment"); ok {
		env, err := parseEnvironmentBlock(envBlock)
		if err != nil {
			return sc, err
		}
		sc.Environment = env
	}
	return sc, nil
}

func parseCeleryWorkersBlock(block *parser.Block) (CeleryWorkersConfig, error) {
	c := CeleryWorkersConfig{}
	if v, ok := block.GetAttribute("memory"); ok {
		n, err := v.AsInt()
		if err != nil {
			return c, fmt.Errorf("invalid memory: %w", err)
		}
		c.Memory = n
	}
	if v, ok := block.GetAttribute("cores"); ok {
		n, err := v.AsInt()
		if err != nil {
			return c, fmt.Errorf("invalid cores: %w", err)
		}
		c.Cores = n
	}
	if v, ok := block.GetAttribute("core_fraction"); ok {
		n, err := v.AsInt()
		if err != nil {
			return c, fmt.Errorf("invalid core_fraction: %w", err)
		}
		c.CoreFraction = n
	}
	if v, ok := block.GetAttribute("execution_timeout"); ok {
		s, err := v.AsString()
		if err != nil {
			return c, fmt.Errorf("invalid execution_timeout: %w", err)
		}
		c.ExecutionTimeout = s
	}
	if v, ok := block.GetAttribute("concurrency"); ok {
		n, err := v.AsInt()
		if err != nil {
			return c, fmt.Errorf("invalid concurrency: %w", err)
		}
		c.Concurrency = n
	}
	return c, nil
}

func parseGitSyncTriggerBlock(block *parser.Block) (GitSyncTriggerConfig, error) {
	t := GitSyncTriggerConfig{}
	if v, ok := block.GetAttribute("schedule"); ok {
		s, err := v.AsString()
		if err != nil {
			return t, fmt.Errorf("invalid schedule: %w", err)
		}
		t.Schedule = s
	}
	if v, ok := block.GetAttribute("service_account"); ok {
		s, err := v.AsString()
		if err != nil {
			return t, fmt.Errorf("invalid service_account: %w", err)
		}
		t.ServiceAccount = s
	}
	return t, nil
}

func parseQueueBlock(block *parser.Block) (QueueConfig, error) {
	q := QueueConfig{}
	if v, ok := block.GetAttribute("name"); ok {
		s, err := v.AsString()
		if err != nil {
			return q, fmt.Errorf("invalid name: %w", err)
		}
		q.Name = s
	}
	if v, ok := block.GetAttribute("visibility_timeout"); ok {
		n, err := v.AsInt()
		if err != nil {
			return q, fmt.Errorf("invalid visibility_timeout: %w", err)
		}
		q.VisibilityTimeout = n
	}
	if v, ok := block.GetAttribute("message_retention"); ok {
		n, err := v.AsInt()
		if err != nil {
			return q, fmt.Errorf("invalid message_retention: %w", err)
		}
		q.MessageRetention = n
	}
	if v, ok := block.GetAttribute("max_message_size"); ok {
		n, err := v.AsInt()
		if err != nil {
			return q, fmt.Errorf("invalid max_message_size: %w", err)
		}
		q.MaxMessageSize = n
	}
	if v, ok := block.GetAttribute("receive_wait_time"); ok {
		n, err := v.AsInt()
		if err != nil {
			return q, fmt.Errorf("invalid receive_wait_time: %w", err)
		}
		q.ReceiveWaitTime = n
	}
	return q, nil
}

func parseMotherGooseQueuesBlock(block *parser.Block) (MotherGooseQueuesConfig, error) {
	q := MotherGooseQueuesConfig{}
	if b, ok := block.GetBlock("task_queue"); ok {
		tq, err := parseQueueBlock(b)
		if err != nil {
			return q, fmt.Errorf("task_queue: %w", err)
		}
		q.TaskQueue = tq
	}
	if b, ok := block.GetBlock("dlq"); ok {
		dlq, err := parseQueueBlock(b)
		if err != nil {
			return q, fmt.Errorf("dlq: %w", err)
		}
		q.DLQ = dlq
	}
	return q, nil
}

func parseDatabaseBlock(block *parser.Block) (DatabaseConfig, error) {
	d := DatabaseConfig{}
	if v, ok := block.GetAttribute("name"); ok {
		s, err := v.AsString()
		if err != nil {
			return d, fmt.Errorf("invalid name: %w", err)
		}
		d.Name = s
	}
	if v, ok := block.GetAttribute("type"); ok {
		s, err := v.AsString()
		if err != nil {
			return d, fmt.Errorf("invalid type: %w", err)
		}
		d.Type = s
	}
	if v, ok := block.GetAttribute("serverless_mode"); ok {
		b, err := v.AsBool()
		if err != nil {
			return d, fmt.Errorf("invalid serverless_mode: %w", err)
		}
		d.ServerlessMode = b
	}
	return d, nil
}

func parseStorageBlock(block *parser.Block) (StorageConfig, error) {
	s := StorageConfig{}

	// Detect legacy sub-bucket definitions and return migration error (Req 25.14)
	var legacyBlocks []string
	if _, ok := block.GetBlock("state_bucket"); ok {
		legacyBlocks = append(legacyBlocks, "state_bucket")
	}
	if _, ok := block.GetBlock("binary_bucket"); ok {
		legacyBlocks = append(legacyBlocks, "binary_bucket")
	}
	if len(legacyBlocks) > 0 {
		return s, fmt.Errorf("legacy storage format detected: %s sub-block(s) no longer supported. "+
			"Migrate to the unified single-bucket format:\n\n  storage {\n    bucket_name = \"my-bucket\"\n    region      = \"ru-central1\"\n  }",
			strings.Join(legacyBlocks, ", "))
	}

	if v, ok := block.GetAttribute("bucket_name"); ok {
		str, err := v.AsString()
		if err != nil {
			return s, fmt.Errorf("invalid bucket_name: %w", err)
		}
		s.BucketName = str
	}
	if v, ok := block.GetAttribute("region"); ok {
		str, err := v.AsString()
		if err != nil {
			return s, fmt.Errorf("invalid region: %w", err)
		}
		s.Region = str
	}

	// Cross-field validation: bucket_name and region must both be present or both absent
	if (s.BucketName != "") != (s.Region != "") {
		return s, fmt.Errorf("storage block requires both 'bucket_name' and 'region' attributes")
	}

	return s, nil
}

func parseServiceAccountBlock(block *parser.Block) (ServiceAccountConfig, error) {
	sa := ServiceAccountConfig{}

	// Name comes from the block label: service_account "my-sa" { ... }
	if len(block.Labels) > 0 {
		sa.Name = block.Labels[0]
	} else if v, ok := block.GetAttribute("name"); ok {
		// Fallback: legacy flat attribute format
		s, err := v.AsString()
		if err != nil {
			return sa, fmt.Errorf("invalid name: %w", err)
		}
		sa.Name = s
	}

	if v, ok := block.GetAttribute("description"); ok {
		s, err := v.AsString()
		if err != nil {
			return sa, fmt.Errorf("invalid description: %w", err)
		}
		sa.Description = s
	}
	if v, ok := block.GetAttribute("roles"); ok {
		rolesList, err := v.AsList()
		if err != nil {
			return sa, fmt.Errorf("invalid roles: %w", err)
		}
		roles := make([]string, len(rolesList))
		for i, rv := range rolesList {
			s, err := rv.AsString()
			if err != nil {
				return sa, fmt.Errorf("invalid role at index %d: %w", i, err)
			}
			roles[i] = s
		}
		sa.Roles = roles
	}
	return sa, nil
}
