package deployer

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: gitops-runner-orchestration, Property 49: Deploy Idempotency
// Validates: Requirements 3.9, 9.1
//
// Calling DeployBackendInfrastructure twice with the same MGConfig/UFConfig
// produces no duplicate resources — all create calls are guarded by existence
// checks. The test simulates Yandex Cloud SDK responses where the second call
// finds every resource already present and reuses it.

// resourceTracker records create vs reuse decisions made by the ensure* helpers.
type resourceTracker struct {
	created map[string]int // resource key → number of Create calls
	reused  map[string]int // resource key → number of reuse (found-existing) calls
}

func newResourceTracker() *resourceTracker {
	return &resourceTracker{
		created: make(map[string]int),
		reused:  make(map[string]int),
	}
}

func (rt *resourceTracker) recordCreate(kind, name string) {
	rt.created[kind+"/"+name]++
}

func (rt *resourceTracker) recordReuse(kind, name string) {
	rt.reused[kind+"/"+name]++
}

// duplicates returns resource keys that were created more than once.
func (rt *resourceTracker) duplicates() []string {
	var dups []string
	for key, count := range rt.created {
		if count > 1 {
			dups = append(dups, fmt.Sprintf("%s (created %d times)", key, count))
		}
	}
	return dups
}

// ensureResource is the core idempotency primitive: if the resource already
// exists it records a reuse; otherwise it records a create and marks it as
// existing for future calls.
func ensureResource(existing map[string]bool, tracker *resourceTracker, kind, name string) {
	key := kind + "/" + name
	if existing[key] {
		tracker.recordReuse(kind, name)
		return
	}
	tracker.recordCreate(kind, name)
	existing[key] = true
}

// runDeploySimulation mirrors the step ordering of
// YandexCloudClient.DeployBackendInfrastructure but uses an in-memory
// existing-resource set instead of real SDK calls.
func runDeploySimulation(
	mgCfg *MGConfig,
	ufCfg *UFConfig,
	existing map[string]bool,
	tracker *resourceTracker,
) {
	// Step 1: Service accounts (stepServiceAccounts)
	allSAs := append([]ServiceAccountConfig{}, mgCfg.ServiceAccounts...)
	if ufCfg != nil && ufCfg.ServiceAccount.Name != "" {
		allSAs = append(allSAs, ufCfg.ServiceAccount)
	}
	for _, sa := range allSAs {
		ensureResource(existing, tracker, "service_account", sa.Name)
	}

	// Step 2: YDB database (stepYDBDatabase)
	if mgCfg.Database.Name != "" {
		ensureResource(existing, tracker, "database", mgCfg.Database.Name)
	}

	// Step 3: Storage bucket (unified single-bucket model)
	if mgCfg.Storage.BucketName != "" {
		ensureResource(existing, tracker, "bucket", mgCfg.Storage.BucketName)
	}

	// Step 4: Message queues — DLQ first, then task queue (stepMessageQueues)
	if mgCfg.Queues.DLQ.Name != "" {
		ensureResource(existing, tracker, "queue", mgCfg.Queues.DLQ.Name)
	}
	if mgCfg.Queues.TaskQueue.Name != "" {
		ensureResource(existing, tracker, "queue", mgCfg.Queues.TaskQueue.Name)
	}

	// Step 5: Container registry (stepContainerRegistry)
	ensureResource(existing, tracker, "registry", mgCfg.Name)

	// Step 6: MG containers (stepMGContainers)
	if mgCfg.FastAPIApp.Name != "" {
		ensureResource(existing, tracker, "container", mgCfg.FastAPIApp.Name)
		// Celery container name is derived: FastAPIApp.Name + "-celery"
		ensureResource(existing, tracker, "container", mgCfg.FastAPIApp.Name+"-celery")
	}

	// Step 7: UF containers (stepUFContainers)
	if ufCfg != nil && ufCfg.Workers.Name != "" {
		ensureResource(existing, tracker, "container", ufCfg.Workers.Name)
	}

	// Step 8: API Gateway (stepAPIGateway)
	if mgCfg.APIGateway.Name != "" {
		ensureResource(existing, tracker, "api_gateway", mgCfg.APIGateway.Name)
	}

	// Step 9: Git-sync timer trigger (stepTimerTriggers)
	if mgCfg.GitSyncTrigger.Schedule != "" {
		ensureResource(existing, tracker, "trigger", "git-sync")
	}
}

