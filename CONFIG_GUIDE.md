## Configuration File Guide

yamlcmt supports configuration files in YAML format, similar to [tfcmt](https://github.com/suzuki-shunsuke/tfcmt).

## Basic Structure

```yaml
repo_owner: <GitHub organization or user>
repo_name: <Repository name>

yamlcmt:
  compare:
    template: |
      <Go template for PR comment>
      # Available variables:
      # .Summary - "Plan: X to add, Y to delete, Z to modify"
      # .Details - Full diff output (with -v flag)
      # .HasChanges - boolean
      # .Added, .Deleted, .Modified - numbers
      # .AddedList, .DeletedList, .ModifiedList - []string
      # .Link - CI build link
      # .Vars - map[string]interface{} of custom variables
    when_has_additions:
      label: "<label when additions exist>"
    when_has_deletions:
      label: "<label when deletions exist>"
    when_has_modifications:
      label: "<label when modifications exist>"
    when_no_changes:
      label: "<label when no changes>"
    disable_comment: false
    disable_label: false
```

## Label Selection Logic

Labels are **cumulative** - multiple labels can be added to a single PR based on what types of changes exist:

1. **No changes** (Added = 0, Deleted = 0, Modified = 0): Only `when_no_changes` label
2. **Has additions** (Added > 0): `when_has_additions` label is added
3. **Has deletions** (Deleted > 0): `when_has_deletions` label is added
4. **Has modifications** (Modified > 0): `when_has_modifications` label is added

**Example**: If a PR has 1 addition, 1 deletion, and 1 modification, **all three labels** will be added:
- `config-sync/add`
- `config-sync/destroy`
- `config-sync/changes`

This allows you to immediately see what types of changes are in a PR at a glance.

## Template Variable Details

### Summary Format

The `.Summary` variable uses the format: `Plan: X to add, Y to delete, Z to modify`

Example:
```
Plan: 1 to add, 2 to delete, 0 to modify
```

This format is consistent with tfcmt-style configuration and makes it clear what changes will occur.

**Note**: This is different from the command-line verbose output (`-v` flag), which uses the format `X added, Y deleted, Z modified` for brevity in terminal display.

### Available Template Variables

| Variable | Type | Description | Example |
|----------|------|-------------|---------|
| `.Summary` | string | Change summary in "Plan:" format | `"Plan: 1 to add, 2 to delete, 0 to modify"` |
| `.Details` | string | Full diff output (requires `-v` flag) | Full YAML diff with +/- prefixes |
| `.HasChanges` | bool | True if any changes exist | `true` or `false` |
| `.Added` | int | Number of added documents | `1` |
| `.Deleted` | int | Number of deleted documents | `2` |
| `.Modified` | int | Number of modified documents | `0` |
| `.AddedList` | []string | List of added document names | `["service-a", "service-b"]` |
| `.DeletedList` | []string | List of deleted document names | `["old-service"]` |
| `.ModifiedList` | []string | List of modified document names | `["config-map"]` |
| `.Link` | string | CI build link (from `--link` flag) | `"https://ci.example.com/build/123"` |
| `.Vars` | map[string]interface{} | Custom variables (from `--var` flags) | Access as `.Vars.environment`, `.Vars.service`, etc. |

### Using Template Variables

```yaml
template: |
  ## YAML Configuration Changes

  {{if .Link}}**[View Build]({{.Link}})**{{end}}

  ### Summary
  {{.Summary}}

  {{if gt .Added 0}}
  #### ✨ Added ({{.Added}})
  {{range .AddedList}}
  - {{.}}
  {{end}}
  {{end}}

  {{if gt .Deleted 0}}
  #### 🗑️ Deleted ({{.Deleted}})
  {{range .DeletedList}}
  - {{.}}
  {{end}}
  {{end}}

  {{if gt .Modified 0}}
  #### ✏️ Modified ({{.Modified}})
  {{range .ModifiedList}}
  - {{.}}
  {{end}}
  {{end}}

  {{if .HasChanges}}
  <details><summary>Full Diff</summary>

  ```
  {{.Details}}
  ```
  </details>
  {{else}}
  ✅ No changes detected
  {{end}}
  
  ---
  Environment: **{{.Vars.environment}}**
```

### Custom Variables

Pass custom variables using `--var key=value`:

```bash
yamlcmt -v old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --var environment="production" \
  --var service="api-server" \
  --var deployer="${USER}"
```

Access in template:
```yaml
template: |
  Deployment to {{.Vars.environment}}
  Service: {{.Vars.service}}
  Deployer: {{.Vars.deployer}}
```

### Template Conditions

Use Go template conditions for dynamic content:

```yaml
template: |
  {{if .HasChanges}}
    {{if gt .Deleted 0}}
    ⚠️ **WARNING**: This PR deletes {{.Deleted}} resource(s)
    {{end}}

    {{if and (eq .Added 0) (gt .Modified 0)}}
    ℹ️ This PR only modifies existing resources
    {{end}}
  {{else}}
    ✅ No changes to YAML files
  {{end}}
```

Common conditions:
- `{{if .HasChanges}}` - If any changes exist
- `{{if eq .Added 0}}` - If no additions
- `{{if gt .Deleted 0}}` - If deletions exist
- `{{if and (condition1) (condition2)}}` - Multiple conditions
- `{{if or (condition1) (condition2)}}` - Any condition true

## Usage Examples

### Basic Usage

```bash
yamlcmt -v old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123
```

### With CI Link

```bash
yamlcmt -v old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --link="https://console.cloud.google.com/cloud-build/builds/..."
```

### With Custom Variables

```bash
yamlcmt -v old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --var message="Deployment to production" \
  --var environment="prod" \
  --var service="api-server"
```

Then in your template:
```
{{.Vars.message}}
Environment: {{.Vars.environment}}
Service: {{.Vars.service}}
```

## Example Configurations

### Minimal Configuration

```yaml
repo_owner: myorg
repo_name: myrepo

yamlcmt:
  compare:
    template: |
      ## YAML Changes
      {{.Summary}}
    when_has_additions:
      label: "has-additions"
    when_has_deletions:
      label: "has-deletions"
    when_has_modifications:
      label: "has-changes"
    when_no_changes:
      label: "no-changes"
```

### Full-featured Configuration

```yaml
repo_owner: ${OWNER}
repo_name: ${REPO_NAME}

yamlcmt:
  compare:
    template: |
      ## [{{.Vars.service}}] Configuration Changes
      
      {{if .Link}}**[View CI Build]({{.Link}})**{{end}}
      
      {{.Vars.message}}
      
      ### Change Summary
      {{.Summary}}

      Details:
      - Added: {{.Added}} resources
      - Deleted: {{.Deleted}} resources
      - Modified: {{.Modified}} resources
      
      {{if .HasChanges}}
      {{if gt .Deleted 0}}
      ⚠️ **WARNING**: This PR contains deletions. Please review carefully.
      {{end}}
      
      <details>
      <summary>📋 Full Diff</summary>
      
      ```yaml
      {{.Details}}
      ```
      
      </details>
      
      ### ⚠️ Review Guidelines
      - Verify all resource names are correct
      - Check for unintended deletions
      - Confirm modifications are expected
      {{else}}
      ✅ No changes detected in YAML files.
      {{end}}
      
      ---
      
      **Environment**: {{.Vars.environment}}  
      **Target Branch**: {{.Vars.target_branch}}
      
      To retry this job, comment `/retry` on this PR.
      
      [Documentation](https://example.com/docs)
      
    when_has_additions:
      label: "yaml/added"
    when_has_deletions:
      label: "yaml/deleted"
    when_has_modifications:
      label: "yaml/modified"
    when_no_changes:
      label: "yaml/no-changes"
```

### Kubernetes-specific Configuration

```yaml
repo_owner: myorg
repo_name: k8s-manifests

yamlcmt:
  compare:
    template: |
      ## Kubernetes Manifest Changes
      
      {{if .Link}}[CI Pipeline]({{.Link}}){{end}}
      
      Cluster: **{{.Vars.cluster}}**  
      Namespace: **{{.Vars.namespace}}**
      
      {{if .HasChanges}}
      ### Changes
      {{if gt .Added 0}}✨ **{{.Added}}** new resource(s){{end}}
      {{if gt .Deleted 0}}🗑️  **{{.Deleted}}** deleted resource(s){{end}}
      {{if gt .Modified 0}}✏️  **{{.Modified}}** modified resource(s){{end}}
      
      <details>
      <summary>View Diff</summary>
      
      ```diff
      {{.Details}}
      ```
      
      </details>
      {{else}}
      ✅ No changes to Kubernetes manifests.
      {{end}}
      
    when_has_deletions:
      label: "k8s/deletion-warning"
    when_has_additions:
      label: "k8s/additions"
    when_has_modifications:
      label: "k8s/modifications"
    when_no_changes:
      label: "k8s/no-changes"
```

## Configuration File Location

By convention, place your config file in one of these locations:
- `.yamlcmt/yamlcmt.yaml` (recommended for project-wide config)
- `.github/yamlcmt.yaml` (for GitHub Actions)
- `yamlcmt.yaml` (root of repository)

## Disabling Features

### Disable Comments Only

```yaml
yamlcmt:
  compare:
    disable_comment: true
    # Labels will still be added
```

### Disable Labels Only

```yaml
yamlcmt:
  compare:
    disable_label: true
    # Comments will still be posted
```

### Disable Both

Use the legacy `--github-label` flag instead, or don't use any GitHub integration flags.

## CI Integration

### Cloud Build

```yaml
steps:
  - name: 'gcr.io/cloud-builders/git'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        yamlcmt -v old.yaml new.yaml \
          --config=.yamlcmt/yamlcmt.yaml \
          --post-comment \
          --github-pr=${_PR_NUMBER} \
          --link=${BUILD_URL} \
          --var environment=${_ENVIRONMENT}
```

### GitHub Actions

```yaml
- name: Check YAML changes
  run: |
    yamlcmt -v old.yaml new.yaml \
      --config=.github/yamlcmt.yaml \
      --post-comment \
      --github-pr=${{ github.event.pull_request.number }} \
      --link=${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }} \
      --var environment=${{ inputs.environment }}
```

## Comparison with tfcmt

| Feature | tfcmt | yamlcmt |
|---------|-------|----------|
| Config file format | YAML | YAML |
| Template engine | Go templates | Go templates |
| Label selection | Plan/Apply states | Diff types (add/delete/modify) |
| Custom variables | ✅ | ✅ |
| GitHub API | Direct | via `gh` CLI |
| Primary use case | Terraform | YAML files |

## Troubleshooting

### Template Rendering Errors

If you see template errors:
1. Check syntax with Go template validator
2. Ensure all variables used in template are passed via `--var`
3. Use `{{if}}` checks before accessing optional variables

### Labels Not Applied

1. Verify `disable_label: false` in config
2. Check that all label conditions have a `label:` value
3. Ensure `gh` CLI is authenticated

### Comments Not Posted

1. Verify `disable_comment: false` in config
2. Check that `template:` is not empty
3. Use `-v` flag to populate `.Details` variable
4. Ensure `--post-comment` flag is present

## Best Practices

1. **Version control your config**: Commit `yamlcmt.yaml` to your repository
2. **Use meaningful labels**: Choose labels that integrate with your workflow
3. **Keep templates concise**: Long comments can be hard to read
4. **Use collapsible sections**: Wrap detailed output in `<details>` tags
5. **Test templates locally**: Use `--config` without `--post-comment` first
6. **Document variables**: Add comments in config explaining custom variables
