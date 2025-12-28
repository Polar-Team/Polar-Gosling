# Specification Changelog

## 2024-12-28: GitLab Multi-Instance and Group-Level Runner Support

### Summary
Added support for multiple GitLab instances (GitLab.com, self-hosted, GitLab Dedicated) and group-level runner registration. Simplified secret management by making it automatic based on egg name and GitLab server.

**Important**: Group-level Eggs and EggsBuckets are different concepts:
- **Group-Level Egg**: Automatically manages all repositories in a GitLab group
- **EggsBucket**: Manually groups specific repositories with shared configuration (can be from different groups/instances)

### Requirements Changes

**Updated Requirement 11: GitLab Integration**
- Added support for multiple GitLab instances via server FQDN
- Added support for both project-level and group-level runner registration
- Added automatic project discovery for group-level runners
- Added automatic webhook configuration for all projects in a group
- Added API token permission requirements for group-level operations

**Updated Glossary**
- Added **GitLab_Server**: The FQDN of a GitLab instance
- Added **Project_Level_Runner**: A runner registered to a specific GitLab project
- Added **Group_Level_Runner**: A runner registered to a GitLab group
- Updated **Egg** definition to include GitLab groups

### Design Changes

**Simplified .fly Configuration**
- Removed explicit secret references (`token_secret`, `api_token_secret`, `webhook_secret`)
- Added required `server` field (GitLab server FQDN)
- Added optional `group_id` field (mutually exclusive with `project_id`)
- System now automatically manages secrets based on egg name and server

**Automatic Secret Management**
- Secrets follow convention: `yc-lockbox://gitlab/{server}/{egg-name}/{secret-type}`
- Example: `yc-lockbox://gitlab/gitlab.com/my-app/api-token`
- Example: `yc-lockbox://gitlab/gitlab.com/my-app/runner-token`
- Example: `yc-lockbox://gitlab/gitlab.com/my-app/webhook-secret`

**New Configuration Examples**

Project-Level:
```hcl
egg "my-app" {
  gitlab {
    server = "gitlab.com"
    project_id = 12345
  }
}
```

Group-Level:
```hcl
egg "platform-team" {
  gitlab {
    server = "gitlab.company.com"
    group_id = 789
  }
}
```

**Updated Webhook Validation**
- Webhook handler now supports both `project_id` and `group_id`
- Secret URIs are built automatically from egg name and server
- Format: `f"yc-lockbox://gitlab/{server}/{egg_name}/webhook-secret"`

**Updated IAM Permissions**
- Simplified MotherGoose permissions to `lockbox.payloadViewer` for `gitlab/*`
- Removed separate permissions for `webhooks/*` and `gitlab-tokens/*`

**Group-Level Runner Workflow**
1. Gosling CLI discovers all projects in the group via GitLab API
2. Registers a single group-level runner with the GitLab group
3. Automatically configures webhooks for all projects in the group
4. All projects share the same runner pool and configuration

### Benefits

1. **Cleaner Configuration**: No need to specify secret URIs in .fly files
2. **Convention Over Configuration**: Secrets follow predictable naming pattern
3. **Multi-Instance Support**: Can manage runners across different GitLab instances
4. **Group-Level Efficiency**: Single runner configuration for entire teams
5. **Automatic Discovery**: System discovers and configures all group projects
6. **Better Security**: Secrets are never exposed in configuration files

### EggsBucket vs Group-Level Egg

Both concepts allow managing multiple repositories, but serve different purposes:

| Aspect | EggsBucket | Group-Level Egg |
|--------|-----------|-----------------|
| **Purpose** | Manually group specific repositories/groups | Automatically manage entire GitLab group |
| **Repository Selection** | Explicit list in config | Automatic discovery via GitLab API |
| **Supports project_id** | Yes (per repo) | No |
| **Supports group_id** | Yes (per repo) | Yes |
| **Mix projects & groups** | Yes | No |
| **GitLab Scope** | Can mix repos from different groups/instances | All repos in one GitLab group |
| **Configuration** | `eggsbucket` block with `repositories` | `egg` block with `group_id` |
| **Maintenance** | Manual updates when adding/removing repos | Automatic when repos added to group |
| **Use Case** | Curated list of specific projects/groups | Entire team/organization group |

**Example EggsBucket** (Manual grouping with mixed project_id and group_id):
```hcl
eggsbucket "selected-services" {
  repositories {
    repo "api-service" {
      gitlab {
        server = "gitlab.com"
        project_id = 12345  # Single project
      }
    }
    repo "frontend-team" {
      gitlab {
        server = "gitlab.company.com"  # Different instance!
        group_id = 67890                # Entire group (auto-discovered)
      }
    }
    repo "payment-service" {
      gitlab {
        server = "gitlab.com"
        project_id = 99999  # Another single project
      }
    }
  }
}
```

**Example Group-Level Egg** (Automatic discovery):
```hcl
egg "platform-team" {
  gitlab {
    server = "gitlab.company.com"
    group_id = 789  # All projects in this group automatically included
  }
}
```

**When to use EggsBucket**:
- You want to group specific repositories OR entire groups from different GitLab groups
- You want to mix repositories from different GitLab instances
- You want to combine individual projects with entire groups
- You need fine-grained control over which repositories/groups share runners
- You want to cherry-pick which projects and groups share runners

**When to use Group-Level Egg**:
- All your repositories are in a single GitLab group
- You want automatic discovery of new repositories
- You want to manage an entire team's infrastructure as one unit
- You want minimal configuration maintenance
- You don't need to mix with other projects/groups

### Migration Notes

For existing configurations:
1. Add `server` field to all `gitlab` blocks
2. Remove `token_secret`, `api_token_secret`, and `webhook_secret` fields
3. Ensure secrets are stored in new convention: `gitlab/{server}/{egg-name}/{type}`
4. For group-level runners, replace `project_id` with `group_id`

### API Token Requirements

**Project-Level**: Standard project access token
**Group-Level**: Token with `api` scope OR combination of:
- `read_api` - to list group projects
- `write_repository` - to configure webhooks
- `manage_runners` - to register group runners