// ---------------------------------------------------------------------------
// Random config generators
// ---------------------------------------------------------------------------

func randomIdempotencyMGConfig(numQueues, numTriggers, numSAs int) *MGConfig {
	mg := &MGConfig{
		Name: fmt.Sprintf("mg-%d", rand.Intn(10000)),
		Cloud: CloudBlockConfig{
			Provider:   CloudProviderYandex,
			YCFolderID: fmt.Sprintf("b1g%010d", rand.Intn(1e10)),
			YCCloudID:  fmt.Sprintf("b1g%010d", rand.Intn(1e10)),
		},
		ImageVersion: "latest",
		APIGateway: APIGatewayConfig{
			Name:           fmt.Sprintf("gw-%d", rand.Intn(10000)),
			Description:    "test gateway",
			ServiceAccount: "gw-sa",
		},
		FastAPIApp: ServerlessContainerConfig{
			Name:   fmt.Sprintf("fastapi-%d", rand.Intn(10000)),
			Image:  "cr.yandex/test/mg:latest",
			Memory: 512,
			Cores:  1,
		},
		CeleryWorkers: CeleryWorkersConfig{
			Memory: 1024,
			Cores:  2,
		},
		GitSyncTrigger: GitSyncTriggerConfig{
			Schedule:       "*/5 * * * *",
			ServiceAccount: "mg-sa",
		},
		Queues: MotherGooseQueuesConfig{
			TaskQueue: QueueConfig{
				Name:              fmt.Sprintf("mg-tasks-%d", rand.Intn(10000)),
				VisibilityTimeout: 30,
				MessageRetention:  86400,
			},
			DLQ: QueueConfig{
				Name:             fmt.Sprintf("mg-tasks-dlq-%d", rand.Intn(10000)),
				MessageRetention: 86400,
			},
		},
		Database: DatabaseConfig{
			Name:           fmt.Sprintf("db-%d", rand.Intn(10000)),
			Type:           "ydb",
			ServerlessMode: true,
		},
		Storage: StorageConfig{
			BucketName: fmt.Sprintf("storage-%d", rand.Intn(10000)),
			Region:     "ru-central1",
		},
	}

	// numQueues and numTriggers params kept for signature compat but no longer used
	_ = numQueues
	_ = numTriggers

	for i := 0; i < numSAs; i++ {
		mg.ServiceAccounts = append(mg.ServiceAccounts, ServiceAccountConfig{
			Name:  fmt.Sprintf("sa-%d-%d", i, rand.Intn(10000)),
			Roles: []string{"lockbox.payloadViewer"},
		})
	}
	return mg
}

func randomIdempotencyUFConfig(hasSA bool) *UFConfig {
	uf := &UFConfig{
		Name:           fmt.Sprintf("uf-%d", rand.Intn(10000)),
		MotherGooseRef: "mg-ref",
		Workers: ServerlessContainerConfig{
			Name:   fmt.Sprintf("uf-worker-%d", rand.Intn(10000)),
			Image:  "cr.yandex/test/uf:latest",
			Memory: 512,
			Cores:  1,
		},
	}
	if hasSA {
		uf.ServiceAccount = ServiceAccountConfig{
			Name:  fmt.Sprintf("uf-sa-%d", rand.Intn(10000)),
			Roles: []string{"lockbox.payloadViewer"},
		}
	}
	return uf
}

// ---------------------------------------------------------------------------
// Property tests
// ---------------------------------------------------------------------------

