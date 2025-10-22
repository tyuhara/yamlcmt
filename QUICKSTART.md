# Quick Start Guide

This guide helps you get started with yamlcmt's config file-based GitHub integration.

## Prerequisites

- Go 1.21+
- `gh` CLI installed and authenticated
- `GITHUB_TOKEN` environment variable set

## Installation

```bash
cd yamlcmt
make dev-deps
make build
```

## Quick Start: Config File Method

### Step 1: Create Config File

Copy the example and customize:

```bash
cp yamlcmt.yaml.example .yamlcmt/yamlcmt.yaml

# Edit to match your repository
vi .yamlcmt/yamlcmt.yaml
```

Example config:
```yaml
repo_owner: ${OWNER}
repo_name: ${REPO}

yamlcmt:
  compare:
    template: |
      ## YAML Changes
      {{if .Link}}[CI Build]({{.Link}}){{end}}
      
      {{.Summary}}
      
      {{if .HasChanges}}
      <details><summary>Details</summary>
      
      ```
      {{.Details}}
      ```
      </details>
      {{end}}
    when_has_additions:
      label: "config-sync/add"
    when_has_deletions:
      label: "config-sync/destroy"
    when_has_modifications:
      label: "config-sync/changes"
    when_no_changes:
      label: "config-sync/no-changes"
```

### Step 2: Test Locally

```bash
./yamlcmt -v test-old.yaml test-new.yaml \
  --config=.yamlcmt/yamlcmt.yaml \
  --post-comment \
  --github-pr=YOUR_PR_NUMBER \
  --link="https://example.com/build/123"
```

### Step 3: Integrate into CI

Add to your CI script:

```bash
yamlcmt -v old.yaml new.yaml \
  --config=.yamlcmt/yamlcmt.yaml \
  --post-comment \
  --github-pr="${PR_NUMBER}" \
  --link="${BUILD_URL}" \
  --var message="Deployment to ${ENV}" \
  --var env="${ENV}"
```

## Quick Start: Legacy Method

For simpler use cases without templates:

```bash
./yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123
```

## Common Use Cases

### Generate Config Script (${REPO_NAME})

Replace the `github_label` and `github_comment` calls:

**Before:**
```bash
detailed_diff=$(git diff --staged)
if [[ ${detailed_diff} != "" ]]; then
  github_comment "${target_branch}" "${detailed_diff}" "${resources_type}" "${env}"
  github_label "config-sync/changes"
fi
```

**After:**
```bash
for file in $(git diff --name-only --staged | grep -E '\.(yaml|yml)$'); do
  tmp_file=$(mktemp)
  git show main:"${file}" > "${tmp_file}" 2>/dev/null
  
  yamlcmt -v "${tmp_file}" "${file}" \
    --config=.yamlcmt/yamlcmt.yaml \
    --post-comment \
    --github-pr="${PR_NUMBER}" \
    --link="${BUILD_URL}" \
    --var message="Generated from ${REPO_NAME}" \
    --var resources_type="${resources_type}" \
    --var env="${env}" \
    --var target_branch="${target_branch}" \
    || true
    
  rm -f "${tmp_file}"
done
```

### Cloud Build

```yaml
steps:
  - name: 'gcr.io/cloud-builders/git'
    args:
      - '-c'
      - |
        yamlcmt -v old.yaml new.yaml \
          --config=.yamlcmt/yamlcmt.yaml \
          --post-comment \
          --github-pr=${_PR_NUMBER} \
          --link=${BUILD_URL}
```

### GitHub Actions

```yaml
- name: YAML Diff
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    yamlcmt -v old.yaml new.yaml \
      --config=.github/yamlcmt.yaml \
      --post-comment \
      --github-pr=${{ github.event.pull_request.number }} \
      --link=${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
```

## Configuration Examples

### Minimal

```yaml
repo_owner: myorg
repo_name: myrepo
yamlcmt:
  compare:
    template: "Changes: {{.Summary}}"
    when_has_additions:
      label: "additions"
    when_has_deletions:
      label: "deletions"
    when_has_modifications:
      label: "changes"
    when_no_changes:
      label: "no-changes"
```

### With Deletions Warning

```yaml
yamlcmt:
  compare:
    template: |
      {{if gt .Deleted 0}}
      ⚠️ **WARNING**: This PR deletes {{.Deleted}} resource(s)
      {{end}}
      {{.Summary}}
    when_has_deletions:
      label: "deletion-warning"
    when_has_modifications:
      label: "has-changes"
```

### ${REPO_NAME}

Use `yamlcmt-microservices.yaml.example` as a starting point:

```bash
cp yamlcmt-microservices.yaml.example .yamlcmt/yamlcmt.yaml
```

## Troubleshooting

### "GitHub token not provided"
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
```

### "failed to post comment"
- Check `gh auth status`
- Verify repository and PR number
- Ensure token has correct permissions

### Template rendering errors
- Validate YAML syntax
- Check that all `{{.Vars.xxx}}` are passed via `--var`
- Test template without `--post-comment` first

## Next Steps

- Read [CONFIG_GUIDE.md](CONFIG_GUIDE.md) for detailed configuration options
- See [GITHUB_LABEL.md](GITHUB_LABEL.md) for legacy label integration
- Check [scripts/ci-integration-with-config.sh](scripts/ci-integration-with-config.sh) for complete example

## Comparison: Legacy vs Config File

| Feature | Legacy (`--github-label`) | Config File (`--config`) |
|---------|--------------------------|--------------------------|
| Labels | Simple (changes/no-changes) | **Cumulative** (add/destroy/changes/no-changes) |
| Comments | Not supported | Fully customizable templates |
| Variables | Not supported | Full template variables support |
| Configuration | Command-line flags | YAML file |
| Label Behavior | Single label only | Multiple labels per PR |
| Best for | Simple CI scripts | Complex workflows |

**Label Behavior Example:**

If a PR has 1 addition, 1 deletion, and 1 modification:

**Legacy mode:**
- Only `config-sync/changes` is added

**Config file mode (cumulative):**
- ✅ `config-sync/add` is added
- ✅ `config-sync/destroy` is added
- ✅ `config-sync/changes` is added

This makes it immediately clear what types of changes exist in the PR.

## Migration Path

1. Start with legacy `--github-label` for basic needs
2. Create config file when you need custom comments
3. Gradually move all logic to config file
4. Both methods can coexist during migration
