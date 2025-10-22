package github

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"text/template"

	"github.com/tyuhara/yamlcmt/internal/diff"
)

// TemplateData represents data available in templates
type TemplateData struct {
	Summary      string
	Details      string
	HasChanges   bool
	Added        int
	Deleted      int
	Modified     int
	AddedList    []string
	DeletedList  []string
	ModifiedList []string
	Link         string
	Vars         map[string]interface{}
}

// PostComment posts a comment to a GitHub PR
func PostComment(repo string, prNumber int, body string) error {
	cmd := exec.Command("gh", "pr", "comment", fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--body", body)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to post comment: %w\nOutput: %s", err, string(output))
	}

	fmt.Fprintf(os.Stderr, "✓ Posted GitHub comment\n")
	return nil
}

// AddLabel adds a label to a GitHub PR
func AddLabel(repo string, prNumber int, label string) error {
	if label == "" {
		return nil
	}

	cmd := exec.Command("gh", "pr", "edit", fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--add-label", label)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add label: %w\nOutput: %s", err, string(output))
	}

	fmt.Fprintf(os.Stderr, "✓ Applied GitHub label: %s\n", label)
	return nil
}

// AddLabels adds multiple labels to a GitHub PR
func AddLabels(repo string, prNumber int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}

	for _, label := range labels {
		if label == "" {
			continue
		}

		if err := AddLabel(repo, prNumber, label); err != nil {
			return err
		}
	}

	return nil
}

// RenderTemplate renders a template with the given data
func RenderTemplate(tmplStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("comment").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// PrepareTemplateData prepares template data from diff result
func PrepareTemplateData(result *diff.Result, details string, link string, vars map[string]interface{}) TemplateData {
	added := len(result.Added)
	deleted := len(result.Deleted)
	modified := len(result.Modified)

	summary := fmt.Sprintf("Plan: %d to add, %d to delete, %d to modify", added, deleted, modified)

	// Extract and sort keys
	addedList := make([]string, 0, len(result.Added))
	for k := range result.Added {
		addedList = append(addedList, k)
	}
	sort.Strings(addedList)

	deletedList := make([]string, 0, len(result.Deleted))
	for k := range result.Deleted {
		deletedList = append(deletedList, k)
	}
	sort.Strings(deletedList)

	modifiedList := make([]string, 0, len(result.Modified))
	for k := range result.Modified {
		modifiedList = append(modifiedList, k)
	}
	sort.Strings(modifiedList)

	return TemplateData{
		Summary:      summary,
		Details:      details,
		HasChanges:   result.HasDifferences(),
		Added:        added,
		Deleted:      deleted,
		Modified:     modified,
		AddedList:    addedList,
		DeletedList:  deletedList,
		ModifiedList: modifiedList,
		Link:         link,
		Vars:         vars,
	}
}
