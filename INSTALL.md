# Installation Guide

## Prerequisites

- Go 1.21 or higher
- Git (for `--git-compare` functionality)
- `gh` CLI (for GitHub integration features)

## Installation Methods

### Installing on Mac

```bash
# Extract the downloaded tar.gz
cd ~/Downloads
tar xzf yamlcmt.tar.gz

# Navigate to directory
cd yamlcmt

# Install dependencies and build
make dev-deps
make build

# Verify installation with test files
./yamlcmt testdata/old.yaml testdata/new.yaml

# Install to system (optional)
make install
# or
sudo mv yamlcmt /usr/local/bin/
```

## Project Structure

```
yamlcmt/
├── cmd/
│   └── yamlcmt/
│       └── main.go              # CLI entry point (using kong)
├── internal/
│   ├── config/
│   │   └── config.go           # Configuration loader
│   ├── diff/
│   │   └── diff.go             # Diff calculation engine
│   ├── git/
│   │   └── git.go              # Git integration
│   ├── github/
│   │   └── github.go           # GitHub integration
│   └── parser/
│       └── parser.go           # YAML parser
├── scripts/
│   └── ci-integration-example.sh  # CI integration examples
├── testdata/
│   ├── old.yaml                # Test file (baseline)
│   └── new.yaml                # Test file (modified)
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── yamlcmt.yaml.example       # Configuration example
```

## Usage

### Basic comparison

```bash
yamlcmt file1.yaml file2.yaml
```

### Specify custom identifier

```bash
yamlcmt --key="spec.name" file1.yaml file2.yaml
```

### Show summary only

```bash
yamlcmt -c file1.yaml file2.yaml
```

### Verbose output (show full documents)

```bash
yamlcmt -v file1.yaml file2.yaml
```

### Disable color output

```bash
yamlcmt --no-color file1.yaml file2.yaml
```

### Git comparison mode

```bash
# Auto-detect all changed YAML files compared to main branch
yamlcmt --git-compare=main

# Compare specific file with main branch
yamlcmt --git-compare=main config.yaml

# Compare with different branch
yamlcmt --git-compare=develop

# Verbose output with Git comparison
yamlcmt -v --git-compare=main
```

### Display help

```bash
yamlcmt --help
```

## GitHub Integration (Config-based)

You can automatically post comments to GitHub PRs and add labels based on diff results using a configuration file.

### Prerequisites

- `gh` CLI installed and authenticated
- `GITHUB_TOKEN` environment variable set or specified via `--github-token`
- yamlcmt.yaml configuration file

### Basic Usage

```bash
# Post comment and add labels
yamlcmt -v --no-color old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123

# With custom variables
yamlcmt -v --no-color old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --var message="Config changes for development environment"

# With CI build link
yamlcmt -v --no-color old.yaml new.yaml \
  --config=yamlcmt.yaml \
  --post-comment \
  --github-pr=123 \
  --link="https://ci.example.com/build/123"
```

### Configuration File Example

Create `yamlcmt.yaml`:

```yaml
repo_owner: your-org
repo_name: your-repo

yamlcmt:
  compare:
    template: |
      ## YAML Diff Result

      {{if .Link}}[CI link]({{.Link}}){{end}}

      {{if .Vars.message}}{{.Vars.message}}{{end}}

      {{if gt .Deleted 0}}
      ⚠️ **WARNING**: This PR deletes {{.Deleted}} resource(s). Please review carefully.
      {{end}}

      ### Summary
      ```
      {{.Summary}}
      ```

      {{if gt (len .AddedList) 0}}
      * Add
      {{range .AddedList}}  * {{.}}
      {{end}}
      {{end}}

      {{if gt (len .DeletedList) 0}}
      * Delete
      {{range .DeletedList}}  * {{.}}
      {{end}}
      {{end}}

      {{if gt (len .ModifiedList) 0}}
      * Modify
      {{range .ModifiedList}}  * {{.}}
      {{end}}
      {{end}}

      {{if .HasChanges}}
      <details><summary>Details (Click me)</summary>

      ```
      {{.Details}}
      ```

      </details>
      {{else}}
      ✅ No changes detected in YAML files.
      {{end}}

    when_has_additions:
      label: "config-sync/add"

    when_has_deletions:
      label: "config-sync/destroy"

    when_has_modifications:
      label: "config-sync/changes"

    when_no_changes:
      label: "config-sync/no-changes"

    disable_comment: false
    disable_label: false
```

### How It Works

