# yamlcmt Project Structure

```
yamlcmt/
├── cmd/
│   └── yamlcmt/
│       └── main.go              # CLI entry point (using kong)
│
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loader
│   │                            # - LoadConfig: Load yamlcmt.yaml
│   │                            # - GetLabels: Determine labels based on changes
│   │
│   ├── diff/
│   │   └── diff.go              # Diff calculation engine
│   │                            # - Engine: Core of diff calculation
│   │                            # - Result: Representation of diff results
│   │                            # - Print/PrintSummary: Output functionality
│   │
│   ├── git/
│   │   └── git.go               # Git integration
│   │                            # - GetChangedYAMLFiles: Detect changed files
│   │                            # - CombineFilesForComparison: Combine files (legacy)
│   │                            # - ParseFilesWithSourceTracking: Parse with source tracking
│   │                            # - cleanYAMLContent: Clean YAML content
│   │                            # - IsGitRepository: Check Git repository
│   │                            # - BranchExists: Verify branch existence
│   │
│   ├── github/
│   │   └── github.go            # GitHub integration
│   │                            # - PostComment: Post comment to PR
│   │                            # - AddLabels: Add labels to PR
│   │                            # - RenderTemplate: Render comment template
│   │                            # - PrepareTemplateData: Prepare template data
│   │
│   └── parser/
│       └── parser.go            # YAML parser
│                                # - ParseMultiDocYAML: Parse multiple documents
│                                # - ExtractKey: Extract identifier
│                                # - CompareValues: Compare values
│
├── scripts/
│   ├── ci-integration-example.sh          # CI integration example
│   └── ci-integration-with-config.sh      # CI integration with config
│
├── .gitignore                   # Git ignore file
├── go.mod                       # Go module definition
├── go.sum                       # Dependency hashes
├── Makefile                     # Build task definitions
├── README.md                    # Project description
├── INSTALL.md                   # Installation guide
├── CONFIG_GUIDE.md              # Configuration guide
├── GITHUB_LABEL.md              # GitHub label integration guide
├── STRUCTURE.md                 # This file
├── QUICKSTART.md                # Quick start guide
├── yamlcmt.yaml.example        # Example configuration file
└── yamlcmt                     # Built binary
```

## Package Dependencies

```
cmd/yamlcmt (main)
    ↓
    ├─→ internal/config
    │       ↓
    │       └─→ gopkg.in/yaml.v3
    │
    ├─→ internal/diff
    │       ↓
    │       └─→ internal/parser
    │               ↓
    │               └─→ gopkg.in/yaml.v3
    │
    ├─→ internal/git
    │       ↓
    │       └─→ os/exec (Git commands)
    │
    ├─→ internal/github
    │       ↓
    │       ├─→ internal/diff
    │       └─→ text/template
    │
    ├─→ internal/parser
    │       ↓
    │       └─→ gopkg.in/yaml.v3
    │
    └─→ github.com/alecthomas/kong
        github.com/fatih/color
```

## Main Functionality Flow

```
1. User Input
   └─→ kong parses CLI arguments
       ├─→ Normal mode: file1.yaml file2.yaml
       └─→ Git mode: --git-compare=<branch> [file]

2. Git Integration (if --git-compare specified)
   └─→ git.GetChangedYAMLFiles()
       ├─→ Execute: git diff --name-only <branch>
       ├─→ Filter for .yaml and .yml files
       └─→ Exclude deleted files
   └─→ git.ParseFilesWithSourceTracking()
       ├─→ For each changed file:
       │   ├─→ Get version from branch (git show)
       │   │   └─→ Apply cleanYAMLContent()
       │   ├─→ Get current version
       │   │   └─→ Apply cleanYAMLContent()
       │   └─→ Parse with source file tracking
       └─→ Return documents with SourceFile set

3. File Reading (normal mode)
   └─→ parser.ParseMultiDocYAML()
       └─→ Convert each document to Document struct

4. Diff Calculation
   └─→ diff.Engine.Compare()
       ├─→ Map documents by identifier
       │   └─→ If SourceFile exists, append to key for uniqueness
       ├─→ Detect added documents
       ├─→ Detect deleted documents
       └─→ Detect modified documents
           └─→ parser.CompareValues() for detailed diff

5. GitHub Integration (if configured)
   ├─→ Load config file (config.LoadConfig)
   │
   ├─→ Prepare template data (github.PrepareTemplateData)
   │   ├─→ Extract added/deleted/modified lists
   │   ├─→ Format summary as "Plan: X to add, Y to delete, Z to modify"
   │   └─→ Include custom variables
   │
   ├─→ Render template (github.RenderTemplate)
   │   └─→ Apply Go template with data
   │
   ├─→ Post comment (github.PostComment)
   │   └─→ Execute gh CLI command
   │
   └─→ Add labels (github.AddLabels)
       └─→ Execute gh CLI command for each label

6. Result Output
   └─→ diff.Result.Print() or PrintSummary()
       ├─→ Non-verbose: Show keys and diffs only
       └─→ Verbose: Show full document content
           └─→ PrintSummaryCompact(): "%d added, %d deleted, %d modified"

7. Cleanup (if Git mode with legacy method)
   └─→ Remove temporary files
```

