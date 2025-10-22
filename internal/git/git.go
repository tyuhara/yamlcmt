package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	
	"github.com/tyuhara/yamlcmt/internal/parser"
	"gopkg.in/yaml.v3"
)

// GetChangedYAMLFiles returns all changed YAML files compared to the specified branch.
// It uses `git diff --name-only` to detect changes and filters for .yaml and .yml files.
// Deleted files are excluded from the results.
func GetChangedYAMLFiles(branch string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", branch)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to get changed files from git: %w\nStderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to get changed files from git: %w", err)
	}

	var yamlFiles []string
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		filename := string(bytes.TrimSpace(line))
		if filename == "" {
			continue
		}

		// Check if it's a YAML file
		if bytes.HasSuffix(line, []byte(".yaml")) || bytes.HasSuffix(line, []byte(".yml")) {
			// Check if file exists (not deleted)
			if _, err := os.Stat(filename); err == nil {
				yamlFiles = append(yamlFiles, filename)
			}
		}
	}

	return yamlFiles, nil
}

// CombineFilesForComparison combines multiple YAML files from Git and current state for comparison.
// It creates two temporary files:
// - oldPath: Contains files from the specified branch
// - newPath: Contains current versions of the files
// Files are separated by "---" (YAML document separator).
// Empty documents and comment-only sections are skipped to avoid __index__ artifacts.
// The cleanup function should be called to remove temporary files after use.
func CombineFilesForComparison(branch string, files []string) (oldPath, newPath string, cleanup func(), err error) {
	// Create temporary files
	tmpOld, err := os.CreateTemp("", "yamlcmt-old-*.yaml")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpOldPath := tmpOld.Name()

	tmpNew, err := os.CreateTemp("", "yamlcmt-new-*.yaml")
	if err != nil {
		tmpOld.Close()
		os.Remove(tmpOldPath)
		return "", "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpNewPath := tmpNew.Name()

	// Combine files
	for i, file := range files {
		fmt.Fprintf(os.Stderr, "Processing: %s\n", file)

		// Get old version from Git
		cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", branch, file))
		output, err := cmd.Output()
		if err == nil {
			// File exists in the branch - clean the content
			cleanedOld := cleanYAMLContent(output)
			if len(cleanedOld) > 0 {
				// Add separator if not the first file
				if i > 0 && tmpOld != nil {
					tmpOld.WriteString("---\n")
				}
				if _, err := tmpOld.Write(cleanedOld); err != nil {
					tmpOld.Close()
					tmpNew.Close()
					os.Remove(tmpOldPath)
					os.Remove(tmpNewPath)
					return "", "", nil, fmt.Errorf("failed to write old file content: %w", err)
				}
				// Ensure the file ends with a newline
				if cleanedOld[len(cleanedOld)-1] != '\n' {
					tmpOld.WriteString("\n")
				}
			}
		} else {
			// New file - skip adding to old
			fmt.Fprintf(os.Stderr, "  (new file)\n")
		}

		// Get current version
		content, err := os.ReadFile(file)
		if err != nil {
			tmpOld.Close()
			tmpNew.Close()
			os.Remove(tmpOldPath)
			os.Remove(tmpNewPath)
			return "", "", nil, fmt.Errorf("failed to read %s: %w", file, err)
		}
		
		// Clean the content
		cleanedNew := cleanYAMLContent(content)
		if len(cleanedNew) > 0 {
			// Add separator if not the first file
			if i > 0 && tmpNew != nil {
				tmpNew.WriteString("---\n")
			}
			if _, err := tmpNew.Write(cleanedNew); err != nil {
				tmpOld.Close()
				tmpNew.Close()
				os.Remove(tmpOldPath)
				os.Remove(tmpNewPath)
				return "", "", nil, fmt.Errorf("failed to write new file content: %w", err)
			}
			// Ensure the file ends with a newline
			if cleanedNew[len(cleanedNew)-1] != '\n' {
				tmpNew.WriteString("\n")
			}
		}
	}

	tmpOld.Close()
	tmpNew.Close()

	cleanup = func() {
		os.Remove(tmpOldPath)
		os.Remove(tmpNewPath)
	}

	return tmpOldPath, tmpNewPath, cleanup, nil
}

// cleanYAMLContent removes leading comments and empty document separators from YAML content.
// This prevents empty documents (which would be identified as __index__) from being included in the diff.
func cleanYAMLContent(content []byte) []byte {
	lines := bytes.Split(content, []byte("\n"))
	var result [][]byte
	skippingLeadingContent := true

	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		
		if skippingLeadingContent {
			// Skip leading comments
			if bytes.HasPrefix(trimmed, []byte("#")) {
				continue
			}
			// Skip leading empty lines
			if len(trimmed) == 0 {
				continue
			}
			// Skip leading document separators
			if bytes.Equal(trimmed, []byte("---")) {
				continue
			}
			// Found real content, stop skipping
			skippingLeadingContent = false
		}
		
		// Add all content after we've found the first real line
		result = append(result, line)
	}
	
	if len(result) == 0 {
		return []byte{}
	}
	
	return bytes.Join(result, []byte("\n"))
}

// ParseFilesWithSourceTracking parses multiple YAML files and tracks their source file paths.
// This is used for --git-compare to maintain file source information for duplicate resource names.
func ParseFilesWithSourceTracking(branch string, files []string) (oldDocs, newDocs []parser.Document, err error) {
	for _, file := range files {
		fmt.Fprintf(os.Stderr, "Processing: %s\n", file)
		
		// Get old version from Git
		cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", branch, file))
		output, gitErr := cmd.Output()
		if gitErr == nil {
			// Parse old version
			cleanedOld := cleanYAMLContent(output)
			decoder := yaml.NewDecoder(bytes.NewReader(cleanedOld))
			for {
				var doc map[string]interface{}
				if err := decoder.Decode(&doc); err != nil {
					if err == io.EOF {
						break
					}
					return nil, nil, fmt.Errorf("failed to parse old version of %s: %w", file, err)
				}
				raw, _ := yaml.Marshal(doc)
				oldDocs = append(oldDocs, parser.Document{
					Content:    doc,
					Raw:        string(raw),
					SourceFile: file,
				})
			}
		} else {
			fmt.Fprintf(os.Stderr, "  (new file)\n")
		}
		
		// Get current version
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read %s: %w", file, err)
		}
		
		cleanedNew := cleanYAMLContent(content)
		decoder := yaml.NewDecoder(bytes.NewReader(cleanedNew))
		for {
			var doc map[string]interface{}
			if err := decoder.Decode(&doc); err != nil {
				if err == io.EOF {
					break
				}
				return nil, nil, fmt.Errorf("failed to parse new version of %s: %w", file, err)
			}
			raw, _ := yaml.Marshal(doc)
			newDocs = append(newDocs, parser.Document{
				Content:    doc,
				Raw:        string(raw),
				SourceFile: file,
			})
		}
	}
	
	return oldDocs, newDocs, nil
}

// IsGitRepository checks if the current directory is inside a Git repository.
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// BranchExists checks if a branch exists in the repository.
func BranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	err := cmd.Run()
	return err == nil
}
