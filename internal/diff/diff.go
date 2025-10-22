package diff

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/tyuhara/yamlcmt/internal/parser"
)

// Engine handles the comparison of YAML documents
type Engine struct {
	identifierPath string
}

// Result represents the result of a comparison
type Result struct {
	Added    map[string]parser.Document
	Deleted  map[string]parser.Document
	Modified map[string]ModifiedDoc
}

// ModifiedDoc represents a modified document with its changes
type ModifiedDoc struct {
	Old    parser.Document
	New    parser.Document
	Diffs  []string
}

// NewEngine creates a new diff engine with the specified identifier path
func NewEngine(identifierPath string) *Engine {
	return &Engine{
		identifierPath: identifierPath,
	}
}

// Compare compares two sets of documents
func (e *Engine) Compare(docs1, docs2 []parser.Document) *Result {
	map1 := e.makeDocMap(docs1)
	map2 := e.makeDocMap(docs2)

	result := &Result{
		Added:    make(map[string]parser.Document),
		Deleted:  make(map[string]parser.Document),
		Modified: make(map[string]ModifiedDoc),
	}

	// Find all unique keys
	allKeys := make(map[string]bool)
	for k := range map1 {
		allKeys[k] = true
	}
	for k := range map2 {
		allKeys[k] = true
	}

	// Compare documents
	for key := range allKeys {
		doc1, exists1 := map1[key]
		doc2, exists2 := map2[key]

		if !exists1 && exists2 {
			// Added
			result.Added[key] = doc2
		} else if exists1 && !exists2 {
			// Deleted
			result.Deleted[key] = doc1
		} else if doc1.Raw != doc2.Raw {
			// Modified
			diffs := parser.CompareValues("", doc1.Content, doc2.Content)
			result.Modified[key] = ModifiedDoc{
				Old:   doc1,
				New:   doc2,
				Diffs: diffs,
			}
		}
	}

	return result
}

func (e *Engine) makeDocMap(docs []parser.Document) map[string]parser.Document {
	result := make(map[string]parser.Document)

	for i, doc := range docs {
		key := parser.ExtractKey(doc.Content, e.identifierPath)

		// Check for SourceFile to handle duplicate names across files
		if doc.SourceFile != "" {
		 // Append source file to key to make it unique
		key = key + " (from " + doc.SourceFile + ")"
		}

		if key == "" {
			// Fallback to index if no identifier found
			key = fmt.Sprintf("__index_%d__", i)
		}
		doc.Key = key
		result[key] = doc
	}

	return result
}

// HasDifferences returns true if there are any differences
func (r *Result) HasDifferences() bool {
	return len(r.Added) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}

// Print prints the diff result
func (r *Result) Print(verbose bool) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	if !verbose {
		// Non-verbose: show key names only
		// Print added documents
		keys := sortedKeys(r.Added)
		for _, key := range keys {
			fmt.Printf("%s %s\n", green("+ Added:"), cyan(key))
		}

		// Print deleted documents
		keys = sortedKeys(r.Deleted)
		for _, key := range keys {
			fmt.Printf("%s %s\n", red("- Deleted:"), cyan(key))
		}

		// Print modified documents
		keys = sortedKeysModified(r.Modified)
		for _, key := range keys {
			mod := r.Modified[key]
			fmt.Printf("%s %s\n", yellow("~ Modified:"), cyan(key))
			for _, diff := range mod.Diffs {
				fmt.Printf("  %s\n", diff)
			}
			fmt.Println()
		}

		// Print summary
		r.PrintSummary()
	} else {
		// Verbose: show summary first, then full document content with diff-style prefixes
		r.PrintSummaryCompact()

		// Print added documents with "+" prefix
		keys := sortedKeys(r.Added)
		for _, key := range keys {
			doc := r.Added[key]
			lines := parser.SplitLines(doc.Raw)
			for _, line := range lines {
				if len(line) > 0 {
					fmt.Printf("%s\n", green("+ "+line))
				}
			}
		}

		// Print deleted documents with "-" prefix
		keys = sortedKeys(r.Deleted)
		for _, key := range keys {
			doc := r.Deleted[key]
			lines := parser.SplitLines(doc.Raw)
			for _, line := range lines {
				if len(line) > 0 {
					fmt.Printf("%s\n", red("- "+line))
				}
			}
		}

		// Print modified documents
		keys = sortedKeysModified(r.Modified)
		for _, key := range keys {
			mod := r.Modified[key]
			fmt.Printf("%s %s\n", yellow("~ Modified:"), cyan(key))
			for _, diff := range mod.Diffs {
				fmt.Printf("  %s\n", diff)
			}
			fmt.Println()
		}
	}
}

// PrintSummary prints a summary of changes
func (r *Result) PrintSummary() {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Printf("\n%s\n", bold("Summary:"))
	fmt.Printf("  %s: %d\n", green("Added"), len(r.Added))
	fmt.Printf("  %s: %d\n", red("Deleted"), len(r.Deleted))
	fmt.Printf("  %s: %d\n", yellow("Modified"), len(r.Modified))
}

// PrintSummaryCompact prints a compact summary suitable for verbose output
func (r *Result) PrintSummaryCompact() {
	fmt.Printf("Summary\n")
	fmt.Printf("%d added, %d deleted, %d modified\n", len(r.Added), len(r.Deleted), len(r.Modified))
}

func sortedKeys(m map[string]parser.Document) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysModified(m map[string]ModifiedDoc) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