## Configuration System

```
yamlcmt.yaml
    ↓
config.LoadConfig()
    ↓
    ├─→ Parse YAML structure
    ├─→ Load template string
    ├─→ Load label configurations
    │   ├─→ when_has_additions
    │   ├─→ when_has_deletions
    │   ├─→ when_has_modifications
    │   └─→ when_no_changes
    └─→ Load flags (disable_comment, disable_label)

During execution:
    ↓
config.GetLabels(added, deleted, modified)
    ↓
    ├─→ If no changes: return when_no_changes label
    └─→ If changes exist: return cumulative labels
        ├─→ added > 0 → when_has_additions
        ├─→ deleted > 0 → when_has_deletions
        └─→ modified > 0 → when_has_modifications
```

## Git Integration System (Updated)

```
--git-compare=<branch> option
    ↓
git.GetChangedYAMLFiles(branch)
    ↓
    ├─→ Execute: git diff --name-only <branch>
    ├─→ Parse output line by line
    ├─→ Filter for .yaml and .yml extensions
    ├─→ Check if file exists (exclude deleted)
    └─→ Return list of changed YAML files

If files found:
    ↓
git.ParseFilesWithSourceTracking(branch, files)
    ↓
    For each file:
    ├─→ Execute: git show <branch>:<file>
    │   ├─→ Success: Apply cleanYAMLContent()
    │   │   └─→ Parse documents with yaml.Decoder
    │   │       └─→ Set SourceFile to track origin
    │   └─→ Fail (new file): Skip old version
    │
    ├─→ Read current file: Apply cleanYAMLContent()
    │   └─→ Parse documents with yaml.Decoder
    │       └─→ Set SourceFile to track origin
    │
    └─→ Return documents with source tracking

cleanYAMLContent() function:
    ├─→ Remove leading comments
    ├─→ Remove leading empty lines
    ├─→ Remove leading document separators (---)
    └─→ Return cleaned content
    Purpose: Prevent empty documents (__index__) in diff

Usage Examples:
    ├─→ yamlcmt --git-compare=main
    │   └─→ Auto-detect all changed YAML files
    │
    ├─→ yamlcmt --git-compare=main config.yaml
    │   └─→ Compare specific file only
    │
    └─→ yamlcmt --git-compare=main --github-pr=123
        └─→ Compare + post to GitHub PR
```

## Source File Tracking

When using `--git-compare` with multiple files:

```
Problem: Multiple files may have resources with same name
Example:
  - service1/config.yaml has Resource "my-app"
  - service2/config.yaml has Resource "my-app"

Solution: SourceFile tracking
    ↓
diff.Engine.makeDocMap()
    ├─→ Extract key (e.g., "my-app")
    ├─→ Check if SourceFile is set
    └─→ If yes: Append source to key
        Result: "my-app (from service1/config.yaml)"

This ensures:
    ✓ Resources with same name in different files are tracked separately
    ✓ Diffs show which file each resource belongs to
    ✓ No false "modified" detections for different resources
```

## Template System

