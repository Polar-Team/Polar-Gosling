# MotherGoose API Client

This package provides a Go client for communicating with the MotherGoose API. The Gosling CLI uses this client to query deployment status and manage Egg configurations.

## Overview

The MotherGoose API client provides a type-safe interface for interacting with the MotherGoose backend server. It handles:

- HTTP communication with proper authentication
- Automatic retry logic with exponential backoff
- Context-aware request cancellation
- Comprehensive error handling

## Usage

### Creating a Client

```go
import "github.com/polar-gosling/gosling/internal/mothergoose"

// Basic client with default settings
client := mothergoose.NewClient(
    "https://api.mothergoose.example.com",
    "your-api-key",
)

// Client with custom options
client := mothergoose.NewClient(
    "https://api.mothergoose.example.com",
    "your-api-key",
    mothergoose.WithTimeout(60 * time.Second),
    mothergoose.WithMaxRetries(5),
)
```

### Getting Egg Status

```go
ctx := context.Background()
status, err := client.GetEggStatus(ctx, "my-app")
if err != nil {
    log.Fatalf("failed to get egg status: %v", err)
}

fmt.Printf("Egg: %s\n", status.EggName)
fmt.Printf("Config Hash: %s\n", status.ConfigHash)
fmt.Printf("Active Runners: %d\n", len(status.ActiveRunners))
```

### Listing All Eggs

```go
eggs, err := client.ListEggs(ctx)
if err != nil {
    log.Fatalf("failed to list eggs: %v", err)
}

for _, egg := range eggs {
    fmt.Printf("Egg: %s (Type: %s)\n", egg.Name, egg.Type)
}
```

### Creating or Updating an Egg

```go
config := &deployer.EggConfig{
    Name: "my-app",
    Type: deployer.RunnerTypeVM,
    Cloud: deployer.CloudConfig{
        Provider: deployer.CloudProviderYandex,
        Region:   "ru-central1-a",
    },
    Resources: deployer.ResourceConfig{
        CPU:    2,
        Memory: 4096,
        Disk:   20,
    },
}

err := client.CreateOrUpdateEgg(ctx, config)
if err != nil {
    log.Fatalf("failed to create/update egg: %v", err)
}
```

### Getting Deployment Plans

```go
// Get a specific plan
plan, err := client.GetDeploymentPlan(ctx, "my-app", "plan-123")
if err != nil {
    log.Fatalf("failed to get deployment plan: %v", err)
}

// List all plans for an egg
plans, err := client.ListDeploymentPlans(ctx, "my-app")
if err != nil {
    log.Fatalf("failed to list deployment plans: %v", err)
}
```

## Features

### Automatic Retry Logic

The client automatically retries failed requests with exponential backoff:

- Default: 3 retries (configurable via `WithMaxRetries`)
- Backoff: 1s, 2s, 4s, etc.
- Does not retry on 4xx errors (except 429 rate limit)
- Respects context cancellation

### Error Handling

The client provides detailed error information:

```go
status, err := client.GetEggStatus(ctx, "nonexistent-egg")
if err != nil {
    var httpErr *mothergoose.HTTPError
    if errors.As(err, &httpErr) {
        fmt.Printf("HTTP %d: %s\n", httpErr.StatusCode, httpErr.Body)
    }
}
```

### Context Support

All methods accept a context for cancellation and timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

status, err := client.GetEggStatus(ctx, "my-app")
```

## Interface

The client implements the `MotherGooseClient` interface:

```go
type MotherGooseClient interface {
    GetEggStatus(ctx context.Context, eggName string) (*EggStatus, error)
    ListEggs(ctx context.Context) ([]*deployer.EggConfig, error)
    CreateOrUpdateEgg(ctx context.Context, config *deployer.EggConfig) error
    GetDeploymentPlan(ctx context.Context, eggName, planID string) (*deployer.DeploymentPlan, error)
    ListDeploymentPlans(ctx context.Context, eggName string) ([]*deployer.DeploymentPlan, error)
}
```

This interface allows for easy mocking in tests.

## Testing

The package includes comprehensive unit tests covering:

- Client creation and configuration
- All API methods
- Error handling
- Retry logic
- Context cancellation
- HTTP error responses

Run tests:

```bash
go test ./internal/mothergoose/... -v
```

## Design Principles

1. **No Direct Database Access**: The Gosling CLI must use this client to communicate with MotherGoose. Direct database access is prohibited.

2. **Type Safety**: All requests and responses use strongly-typed structs from the `deployer` package.

3. **Resilience**: Automatic retries and proper error handling ensure robust operation in production.

4. **Context Awareness**: All operations respect context cancellation for proper resource management.

5. **Testability**: The interface-based design allows for easy mocking in tests.
