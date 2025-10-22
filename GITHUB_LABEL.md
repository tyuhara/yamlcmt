# GitHub Label Integration Guide

The `--github-label` feature of yamlcmt allows you to automatically apply labels to GitHub PRs based on diff results.

## Overview

This feature eliminates the need to call the `github_label` function within CI scripts, integrating the labeling logic into yamlcmt.

### Traditional Approach

```bash
# Get diff
detailed_diff=$(git diff --staged)

# Determine changes and apply label
if [[ ${detailed_diff} != "" ]]; then
  github_label "config-sync/changes"
else
  github_label "config-sync/no-changes"
fi
```

### New Approach (yamlcmt Integration)

```bash
# yamlcmt automatically executes diff detection and labeling
yamlcmt file1.yaml file2.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123
```

## Prerequisites

1. **gh CLI installed**
   ```bash
   brew install gh
   # or
   apt-get install gh
   ```

2. **GitHub authenticated**
   ```bash
   gh auth login
   # or set environment variable
   export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
   ```

3. **Appropriate permissions**
   - Write access to the repository
   - Permission to add labels to PRs

## Basic Usage

### Specify token via environment variable

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="${OWNER}/${REPO_NAME}" \
  --github-pr=456
```

### Specify token via command line

```bash
yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="${OWNER}/${REPO_NAME}" \
  --github-pr=456 \
  --github-token="ghp_xxxxxxxxxxxx"
```

### Use custom label names

```bash
yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123 \
  --changes-label="yaml/has-changes" \
  --no-changes-label="yaml/no-changes"
```

## CI Integration Examples

### Cloud Build

```yaml
steps:
  - name: 'gcr.io/cloud-builders/git'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        # Install yamlcmt
        go install github.com/tyuhara/yamlcmt/cmd/yamlcmt@latest
        
        # Get PR number
        PR_NUMBER=$(fetch_pr_number)
        
        # Compare each YAML file and apply labels
        for file in $(git diff --name-only main | grep -E '\.(yaml|yml)$'); do
          tmp_file=$(mktemp)
          git show main:"${file}" > "${tmp_file}" 2>/dev/null
          
          yamlcmt -v "${tmp_file}" "${file}" \
            --github-label \
            --github-repo="${REPO_OWNER}/${REPO_NAME}" \
            --github-pr="${PR_NUMBER}" \
            || true
            
          rm -f "${tmp_file}"
        done
    env:
      - 'GITHUB_TOKEN=${_GITHUB_TOKEN}'
```

### GitHub Actions

```yaml
name: Check YAML Changes

on:
  pull_request:
    paths:
      - '**.yaml'
      - '**.yml'

jobs:
  check-yaml:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install yamlcmt
        run: go install github.com/tyuhara/yamlcmt/cmd/yamlcmt@latest
      
      - name: Check YAML changes
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          for file in $(git diff --name-only origin/main | grep -E '\.(yaml|yml)$'); do
            tmp_file=$(mktemp)
            git show origin/main:"${file}" > "${tmp_file}" 2>/dev/null
            
            yamlcmt -v "${tmp_file}" "${file}" \
              --github-label \
              --github-repo="${{ github.repository }}" \
              --github-pr="${{ github.event.pull_request.number }}" \
              || true
              
            rm -f "${tmp_file}"
          done
```

## Updating Existing CI Scripts

### Example from generate-config script

**Before:**
```bash
detailed_diff=$(git diff --staged)

if [[ ${detailed_diff} != "" ]]; then
  github_comment "${target_branch}" "${detailed_diff}" "${resources_type}" "${env}"
  github_label "config-sync/changes"
else
  github_label "config-sync/no-changes"
fi
```

**After:**
```bash
# Get PR number
pr_number="$(fetch_pr_number)"

# Execute diff detection and labeling with yamlcmt
detailed_diff=""
for file in $(git diff --name-only --staged | grep -E '\.(yaml|yml)$'); do
  tmp_file=$(mktemp)
  git show main:"${file}" > "${tmp_file}" 2>/dev/null
  
  # Get detailed output for comments with -v option
  yamlcmt_output=$(yamlcmt -v "${tmp_file}" "${file}" \
    --github-label \
    --github-repo="${OWNER}/${REPO_NAME}" \
    --github-pr="${pr_number}" \
    2>&1 || true)
  
  if [[ -n "${detailed_diff}" ]]; then
    detailed_diff+=$'\n\n'"---"$'\n\n'
  fi
  detailed_diff+="${yamlcmt_output}"
  
  rm -f "${tmp_file}"
done

# Comment as before
if [[ ${detailed_diff} != "" ]]; then
  github_comment "${target_branch}" "${detailed_diff}" "${resources_type}" "${env}"
fi
```

## How It Works

### Label Determination Logic

```go
if result.HasDifferences() {
    // Added, Deleted, or Modified exists
    label = c.ChangesLabel  // Default: "config-sync/changes"
} else {
    // No differences
    label = c.NoChangesLabel  // Default: "config-sync/no-changes"
}
```

### Executed gh command

```bash
gh pr edit <PR_NUMBER> \
  --repo <REPOSITORY> \
  --add-label <LABEL_NAME>
```

### Error Handling

- Missing required parameters: Display error message and exit
- GitHub token not found: Display error message
- gh command fails: Display error details and output

## Troubleshooting

### "gh: command not found"

gh CLI is not installed.

```bash
# macOS
brew install gh

# Ubuntu/Debian
sudo apt-get install gh

# Other environments
https://cli.github.com/manual/installation
```

### "GitHub token not provided"

Specify the token via environment variable or option.

```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
```

or

```bash
yamlcmt ... --github-token="ghp_xxxxxxxxxxxx"
```

### "failed to add label: HTTP 404"

- Verify repository name is correct
- Verify PR number is correct
- Verify token has appropriate permissions

### Labels are added multiple times

`gh pr edit --add-label` does not add the same label multiple times.
If the label already exists, it does nothing.

## Best Practices

1. **Use verbose mode (-v) for comments**
   ```bash
   yamlcmt -v file1.yaml file2.yaml --github-label ...
   ```
   → Detailed diff can be pasted into GitHub comments

2. **When processing multiple files**
   ```bash
   # Execute yamlcmt for each file
   # Label is applied on the first file
   ```

3. **Using with existing github_label function**
   ```bash
   # Gradual migration is possible
   # Some use yamlcmt, some use traditional approach
   ```

4. **Idempotency in CI**
   ```bash
   # Don't fail entire CI even if yamlcmt fails
   yamlcmt ... || true
   ```

## Summary

The `--github-label` feature provides:
- ✅ Simplified CI scripts
- ✅ Label determination logic integrated into yamlcmt
- ✅ Diff detection and labeling in one execution
- ✅ Gradual integration with existing CI possible