```
Template Definition (in yamlcmt.yaml)
    ↓
Available Variables:
    ├─→ .Summary        (e.g., "Plan: 1 to add, 2 to delete, 0 to modify")
    ├─→ .Details        (verbose diff output)
    ├─→ .HasChanges     (true/false)
    ├─→ .Added          (number of added documents)
    ├─→ .Deleted        (number of deleted documents)
    ├─→ .Modified       (number of modified documents)
    ├─→ .AddedList      ([]string of added document names)
    ├─→ .DeletedList    ([]string of deleted document names)
    ├─→ .ModifiedList   ([]string of modified document names)
    ├─→ .Link           (CI build link, optional)
    └─→ .Vars           (custom variables, map[string]interface{})

Note: .Summary uses "Plan:" format for consistency with tfcmt
      Command-line verbose output uses "%d added, %d deleted, %d modified" format

Rendering Process:
    ↓
github.RenderTemplate(templateStr, data)
    ↓
text/template execution
    ↓
Formatted Markdown output
    ↓
Posted to GitHub PR as comment
```

## Similarities with expiry-monitor

This project adopts the same design pattern as `expiry-monitor`:

### Common Structure
- `cmd/<tool>/main.go`: CLI entry point
- `internal/`: Internal packages (not importable from outside)
- `github.com/alecthomas/kong`: CLI framework
- Subcommand-based design

### expiry-monitor structure (reference)
```
expiry-monitor/
├── cmd/
│   └── expiry-monitor/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   └── datadog/
│       ├── client.go
│       ├── metrics.go
│       └── monitor.go
└── ...
```

### yamlcmt structure (this project)
```
yamlcmt/
├── cmd/
│   └── yamlcmt/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── diff/
│   │   └── diff.go
│   ├── git/
│   │   └── git.go
│   ├── github/
│   │   └── github.go
│   └── parser/
│       └── parser.go
└── ...
```

## Extensibility

Examples of subcommands that can be added in the future:

```go
type CLI struct {
    Compare  CompareCmd  `cmd:"" help:"Compare two YAML files."`
    Validate ValidateCmd `cmd:"" help:"Validate YAML syntax."`
    Format   FormatCmd   `cmd:"" help:"Format YAML files."`
    Merge    MergeCmd    `cmd:"" help:"Merge multiple YAML files."`
}
```

This design allows easy addition of new features.

## Color Output System

```
internal/diff/diff.go
    ↓
Uses github.com/fatih/color
    ↓
    ├─→ Red: Deleted documents/lines
    ├─→ Green: Added documents/lines
    ├─→ Yellow: Modified documents
    └─→ Cyan: Document identifiers

Can be disabled with:
    ├─→ --no-color flag
    └─→ color.NoColor = true (automatic in CI environments)
```

## Error Handling

```
Main execution flow:
    ↓
    ├─→ File parsing errors
    │   └─→ Return fmt.Errorf with context
    │
    ├─→ Config loading errors
    │   └─→ Return fmt.Errorf with context
    │
    ├─→ GitHub API errors
    │   ├─→ PostComment failure
    │   │   └─→ Return error with gh output
    │   └─→ AddLabels failure
    │       └─→ Return error with gh output
    │
    └─→ Exit codes
        ├─→ 0: No differences or successful execution
        └─→ 1: Differences found (expected behavior)
```

## Development Workflow

```
1. Make changes to code
   └─→ Edit files in cmd/ or internal/

2. Install dependencies (if needed)
   └─→ make dev-deps

3. Build
   └─→ make build

4. Test
   └─→ ./yamlcmt testdata/old.yaml testdata/new.yaml
   └─→ Test with actual YAML files

5. Install (optional)
   └─→ make install
   └─→ Installs to $GOPATH/bin
```

## Summary Format Differences

There are two different summary formats used in yamlcmt:

1. **Command-line verbose output** (`-v` flag):
   ```
   Summary
   0 added, 2 deleted, 0 modified
   ```
   - Format: `%d added, %d deleted, %d modified`
   - Used by: `diff.PrintSummaryCompact()`
   - Purpose: Compact display for terminal output

2. **Template variable** (`.Summary` in config):
   ```
   Plan: 0 to add, 2 to delete, 0 to modify
   ```
   - Format: `Plan: %d to add, %d to delete, %d to modify`
   - Used by: `github.PrepareTemplateData()`
   - Purpose: Consistency with tfcmt-style configuration

This design provides optimal readability in both contexts:
- Terminal output is concise
- PR comments use familiar tfcmt-style format
