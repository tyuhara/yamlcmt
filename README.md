# yamlcmt

Compare YAML files with support for multiple documents, designed for Kubernetes manifests and similar use cases.

## Features

- **Identifier-based comparison**: Uses document identifiers (e.g., `metadata.name`) to match documents
- **Multi-document support**: Handles YAML files with multiple documents separated by `---`
- **Clear output**: Color-coded diff showing added, deleted, and modified documents
- **Flexible**: Customizable identifier path for different YAML structures

## Installation

### From source

```bash
# Clone or download the project
cd yamlcmt

# Install dependencies and build
make dev-deps
make build

# Or install directly to $GOPATH/bin
make install
```

### Binary

```bash
# Build for your platform
go build -o yamlcmt ./cmd/yamlcmt
```

## Usage

### Basic comparison

```bash
yamlcmt file1.yaml file2.yaml
```

### With custom identifier

```bash
yamlcmt --key="spec.name" file1.yaml file2.yaml
```

### Summary only

```bash
yamlcmt -c file1.yaml file2.yaml
```

### Verbose output (show full document content)

```bash
yamlcmt -v file1.yaml file2.yaml
```

### Git integration (compare with branch)

```bash
# Auto-detect all changed YAML files compared to main branch
yamlcmt --git-compare=main

# Compare specific file with main branch
yamlcmt --git-compare=main config.yaml

# Compare with different branch
yamlcmt --git-compare=develop

# Combine with GitHub integration
yamlcmt --git-compare=main --github-label --github-repo="owner/repo" --github-pr=123
```

Output format in verbose mode:
```
Summary
0 added, 2 deleted, 0 modified
- apiVersion: rbac.authorization.k8s.io/v1
  kind: RoleBinding
  metadata:
    name: deleted-resource
    namespace: example
  ...
```

Each line of deleted documents is prefixed with `- ` (in red).
Each line of added documents is prefixed with `+ ` (in green).

### Get help

```bash
yamlcmt --help
```

### GitHub Label Integration

Automatically add labels to GitHub PRs based on diff results:

```bash
yamlcmt file1.yaml file2.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123

# Uses GITHUB_TOKEN environment variable
# Or specify token explicitly:
yamlcmt file1.yaml file2.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123 \
  --github-token="ghp_xxx"

# Custom labels
yamlcmt file1.yaml file2.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123 \
  --changes-label="has-changes" \
  --no-changes-label="no-changes"
```

When changes are detected: adds `config-sync/changes` label  
When no changes: adds `config-sync/no-changes` label

**Note**: Requires `gh` CLI to be installed and authenticated.

### Config File-based Integration (tfcmt-style)

For more advanced use cases, use a config file (like tfcmt):

```bash
# Create yamlcmt.yaml config file
cat > yamlcmt.yaml << 'EOF'
repo_owner: ${OWNER}
repo_name: ${REPO_NAME}
yamlcmt:
  compare:
    template: |
      ## YAML Diff Result
      {{if .Link}}[CI link]({{.Link}}){{end}}
      
      ### Summary
      {{.Summary}}
      
      {{if .HasChanges}}
      {{if gt .Deleted 0}}
      ⚠️ **WARNING**: This PR deletes {{.Deleted}} resource(s)
      {{end}}
      
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
EOF

# Use config file
yamlcmt -v file1.yaml file2.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --link="https://console.cloud.google.com/..." \
  --var message="Generated from ${REPO_NAME}" \
  --var resources_type="Service" \
  --var env="production"
```

**Benefits:**
- Template-based comments with custom formatting
- **Cumulative labels** - multiple labels can be added based on change types
- Reusable configuration across CI pipelines
- Support for custom variables in templates

**Label Behavior:**
Labels are cumulative. For example, if a PR has 1 addition, 1 deletion, and 1 modification:
- ✅ `config-sync/add` will be added
- ✅ `config-sync/destroy` will be added  
- ✅ `config-sync/changes` will be added

This makes it easy to see at a glance what types of changes are in the PR.

See `yamlcmt.yaml.example` and `yamlcmt-microservices.yaml.example` for complete examples.

## Project Structure

```
yamlcmt/
├── cmd/
│   └── yamlcmt/
│       └── main.go           # CLI entry point
├── internal/
│   ├── diff/
│   │   └── diff.go          # Diff calculation engine
│   └── parser/
│       └── parser.go        # YAML parser
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Git Integration

### Shell function

Add to your `.bashrc` or `.zshrc`:

```bash
# Compare YAML file with main branch
yamlcmt-git() {
    local file="$1"
    if [ -z "$file" ]; then
        echo "Usage: yamlcmt-git <yaml-file>"
        return 1
    fi
    
    local tmp_file=$(mktemp)
    git show main:"$file" > "$tmp_file" 2>/dev/null
    
    if [ $? -eq 0 ]; then
        yamlcmt "$tmp_file" "$file"
        local exit_code=$?
        rm "$tmp_file"
        return $exit_code
    else
        echo "Error: Could not get file from main branch"
        rm "$tmp_file"
        return 1
    fi
}

# Compare all changed YAML files
yamlcmt-git-all() {
    for file in $(git diff --name-only main | grep -E '\.(yaml|yml)$'); do
        echo "=== $file ==="
        yamlcmt-git "$file"
        echo ""
    done
}
```

Usage:

```bash
# Single file
yamlcmt-git config.yaml

# All changed YAML files
yamlcmt-git-all
```

## Output Example

### Default output (non-verbose)

```
+ Added: new-service-binding
- Deleted: old-service-binding
~ Modified: existing-service-binding
  ~ .subjects[0].name: old-user@example.com → new-user@example.com

Summary:
  Added: 1
  Deleted: 1
  Modified: 1
```

### Verbose output (`-v`)

```
Summary
0 added, 2 deleted, 0 modified
- apiVersion: rbac.authorization.k8s.io/v1
  kind: RoleBinding
  metadata:
    annotations:
      configmanagement.gke.io/cluster-selector: development
    name: deleted-resource-1
    namespace: example
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: edit
  subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: user@example.com
- apiVersion: rbac.authorization.k8s.io/v1
  kind: RoleBinding
  metadata:
    name: deleted-resource-2
  ...
```

## Why yamlcmt?

Standard `yamlcmt` tools compare documents by position, which causes problems with multi-document YAML files:
- When a document is removed, all subsequent documents appear as "modified"
- Hard to track which specific resource was added/deleted

This tool:
- Matches documents by identifier (e.g., `metadata.name`)
- Shows exactly what was added, deleted, or modified
- Perfect for Kubernetes manifests and similar structured YAML

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Run vet
make vet

# Demo with test files
make demo
```

## Project Files

- `README.md` - This file
- `INSTALL.md` - Installation guide
- `CONFIG_GUIDE.md` - Configuration file guide (tfcmt-style)
- `GITHUB_LABEL.md` - GitHub label integration guide
- `STRUCTURE.md` - Architecture documentation
- `yamlcmt.yaml.example` - Basic configuration example

## License

MIT