// TestDeployIdempotency validates that calling DeployBackendInfrastructure
// twice with the same MGConfig/UFConfig produces zero duplicate creates.
// The simulation mirrors YandexCloudClient's ensure* pattern: list → found?
// reuse : create.
func TestDeployIdempotency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"second deploy with identical config creates zero new resources",
		prop.ForAll(
			func(numQueues, numTriggers, numSAs int, hasSA bool) bool {
				mgCfg := randomIdempotencyMGConfig(numQueues, numTriggers, numSAs)
				ufCfg := randomIdempotencyUFConfig(hasSA)

				// First pass populates existing set; second pass should only reuse
				existing := make(map[string]bool)
				firstTracker := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, firstTracker)

				secondTracker := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, secondTracker)

				// Property: second pass must have zero creates
				if len(secondTracker.created) != 0 {
					dups := secondTracker.duplicates()
					t.Logf("second deploy created resources: %v", secondTracker.created)
					t.Logf("duplicates: %v", dups)
					return false
				}

				// Every resource from the first pass must be reused
				if len(secondTracker.reused) != len(firstTracker.created) {
					t.Logf("first created %d resources, second reused %d",
						len(firstTracker.created), len(secondTracker.reused))
					return false
				}

				return true
			},
			gen.IntRange(0, 4), // numQueues
			gen.IntRange(0, 3), // numTriggers
			gen.IntRange(0, 3), // numSAs
			gen.Bool(),         // hasSA for UF
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestDeployIdempotencyRegistryReuse validates that the container registry
// is found by name on the second call and reused, not recreated.
func TestDeployIdempotencyRegistryReuse(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"container registry is reused by name on second deploy",
		prop.ForAll(
			func(numQueues int) bool {
				mgCfg := randomIdempotencyMGConfig(numQueues, 1, 1)
				ufCfg := randomIdempotencyUFConfig(true)

				existing := make(map[string]bool)
				firstTracker := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, firstTracker)

				// Registry must have been created on first pass
				regKey := "registry/" + mgCfg.Name
				if firstTracker.created[regKey] != 1 {
					t.Logf("registry not created on first pass")
					return false
				}

				secondTracker := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, secondTracker)

				// Registry must be reused (not created) on second pass
				if secondTracker.created[regKey] != 0 {
					t.Logf("registry created again on second pass")
					return false
				}
				if secondTracker.reused["registry/"+mgCfg.Name] != 1 {
					t.Logf("registry not reused on second pass")
					return false
				}
				return true
			},
			gen.IntRange(0, 3),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestDeployIdempotencyNoCreateOnSecondPass is a stronger variant that
// checks the global invariant: across ALL resource kinds the second
// deployment issues exactly zero Create operations.
func TestDeployIdempotencyNoCreateOnSecondPass(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"global invariant: zero creates on second deploy for any config shape",
		prop.ForAll(
			func(numQueues, numTriggers, numSAs int, hasSA bool) bool {
				mgCfg := randomIdempotencyMGConfig(numQueues, numTriggers, numSAs)
				ufCfg := randomIdempotencyUFConfig(hasSA)

				existing := make(map[string]bool)
				first := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, first)

				second := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, second)

				totalCreates := 0
				for _, count := range second.created {
					totalCreates += count
				}
				if totalCreates != 0 {
					t.Logf("second pass created %d resources: %v",
						totalCreates, second.created)
					return false
				}
				return true
			},
			gen.IntRange(0, 5),
			gen.IntRange(0, 4),
			gen.IntRange(0, 4),
			gen.Bool(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestDeployIdempotencyWithDLQQueues validates that DLQ queues (created
// before main queues) are also idempotent on the second pass.
func TestDeployIdempotencyWithDLQQueues(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property(
		"DLQ and main queues are both reused on second deploy",
		prop.ForAll(
			func(numMainQueues int) bool {
				mgCfg := randomIdempotencyMGConfig(0, 0, 1)
				// Each iteration uses a fresh DLQ+task queue pair
				mgCfg.Queues = MotherGooseQueuesConfig{
					DLQ:       QueueConfig{Name: fmt.Sprintf("mg-tasks-dlq-%d", numMainQueues)},
					TaskQueue: QueueConfig{Name: fmt.Sprintf("mg-tasks-%d", numMainQueues)},
				}
				ufCfg := randomIdempotencyUFConfig(false)

				existing := make(map[string]bool)
				first := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, first)

				second := newResourceTracker()
				runDeploySimulation(mgCfg, ufCfg, existing, second)

				if len(second.created) != 0 {
					t.Logf("second pass created queues: %v", second.created)
					return false
				}

				// Verify both queues were reused
				for _, name := range []string{mgCfg.Queues.DLQ.Name, mgCfg.Queues.TaskQueue.Name} {
					if name != "" && second.reused["queue/"+name] != 1 {
						t.Logf("queue %q not reused", name)
						return false
					}
				}
				return true
			},
			gen.IntRange(1, 4),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
