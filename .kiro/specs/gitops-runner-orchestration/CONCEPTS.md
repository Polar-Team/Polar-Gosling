# GitOps Runner Orchestration: Core Concepts

## Repository Management Strategies

The system provides three ways to configure runners for GitLab repositories:

### 1. Single Egg (Project-Level)
**One repository, one runner configuration**

```
┌─────────────────────────────────────┐
│ Egg: "my-app"                       │
│                                     │
│ gitlab {                            │
│   server = "gitlab.com"             │
│   project_id = 12345                │
│ }                                   │
│                                     │
│ ┌─────────────────────────────────┐ │
│ │ GitLab Project: my-app          │ │
│ │ ID: 12345                       │ │
│ │ Runners: Dedicated              │ │
│ └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

**Use when**: Single repository needs dedicated runners

---

### 2. Group-Level Egg
**One GitLab group, automatic discovery of all projects**

```
┌─────────────────────────────────────────────────────────┐
│ Egg: "platform-team"                                    │
│                                                         │
│ gitlab {                                                │
│   server = "gitlab.company.com"                         │
│   group_id = 789                                        │
│ }                                                       │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ GitLab Group: platform-team (ID: 789)               │ │
│ │                                                     │ │
│ │ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐   │ │
│ │ │ Project A   │ │ Project B   │ │ Project C   │   │ │
│ │ │ ID: 100     │ │ ID: 101     │ │ ID: 102     │   │ │
│ │ └─────────────┘ └─────────────┘ └─────────────┘   │ │
│ │                                                     │ │
│ │ All projects automatically discovered via API       │ │
│ │ Runners: Shared across all group projects          │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**Use when**: 
- All repositories in a GitLab group
- Want automatic discovery
- Minimal maintenance

---

### 3. EggsBucket
**Manual selection of specific repositories or groups (can span groups/instances)**

```
┌───────────────────────────────────────────────────────────────┐
│ EggsBucket: "selected-microservices"                          │
│                                                               │
│ repositories {                                                │
│   repo "api" { gitlab { server = "gitlab.com", id = 12345 }} │
│   repo "team-a" { gitlab { server = "internal.com", gid = 67 }}│
│   repo "pay" { gitlab { server = "gitlab.com", id = 99999 }} │
│ }                                                             │
│                                                               │
│ ┌───────────────┐  ┌───────────────┐  ┌───────────────┐     │
│ │ gitlab.com    │  │ internal.com  │  │ gitlab.com    │     │
│ │               │  │               │  │               │     │
│ │ ┌───────────┐ │  │ ┌───────────┐ │  │ ┌───────────┐ │     │
│ │ │ api-svc   │ │  │ │ Group: 67 │ │  │ │ payment   │ │     │
│ │ │ ID: 12345 │ │  │ │ ├─ proj-1 │ │  │ │ ID: 99999 │ │     │
│ │ └───────────┘ │  │ │ ├─ proj-2 │ │  │ └───────────┘ │     │
│ └───────────────┘  │ │ └─ proj-3 │ │  └───────────────┘     │
│                    │ └───────────┘ │                         │
│                    └───────────────┘                         │
│                                                               │
│ Manually curated list of repositories AND groups              │
│ Can span different GitLab instances and groups                │
│ Each repo can be a project_id OR group_id                     │
│ Runners: Shared across selected projects/groups               │
└───────────────────────────────────────────────────────────────┘
```

**Use when**:
- Cherry-picking specific repositories OR entire groups
- Mixing repos from different GitLab instances
- Need fine-grained control
- Want to combine individual projects with entire groups

---

## Decision Tree

```
Do you need to manage multiple repositories?
│
├─ NO → Use Single Egg (Project-Level)
│       ✓ Simple configuration
│       ✓ Dedicated runners
│
└─ YES → Are all repositories in one GitLab group?
         │
         ├─ YES → Use Group-Level Egg
         │        ✓ Automatic discovery
         │        ✓ Zero maintenance
         │        ✓ Auto-includes new repos
         │
         └─ NO → Use EggsBucket
                  ✓ Manual selection
                  ✓ Cross-instance support
                  ✓ Fine-grained control
```

---

## Secret Management

All three approaches use **automatic secret management**:

```
Secret Path Convention:
yc-lockbox://gitlab/{server}/{egg-name}/{secret-type}

Examples:
- yc-lockbox://gitlab/gitlab.com/my-app/api-token
- yc-lockbox://gitlab/gitlab.com/my-app/runner-token
- yc-lockbox://gitlab/gitlab.com/my-app/webhook-secret
```

**No need to specify secrets in .fly files!**

The system automatically:
1. Generates secret paths based on egg name and server
2. Retrieves secrets from secret storage
3. Configures GitLab webhooks and runners
4. Masks secrets in all logs and outputs

---

## Configuration Comparison

### Single Egg
```hcl
egg "my-app" {
  gitlab {
    server = "gitlab.com"
    project_id = 12345
  }
}
```

### Group-Level Egg
```hcl
egg "platform-team" {
  gitlab {
    server = "gitlab.company.com"
    group_id = 789
  }
}
```

### EggsBucket
```hcl
eggsbucket "selected-services" {
  repositories {
    repo "api" {
      gitlab {
        server = "gitlab.com"
        project_id = 12345  # Single project
      }
    }
    repo "team-a-group" {
      gitlab {
        server = "internal.com"
        group_id = 67890  # Entire group
      }
    }
    repo "payment" {
      gitlab {
        server = "gitlab.com"
        project_id = 99999  # Another single project
      }
    }
  }
}
```

**Note**: Each repository in an EggsBucket can specify either `project_id` OR `group_id`, giving you maximum flexibility to mix individual projects and entire groups.

---

## Summary

| Feature | Single Egg | Group-Level Egg | EggsBucket |
|---------|-----------|-----------------|------------|
| **Repositories** | 1 | All in group | Selected list |
| **Discovery** | N/A | Automatic | Manual |
| **Supports project_id** | Yes | No | Yes |
| **Supports group_id** | Yes | Yes | Yes |
| **Mix projects & groups** | N/A | N/A | Yes |
| **Maintenance** | Low | Zero | Medium |
| **Flexibility** | N/A | Low | High |
| **Cross-Instance** | N/A | No | Yes |
| **Use Case** | Single repo | Entire team | Curated selection |
