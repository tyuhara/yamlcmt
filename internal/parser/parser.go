package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document represents a single YAML document
type Document struct {
	Content    map[string]interface{}
	Raw        string
	Key        string
	SourceFile string // Source file path for tracking across multiple files
}

// ParseMultiDocYAML parses a YAML file that may contain multiple documents
func ParseMultiDocYAML(filename string) ([]Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var docs []Document

	for {
		var doc map[string]interface{}
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Marshal back to YAML for display
		raw, err := yaml.Marshal(doc)
		if err != nil {
			return nil, err
		}

		docs = append(docs, Document{
			Content: doc,
			Raw:     string(raw),
		})
	}

	return docs, nil
}

// ExtractKey extracts a value from a document using a dot-notation path
func ExtractKey(data map[string]interface{}, path string) string {
	keys := splitPath(path)

	current := interface{}(data)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}
	return ""
}

func splitPath(path string) []string {
	var result []string
	var current string

	for _, char := range path {
		if char == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// Indent adds a prefix to each line of text
func Indent(text string, prefix string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if len(line) > 0 {
			result = append(result, prefix+line)
		} else {
			result = append(result, "")
		}
	}

	return strings.Join(result, "\n")
}

// SplitLines splits text into lines, removing trailing empty lines
func SplitLines(text string) []string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	return lines
}

// CompareValues recursively compares two values and returns a formatted diff
func CompareValues(path string, oldVal, newVal interface{}) []string {
	var diffs []string

	oldMap, oldIsMap := oldVal.(map[string]interface{})
	newMap, newIsMap := newVal.(map[string]interface{})

	if oldIsMap && newIsMap {
		// Both are maps - recurse
		allKeys := make(map[string]bool)
		for k := range oldMap {
			allKeys[k] = true
		}
		for k := range newMap {
			allKeys[k] = true
		}

		for key := range allKeys {
			newPath := path + "." + key
			if path == "" {
				newPath = key
			}

			oldV, oldExists := oldMap[key]
			newV, newExists := newMap[key]

			if !oldExists && newExists {
				diffs = append(diffs, fmt.Sprintf("+ %s: %v", newPath, newV))
			} else if oldExists && !newExists {
				diffs = append(diffs, fmt.Sprintf("- %s: %v", newPath, oldV))
			} else if oldExists && newExists {
				subDiffs := CompareValues(newPath, oldV, newV)
				diffs = append(diffs, subDiffs...)
			}
		}
	} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
		diffs = append(diffs, fmt.Sprintf("~ %s: %v → %v", path, oldVal, newVal))
	}

	return diffs
}