- **Template Variables**: `.Summary` (format: "Plan: X to add, Y to delete, Z to modify"), `.Details`, `.HasChanges`, `.Added`, `.Deleted`, `.Modified`, `.AddedList`, `.DeletedList`, `.ModifiedList`, `.Link`, `.Vars`
- **Labels**: Applied based on diff results (cumulative for changes, exclusive for no-changes)
- **Custom Variables**: Pass via `--var key=value` and use as `{{.Vars.key}}`

### CI Usage Example

```bash
#!/bin/bash
PR_NUMBER="$(fetch_pr_number)"

for file in $(git diff --name-only main | grep -E '\.(yaml|yml)$'); do
  tmp_file=$(mktemp)
  git show main:"${file}" > "${tmp_file}" 2>/dev/null
  
  yamlcmt -v --no-color "${tmp_file}" "${file}" \
    --config=yamlcmt.yaml \
    --post-comment \
    --github-pr="${PR_NUMBER}" \
    --var message="Config changes detected" \
    || true
    
  rm -f "${tmp_file}"
done
```

Benefits of this approach:
- Custom comment templates
- Flexible label configuration
- Rich template variables
- Integration with CI systems

## GitHub Label Integration (Legacy)

Automatically apply labels to GitHub PRs based on diff results.

### Prerequisites

- `gh` CLI installed and authenticated
- `GITHUB_TOKEN` environment variable set or specified via `--github-token`

### Basic Usage

```bash
# Using environment variable
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="${OWNER}/${REPO_NAME}" \
  --github-pr=456

# Specify token via command line
yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="${OWNER}/${REPO_NAME}" \
  --github-pr=456 \
  --github-token="ghp_xxxxxxxxxxxx"

# Use custom label names
yamlcmt old.yaml new.yaml \
  --github-label \
  --github-repo="owner/repo" \
  --github-pr=123 \
  --changes-label="yaml/has-changes" \
  --no-changes-label="yaml/no-changes"
```

### How It Works

- **When changes exist**: Add `config-sync/changes` label
- **When no changes**: Add `config-sync/no-changes` label
- Default label names can be changed with `--changes-label` and `--no-changes-label`

### CI Usage Example

```bash
#!/bin/bash
PR_NUMBER="$(fetch_pr_number)"

for file in $(git diff --name-only main | grep -E '\.(yaml|yml)$'); do
  tmp_file=$(mktemp)
  git show main:"${file}" > "${tmp_file}" 2>/dev/null
  
  yamlcmt -v "${tmp_file}" "${file}" \
    --github-label \
    --github-repo="${GITHUB_REPOSITORY}" \
    --github-pr="${PR_NUMBER}" \
    || true
    
  rm -f "${tmp_file}"
done
```

## Git Integration

### Set up as Shell Function

Add to `.bashrc` or `.zshrc`:

```bash
# Check YAML diff against main branch using yamlcmt
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

# Check all changed YAML files
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

## CI/CD Integration

Refer to `scripts/ci-integration-example.sh`.

By replacing the git diff part of existing CI scripts with yamlcmt,
you can achieve more readable diff display.

### Usage in Cloud Build or GitHub Actions

```yaml
# GitHub Actions example
- name: Install yamlcmt
  run: |
    cd /path/to/yamlcmt
    make install

- name: Generate config diff
  run: |
    # Use yamlcmt in CI script
    ./path/to/generate-config
```

## Troubleshooting

### "go: not found" error

Go is not installed. Install from the [official site](https://go.dev/doc/install).

### "command not found: yamlcmt"

Binary is not in PATH:

```bash
# Check installation location
which yamlcmt

# Check PATH
echo $PATH

# Check GOPATH
echo $GOPATH
```

Add `$GOPATH/bin` or `/usr/local/bin` to PATH:

```bash
export PATH=$PATH:$GOPATH/bin
```

### "cannot find package" error

Dependencies are missing:

```bash
cd /Users/yuhara/tyuhara/yamlcmt
make dev-deps
```

## Difference from expiry-monitor

This project adopts the same structure as `expiry-monitor`:

- CLI entry point in `cmd/` directory
- Internal packages in `internal/` directory
- CLI framework using `github.com/alecthomas/kong`
- Subcommand format (currently only compare)

The extensible design allows adding other subcommands (e.g., validate, format) in the future.

## Next Steps

- Check [README.md](README.md) for detailed usage examples
- Check [scripts/ci-integration-example.sh](scripts/ci-integration-example.sh) for CI integration examples
- Verify operation with test files (`testdata/old.yaml`, `testdata/new.yaml`)
- See [CONFIG_GUIDE.md](CONFIG_GUIDE.md) for configuration file details
- Review [STRUCTURE.md](STRUCTURE.md) for architectural overview
